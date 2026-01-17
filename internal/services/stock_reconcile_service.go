package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/araquach/phorest-datahub/internal/repos"
)

type StockAdjuster interface {
	AdjustStock(ctx context.Context, branchID string, req StockAdjustmentRequest) error
}

type StockAdjustmentItem struct {
	Barcode       string `json:"barcode"`
	Quantity      int    `json:"quantity"`
	OperationType string `json:"operationType"` // "DEDUCT" or "INCREASE"
}

type StockAdjustmentRequest struct {
	Stocks []StockAdjustmentItem `json:"stocks"`
}

type BranchPayload struct {
	BranchID string
	Req      StockAdjustmentRequest
}

type StockReconcileService struct {
	Repo repos.StockReconcileRepo

	Adjuster StockAdjuster

	Logger *log.Logger

	// Config
	PKBranchID string
	DryRun     bool // this service currently only logs, but keep the flag for future live mode

	// Run limits
	FromTS time.Time
	ToTS   time.Time
	Limit  int

	// Optional test filter
	TestBarcode string // if set, only process this barcode

	// Logging controls
	MaxPreview int  // how many stock lines to preview per payload
	PrintJSON  bool // print full JSON payloads
}

func (s StockReconcileService) lg() *log.Logger {
	if s.Logger != nil {
		return s.Logger
	}
	return log.Default()
}

func (s StockReconcileService) Run(ctx context.Context) error {
	if s.PKBranchID == "" {
		return fmt.Errorf("PKBranchID is required")
	}
	if s.Limit <= 0 {
		s.Limit = 500
	}
	if s.MaxPreview <= 0 {
		s.MaxPreview = 20
	}
	if s.FromTS.IsZero() {
		s.FromTS = time.Now().AddDate(0, 0, -30)
	}
	if s.ToTS.IsZero() {
		s.ToTS = time.Now()
	}

	totalRows := 0
	totalMapped := 0
	totalUnmapped := 0
	totalTransfers := 0
	totalExceptions := 0
	batches := 0

	for {
		rows, err := s.Repo.FetchUnprocessedPKItems(ctx, s.PKBranchID, s.FromTS, s.ToTS, s.Limit, s.TestBarcode)
		if err != nil {
			return fmt.Errorf("fetch pk items: %w", err)
		}

		if len(rows) == 0 {
			if batches == 0 {
				s.lg().Printf("[stockrecon] dry-run=%v: no rows to process", s.DryRun)
			} else {
				s.lg().Printf("[stockrecon] done batches=%d rows=%d mapped=%d unmapped=%d transfers=%d exceptions=%d",
					batches, totalRows, totalMapped, totalUnmapped, totalTransfers, totalExceptions)
			}
			return nil
		}

		batches++
		totalRows += len(rows)

		// Split mapped vs unmapped
		var mapped []repos.PKStockRow
		var unmapped []repos.PKStockRow

		for _, r := range rows {
			if !r.PhysicalBranchID.Valid || r.PhysicalBranchID.String == "" {
				unmapped = append(unmapped, r)
				continue
			}
			mapped = append(mapped, r)
		}

		totalMapped += len(mapped)
		totalUnmapped += len(unmapped)

		// ---- RECORD EXCEPTIONS (unmapped staff) ----
		if len(unmapped) > 0 {
			s.lg().Printf("[stockrecon] %d unmapped rows -> recording exceptions", len(unmapped))

			if err := s.Repo.InsertStockVirtualTransferExceptions(
				ctx,
				unmapped,
				"UNMAPPED_STAFF",
				nil, // productNameByBarcode
				nil, // staffNameByID
			); err != nil {
				return fmt.Errorf("insert exceptions failed: %w", err)
			}
			totalExceptions += len(unmapped)
		}

		// Aggregate (mapped only)
		deductAgg := make(map[string]map[string]int) // physical_branch_id -> barcode -> qty
		increaseAgg := make(map[string]int)          // barcode -> qty

		for _, r := range mapped {
			branch := r.PhysicalBranchID.String
			if _, ok := deductAgg[branch]; !ok {
				deductAgg[branch] = make(map[string]int)
			}
			deductAgg[branch][r.Barcode] += r.Quantity
			increaseAgg[r.Barcode] += r.Quantity
		}

		// Build payloads
		deductPayloads := make([]BranchPayload, 0, len(deductAgg))
		for branchID, byBarcode := range deductAgg {
			deductPayloads = append(deductPayloads, BranchPayload{
				BranchID: branchID,
				Req:      buildRequest(byBarcode, "DEDUCT"),
			})
		}
		sort.Slice(deductPayloads, func(i, j int) bool { return deductPayloads[i].BranchID < deductPayloads[j].BranchID })

		pkIncrease := BranchPayload{
			BranchID: s.PKBranchID,
			Req:      buildRequest(increaseAgg, "INCREASE"),
		}

		// ---- Logging ----
		s.lg().Printf("[stockrecon] batch=%d dry-run=%v window=[%s .. %s) limit=%d rows=%d mapped=%d unmapped=%d",
			batches,
			s.DryRun,
			s.FromTS.Format(time.RFC3339),
			s.ToTS.Format(time.RFC3339),
			s.Limit,
			len(rows),
			len(mapped),
			len(unmapped),
		)

		// Unmapped preview
		if len(unmapped) > 0 {
			n := min(s.MaxPreview, len(unmapped))
			s.lg().Printf("[stockrecon] unmapped staff override missing (showing %d/%d):", n, len(unmapped))
			for i := 0; i < n; i++ {
				u := unmapped[i]
				s.lg().Printf("  - item_id=%s barcode=%s qty=%d staff_id=%s updated_at_phorest=%s purchased_at=%s",
					u.TransactionItemID,
					u.Barcode,
					u.Quantity,
					u.StaffID,
					u.UpdatedAtPhorest.Format(time.RFC3339),
					formatNullTime(u.PurchasedAt),
				)
			}
		}

		// Deduct payload previews
		for _, p := range deductPayloads {
			lines, total := payloadStats(p.Req)
			s.lg().Printf("[stockrecon] would POST DEDUCT branch=%s lines=%d total_qty=%d", p.BranchID, lines, total)
			printPreview(s.lg(), p.Req, s.MaxPreview)
			if s.PrintJSON {
				printJSON(s.lg(), fmt.Sprintf("DEDUCT branch=%s", p.BranchID), p.Req)
			}
		}

		// Increase payload preview
		lines, total := payloadStats(pkIncrease.Req)
		s.lg().Printf("[stockrecon] would POST INCREASE branch=%s lines=%d total_qty=%d", pkIncrease.BranchID, lines, total)
		printPreview(s.lg(), pkIncrease.Req, s.MaxPreview)
		if s.PrintJSON {
			printJSON(s.lg(), "INCREASE PK", pkIncrease.Req)
		}

		// ---- STOP HERE IN DRY RUN ----
		if s.DryRun {
			// in dry-run we still loop so you can see *everything* it would process
			// but we must prevent infinite loops: mark nothing, so we'd keep seeing same rows.
			// Therefore: break after one batch in dry-run.
			s.lg().Printf("[stockrecon] dry-run=true: stopping after 1 batch (no DB marks written)")
			return nil
		}

		// ---- LIVE MODE GUARDS ----
		if s.Adjuster == nil {
			return fmt.Errorf("refusing LIVE run: Adjuster is nil")
		}
		if len(unmapped) > 0 {
			s.lg().Printf("[stockrecon] LIVE: continuing with mapped rows; %d rows were exceptioned", len(unmapped))
		}

		// ---- LIVE: POST DEDUCT payloads ----
		for _, p := range deductPayloads {
			if len(p.Req.Stocks) == 0 {
				continue
			}
			s.lg().Printf("[stockrecon] LIVE: POST DEDUCT branch=%s lines=%d", p.BranchID, len(p.Req.Stocks))
			if err := s.Adjuster.AdjustStock(ctx, p.BranchID, p.Req); err != nil {
				return fmt.Errorf("live deduct failed branch=%s: %w", p.BranchID, err)
			}
		}

		// ---- LIVE: POST INCREASE to PK ----
		if len(pkIncrease.Req.Stocks) > 0 {
			s.lg().Printf("[stockrecon] LIVE: POST INCREASE branch=%s lines=%d", pkIncrease.BranchID, len(pkIncrease.Req.Stocks))
			if err := s.Adjuster.AdjustStock(ctx, pkIncrease.BranchID, pkIncrease.Req); err != nil {
				return fmt.Errorf("live increase failed branch=%s: %w", pkIncrease.BranchID, err)
			}
		}

		// ---- LIVE: record processed items (mapped only) ----
		transferRows := make([]repos.StockVirtualTransferRow, 0, len(mapped))
		for _, r := range mapped {
			transferRows = append(transferRows, repos.StockVirtualTransferRow{
				TransactionItemID: r.TransactionItemID,
				FromBranchID:      r.PhysicalBranchID.String,
				ToBranchID:        s.PKBranchID,
				Barcode:           r.Barcode,
				Quantity:          r.Quantity,
			})
		}

		if err := s.Repo.InsertStockVirtualTransfers(ctx, transferRows); err != nil {
			return fmt.Errorf("insert stock_virtual_transfers failed: %w", err)
		}

		totalTransfers += len(transferRows)

		s.lg().Printf("[stockrecon] LIVE batch=%d complete: recorded %d transfers", batches, len(transferRows))

		// loop continues: next FetchUnprocessedPKItems will exclude transfers + exceptions
	}
}

func buildRequest(agg map[string]int, op string) StockAdjustmentRequest {
	barcodes := make([]string, 0, len(agg))
	for bc := range agg {
		barcodes = append(barcodes, bc)
	}
	sort.Strings(barcodes)

	items := make([]StockAdjustmentItem, 0, len(barcodes))
	for _, bc := range barcodes {
		qty := agg[bc]
		if qty <= 0 {
			continue
		}
		items = append(items, StockAdjustmentItem{
			Barcode:       bc,
			Quantity:      qty,
			OperationType: op,
		})
	}

	return StockAdjustmentRequest{Stocks: items}
}

func payloadStats(req StockAdjustmentRequest) (lines int, totalQty int) {
	lines = len(req.Stocks)
	for _, it := range req.Stocks {
		totalQty += it.Quantity
	}
	return lines, totalQty
}

func printPreview(lg *log.Logger, req StockAdjustmentRequest, max int) {
	if lg == nil {
		lg = log.Default()
	}

	n := min(max, len(req.Stocks))
	for i := 0; i < n; i++ {
		it := req.Stocks[i]
		lg.Printf("  - %s %s x%d", it.OperationType, it.Barcode, it.Quantity)
	}
	if len(req.Stocks) > n {
		lg.Printf("  ... and %d more", len(req.Stocks)-n)
	}
}

func printJSON(lg *log.Logger, label string, req StockAdjustmentRequest) {
	if lg == nil {
		lg = log.Default()
	}

	b, err := json.MarshalIndent(req, "", "  ")
	if err != nil {
		lg.Printf("[stockrecon] json marshal failed (%s): %v", label, err)
		return
	}
	lg.Printf("[stockrecon] payload (%s):\n%s", label, string(b))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func formatNullTime(nt sql.NullTime) string {
	if !nt.Valid {
		return "NULL"
	}
	return nt.Time.Format(time.RFC3339)
}
