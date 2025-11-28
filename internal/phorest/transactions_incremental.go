package phorest

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"staff-appraisals/internal/repos"
)

func (r *Runner) RunIncrementalTransactionsSync(ctx context.Context) error {
	lg := r.Logger
	db := r.DB

	lg.Printf("▶️ Starting incremental TRANSACTIONS_CSV sync...")

	wr := repos.NewWatermarksRepo(db, lg)

	// We'll iterate each branch separately
	for _, b := range r.Cfg.Branches {
		lg.Printf("🏢 Branch %s (%s): starting TRANSACTIONS_CSV sync", b.Name, b.BranchID)

		// 1) Get per-branch watermark
		last, err := wr.GetLastUpdated("transactions_csv", b.BranchID)
		if err != nil {
			return fmt.Errorf("get transactions_csv watermark for %s: %w", b.BranchID, err)
		}

		// 2) Build date window + filterExpression
		var (
			startDate  string
			finishDate string
			filterExpr string
		)

		// Date format used by Phorest for startFilter/finishFilter
		const dateFmt = "2006-01-02"

		if last == nil {
			// No watermark yet for this branch:
			// use some sensible "start of history" date
			startDate = "2000-01-01"
		} else {
			// Use the *date* part of the last updated value
			startDate = last.UTC().Format(dateFmt)
		}

		// Up to today
		now := time.Now().UTC()
		finishDate = now.Format(dateFmt)

		// Build filterExpression per Phorest docs:
		// updated=<2018-01-31T23:59:59.999Z&updated=>2018-01-01T00:0:00.000Z
		fromTS := startDate + "T00:00:00.000Z"
		toTS := finishDate + "T23:59:59.999Z"
		filterExpr = fmt.Sprintf("updated=<%s&updated=>%s", toTS, fromTS)

		lg.Printf("ℹ️ %s: using startFilter=%q finishFilter=%q filterExpression=%q",
			b.BranchID, startDate, finishDate, filterExpr)

		// 3) Create CSV export job
		job, err := r.Export.CreateCSVExport(
			ctx,
			b.BranchID,
			JobTypeTransactionsCSV, // "TRANSACTIONS_CSV"
			filterExpr,
			startDate,
			finishDate,
		)
		if err != nil {
			return fmt.Errorf("create TRANSACTIONS_CSV export for %s: %w", b.BranchID, err)
		}
		lg.Printf("📝 %s: created TRANSACTIONS_CSV job %s (%s)", b.BranchID, job.JobID, job.JobStatus)

		// 4) Poll job
		waitMax := 5 * time.Minute
		final, err := r.Export.WaitForCSVJob(
			r.Cfg.PhorestBusiness,
			b.BranchID,
			job.JobID,
			waitMax,
		)
		if err != nil {
			// Special-case "No records found" so we don't treat it as a hard failure
			if final != nil && final.FailureReason != nil && *final.FailureReason == "No records found" {
				lg.Printf("ℹ️ %s: no new transactions in window %s..%s", b.BranchID, startDate, finishDate)
				continue
			}
			return fmt.Errorf("wait for TRANSACTIONS_CSV job %s (%s): %w", job.JobID, b.BranchID, err)
		}

		if final.TempCSVExternalURL == nil || *final.TempCSVExternalURL == "" {
			lg.Printf("⚠️ %s: job %s DONE but no csv URL; skipping import", b.BranchID, job.JobID)
			continue
		}
		lg.Printf("📥 %s: job %s DONE, URL received", b.BranchID, job.JobID)

		// 5) Download CSV to export dir
		filename := fmt.Sprintf("transactions_incremental_%s_%s.csv", b.BranchID, now.Format("20060102_150405"))
		dest := filepath.Join(r.Cfg.ExportDir, filename)

		if err := r.Export.DownloadCSV(*final.TempCSVExternalURL, dest); err != nil {
			return fmt.Errorf("%s: download csv: %w", b.BranchID, err)
		}
		lg.Printf("💾 %s: saved TRANSACTIONS_CSV to %s", b.BranchID, dest)

		// 6) Re-use your existing CSV import logic
		if err := r.importSingleTransactionsCSV(dest); err != nil {
			return fmt.Errorf("import incremental transactions csv %s: %w", dest, err)
		}

		// Archive this CSV into the bootstrap transactions dir
		r.archiveCSVToSeed(dest, "data/transactions")

		lg.Printf("✅ TRANSACTIONS_CSV incremental sync finished for %s", b.BranchID)
	}

	lg.Printf("✅ All branches incremental TRANSACTIONS_CSV sync finished")
	return nil
}
