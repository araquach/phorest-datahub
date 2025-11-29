package phorest

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"staff-appraisals/internal/models"
	"staff-appraisals/internal/repos"
)

func (r *Runner) RunIncrementalReviewsSync(ctx context.Context) error {
	lg := r.Logger
	db := r.DB

	lg.Printf("▶️ Starting incremental REVIEWS sync...")

	rc := NewReviewsClient(
		r.Cfg.PhorestUsername,
		r.Cfg.PhorestPassword,
		r.Cfg.PhorestBusiness,
	)

	rr := repos.NewReviewsRepo(db, lg)
	wr := repos.NewWatermarksRepo(db, lg)

	// Process branch by branch
	for _, b := range r.Cfg.Branches {
		branchID := b.BranchID
		if branchID == "" {
			lg.Printf("⚠️ Skipping branch %q with empty BranchID", b.Name)
			continue
		}

		lg.Printf("🏢 Branch %s (%s): starting REVIEWS sync", b.Name, branchID)

		// Last known review date (in DB, not from watermark)
		lastDateStr, err := rr.MaxReviewDate(branchID)
		if err != nil {
			return fmt.Errorf("max review_date for %s: %w", branchID, err)
		}

		if lastDateStr != nil && *lastDateStr != "" {
			lg.Printf("ℹ️ %s: existing max review_date = %s", branchID, *lastDateStr)
		} else {
			lg.Printf("ℹ️ %s: no existing reviews in DB, treating as full bootstrap", branchID)
		}

		const pageSize = 100
		page := 0
		duplicatePages := 0
		const duplicatePageThreshold = 3

		var allNew []models.Review
		var latestInRun *time.Time

		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			rows, totalPages, err := rc.FetchReviews(branchID, "", page, pageSize)
			if err != nil {
				return fmt.Errorf("fetch reviews branch=%s page=%d: %w", branchID, page, err)
			}
			if len(rows) == 0 {
				lg.Printf("ℹ️ %s: no rows on page %d (totalPages=%d), stopping", branchID, page, totalPages)
				break
			}

			// How many of these review IDs are already in DB?
			ids := make([]string, len(rows))
			for i, rv := range rows {
				ids[i] = rv.ReviewID
			}
			existingCount, err := rr.CountExistingByIDs(branchID, ids)
			if err != nil {
				return fmt.Errorf("count existing reviews branch=%s page=%d: %w", branchID, page, err)
			}

			dupRatio := float64(existingCount) / float64(len(rows))
			lg.Printf("   %s: page=%d size=%d existing=%d dupRatio=%.2f",
				branchID, page, len(rows), existingCount, dupRatio)

			// Upsert entire page – UpsertMany() is idempotent (DO NOTHING on conflict).
			if err := rr.UpsertMany(rows); err != nil {
				return fmt.Errorf("upsert reviews branch=%s page=%d: %w", branchID, page, err)
			}

			// Only consider pages that have at least one non-duplicate for CSV + watermark.
			if dupRatio < 1.0 {
				allNew = append(allNew, rows...)

				for i := range rows {
					if rows[i].ReviewDate != nil {
						if latestInRun == nil || rows[i].ReviewDate.After(*latestInRun) {
							// NOTE: ReviewDate is a *date*, but we keep it as midnight UTC.
							t := time.Date(
								rows[i].ReviewDate.Year(),
								rows[i].ReviewDate.Month(),
								rows[i].ReviewDate.Day(),
								0, 0, 0, 0,
								time.UTC,
							)
							latestInRun = &t
						}
					}
				}
			}

			// Duplicate-page detection: once we see several pages that are mostly
			// already in DB, assume we’ve overlapped the historical region and stop.
			if dupRatio >= 0.9 {
				duplicatePages++
			} else {
				duplicatePages = 0
			}

			if duplicatePages >= duplicatePageThreshold {
				lg.Printf("ℹ️ %s: hit %d near-duplicate pages in a row, stopping at page %d",
					branchID, duplicatePageThreshold, page)
				break
			}

			page++
			if totalPages > 0 && page >= totalPages {
				lg.Printf("ℹ️ %s: reached totalPages=%d, stopping", branchID, totalPages)
				break
			}
		}

		if len(allNew) == 0 {
			lg.Printf("✅ %s: no new reviews detected; nothing to archive", branchID)
			continue
		}

		// 1) Write per-run CSV backup into ExportDir
		timestamp := time.Now().UTC().Format("20060102_150405")
		filename := fmt.Sprintf("reviews_incremental_%s_%s.csv", branchID, timestamp)
		tmpPath := filepath.Join(r.Cfg.ExportDir, filename)

		if err := writeReviewsCSV(tmpPath, allNew); err != nil {
			return fmt.Errorf("write reviews CSV for %s: %w", branchID, err)
		}
		lg.Printf("💾 %s: saved reviews CSV to %s", branchID, tmpPath)

		// 2) Archive into data/reviews for future bootstrap
		archiveDir := "data/reviews"
		if err := os.MkdirAll(archiveDir, 0o755); err != nil {
			return fmt.Errorf("mkdir %s: %w", archiveDir, err)
		}
		finalPath := filepath.Join(archiveDir, filename)
		if err := os.Rename(tmpPath, finalPath); err != nil {
			return fmt.Errorf("archive reviews CSV for %s: %w", branchID, err)
		}
		lg.Printf("📦 %s: archived %s → %s (for future bootstrap)", branchID, tmpPath, finalPath)

		// 3) Update watermark if we actually saw newer review dates
		if latestInRun != nil {
			if err := wr.UpsertLastUpdated("reviews_api", branchID, *latestInRun); err != nil {
				return fmt.Errorf("update reviews_api watermark for %s: %w", branchID, err)
			}
			lg.Printf("💾 %s: updated reviews_api watermark → %s",
				branchID, latestInRun.Format("2006-01-02"))
		}

		lg.Printf("✅ %s: incremental REVIEWS sync finished (%d rows touched)", branchID, len(allNew))
	}

	lg.Printf("✅ All branches incremental REVIEWS sync finished")
	return nil
}
