package phorest

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/araquach/phorest-datahub/internal/models"
	"github.com/araquach/phorest-datahub/internal/repos"
)

func (r *Runner) RunIncrementalClientsAPISync(ctx context.Context) error {
	lg := r.Logger
	db := r.DB

	lg.Printf("▶️ Starting incremental CLIENTS_API sync...")

	c := NewClientsAPIClient(
		r.Cfg.PhorestUsername,
		r.Cfg.PhorestPassword,
		r.Cfg.PhorestBusiness,
	)
	repo := repos.NewClientsAPIRepo(db, lg)
	wr := repos.NewWatermarksRepo(db, lg)

	// business-wide, so branchID = "ALL"
	last, err := wr.GetLastUpdated("clients_api", "ALL")
	if err != nil {
		return fmt.Errorf("get clients_api watermark: %w", err)
	}

	if last != nil {
		lg.Printf("ℹ️ clients_api: existing watermark = %s", last.UTC().Format(time.RFC3339))
	} else {
		lg.Printf("ℹ️ clients_api: no watermark yet (full-ish sweep)")
	}

	const pageSize = 100
	page := 0
	var allNew []models.ClientAPI
	var maxUpdated *time.Time

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		rows, totalPages, err := c.FetchClientsPage(ctx, last, page, pageSize)
		if err != nil {
			return fmt.Errorf("fetch clients page=%d: %w", page, err)
		}
		if len(rows) == 0 {
			lg.Printf("ℹ️ clients_api: no rows on page %d (totalPages=%d), stopping", page, totalPages)
			break
		}

		if err := repo.UpsertBatch(rows, 500); err != nil {
			return fmt.Errorf("upsert clients_api page=%d: %w", page, err)
		}

		allNew = append(allNew, rows...)

		for i := range rows {
			if rows[i].UpdatedAtPhorest != nil {
				if maxUpdated == nil || rows[i].UpdatedAtPhorest.After(*maxUpdated) {
					maxUpdated = rows[i].UpdatedAtPhorest
				}
			}
		}

		page++
		if totalPages > 0 && page >= totalPages {
			lg.Printf("ℹ️ clients_api: reached totalPages=%d", totalPages)
			break
		}
	}

	if len(allNew) == 0 {
		lg.Printf("✅ clients_api: no new/updated clients; nothing to archive")
		return nil
	}

	// 1) write CSV into ExportDir
	ts := time.Now().UTC().Format("20060102_150405")
	filename := fmt.Sprintf("clients_api_incremental_%s.csv", ts)
	tmpPath := filepath.Join(r.Cfg.ExportDir, filename)

	if err := writeClientsAPICSV(tmpPath, allNew); err != nil {
		return fmt.Errorf("write clients_api CSV: %w", err)
	}
	lg.Printf("💾 clients_api: saved CSV to %s", tmpPath)

	// 2) archive into data/clients_api for future bootstrap
	archiveDir := "data/clients_api"
	if err := os.MkdirAll(archiveDir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", archiveDir, err)
	}
	finalPath := filepath.Join(archiveDir, filename)
	if err := os.Rename(tmpPath, finalPath); err != nil {
		return fmt.Errorf("archive clients_api CSV: %w", err)
	}
	lg.Printf("📦 clients_api: archived %s → %s", tmpPath, finalPath)

	// 3) update watermark
	if maxUpdated != nil {
		if err := wr.UpsertLastUpdated("clients_api", "ALL", *maxUpdated); err != nil {
			return fmt.Errorf("update clients_api watermark: %w", err)
		}
		lg.Printf("💾 clients_api: updated watermark → %s", maxUpdated.UTC().Format(time.RFC3339))
	}

	lg.Printf("✅ Incremental CLIENTS_API sync finished (%d rows touched)", len(allNew))
	return nil
}
