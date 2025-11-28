package phorest

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"staff-appraisals/internal/repos"
)

func (r *Runner) RunIncrementalClientsSync(ctx context.Context) error {
	lg := r.Logger
	db := r.DB

	lg.Printf("▶️ Starting incremental CLIENT_CSV sync...")

	wr := repos.NewWatermarksRepo(db, lg)

	// --- 1) Read watermark
	last, err := wr.GetLastUpdated("clients_csv", "ALL")
	if err != nil {
		return fmt.Errorf("get clients_csv watermark: %w", err)
	}

	var filterExpr string
	if last == nil {
		lg.Printf("ℹ️ No clients_csv watermark – requesting full export")
		filterExpr = "" // full export
	} else {
		from := last.UTC().Format("2006-01-02T15:04:05.000Z")
		filterExpr = fmt.Sprintf("updated=>%s", from)
		lg.Printf("ℹ️ Using filterExpression=%q", filterExpr)
	}

	// --- 2) Choose a branch
	if len(r.Cfg.Branches) == 0 {
		return fmt.Errorf("no branches available")
	}
	b := r.Cfg.Branches[0]
	lg.Printf("🏢 Using branch %s (%s) for CLIENT_CSV", b.Name, b.BranchID)

	// --- 3) Create CSV export job
	job, err := r.Export.CreateCSVExport(
		ctx,
		b.BranchID,
		JobTypeClientsCSV, // "CLIENT_CSV"
		filterExpr,
		"", // startFilter – not required for clients
		"", // finishFilter
	)
	if err != nil {
		return fmt.Errorf("create CLIENT_CSV export: %w", err)
	}
	lg.Printf("📝 Created CLIENT_CSV job %s (%s)", job.JobID, job.JobStatus)

	// --- 4) Poll job
	waitMax := 5 * time.Minute
	final, err := r.Export.WaitForCSVJob(
		r.Cfg.PhorestBusiness,
		b.BranchID,
		job.JobID,
		waitMax,
	)
	if err != nil {
		return fmt.Errorf("wait for job %s: %w", job.JobID, err)
	}

	if final.TempCSVExternalURL == nil || *final.TempCSVExternalURL == "" {
		return fmt.Errorf("job %s DONE but no csv URL", job.JobID)
	}
	lg.Printf("📥 job done, URL received")

	// --- 5) Download CSV to export dir
	filename := time.Now().UTC().Format("clients_incremental_20060102_150405.csv")
	dest := filepath.Join(r.Cfg.ExportDir, filename)

	if err := r.Export.DownloadCSV(*final.TempCSVExternalURL, dest); err != nil {
		return fmt.Errorf("download csv: %w", err)
	}
	lg.Printf("💾 Saved CLIENT_CSV to %s", dest)

	// --- 6) Re-use your existing CSV import logic
	if err := r.importSingleClientsCSV(dest); err != nil {
		return fmt.Errorf("import incremental clients csv: %w", err)
	}

	// Archive this CSV into the bootstrap clients dir
	r.archiveCSVToSeed(dest, "data/clients")

	lg.Printf("✅ Incremental CLIENT_CSV sync finished")
	return nil
}
