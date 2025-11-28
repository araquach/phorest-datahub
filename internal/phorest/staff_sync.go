package phorest

import (
	"time"

	"staff-appraisals/internal/repos"
)

// SyncStaffFromAPI fetches staff for each configured branch and upserts them.
func (r *Runner) SyncStaffFromAPI() error {
	c := NewStaffClient(
		r.Cfg.PhorestUsername,
		r.Cfg.PhorestPassword,
		r.Cfg.PhorestBusiness,
		r.Logger,
	)

	repo := repos.NewStaffRepo(r.DB, r.Logger)
	wr := repos.NewWatermarksRepo(r.DB, r.Logger)

	for _, b := range r.Cfg.Branches {
		if b.BranchID == "" {
			r.Logger.Printf("⚠️  Skipping branch with empty BranchID (name=%q)", b.Name)
			continue
		}

		r.Logger.Printf("Fetching staff for %s (%s)", b.Name, b.BranchID)

		rows, err := c.FetchStaff(b.BranchID)
		if err != nil {
			r.Logger.Printf("❌ staff fetch failed for %s (%s): %v", b.Name, b.BranchID, err)
			continue
		}
		if len(rows) == 0 {
			r.Logger.Printf("No staff to upsert for %s (%s)", b.Name, b.BranchID)
			continue
		}

		if err := repo.UpsertMany(rows); err != nil {
			r.Logger.Printf("❌ staff upsert failed for %s (%s): %v", b.Name, b.BranchID, err)
			continue
		}

		// 🔹 record / advance watermark for this branch
		now := time.Now().UTC()
		if err := wr.UpsertLastUpdated("staff_api", b.BranchID, now); err != nil {
			r.Logger.Printf("⚠️ failed to update staff_api watermark for %s (%s): %v", b.Name, b.BranchID, err)
			// you could `continue` or `return err` here depending on how strict you want to be
		}

		r.Logger.Printf("✅ staff upserted for %s (%s): %d", b.Name, b.BranchID, len(rows))
	}

	return nil
}
