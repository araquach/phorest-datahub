package phorest

import (
	"fmt"
	"path/filepath"

	"staff-appraisals/internal/models"
	"staff-appraisals/internal/repos"
)

// BootstrapReviewsFromCSVsIfNeeded:
// - If the DB already has reviews, do nothing.
// - Otherwise, import all reviews from data/reviews/*.csv (if any).
func (r *Runner) BootstrapReviewsFromCSVsIfNeeded() error {
	lg := r.Logger
	db := r.DB

	// 1) Check if we already have any reviews
	var existing int64
	if err := db.Model(&models.Review{}).Count(&existing).Error; err != nil {
		return fmt.Errorf("count reviews: %w", err)
	}

	if existing > 0 {
		lg.Printf("⏭ Reviews bootstrap skipped: %d reviews already present", existing)
		return nil
	}

	// 2) Look for local backup CSVs
	reviewsDir := "data/reviews"
	paths, err := filepath.Glob(filepath.Join(reviewsDir, "*.csv"))
	if err != nil {
		return fmt.Errorf("scan reviews dir: %w", err)
	}
	if len(paths) == 0 {
		lg.Printf("ℹ️ No reviews CSV files found in %s; nothing to bootstrap", reviewsDir)
		return nil
	}

	lg.Printf("📂 Found %d reviews CSV files in %s; bootstrapping…", len(paths), reviewsDir)

	repo := repos.NewReviewsRepo(db, lg)

	for _, p := range paths {
		lg.Printf("📥 Importing reviews CSV: %s", p)

		batch, err := ParseReviewsCSV(p, lg)
		if err != nil {
			return fmt.Errorf("parse reviews csv %s: %w", p, err)
		}
		if len(batch.Reviews) == 0 {
			lg.Printf("⚠️  No reviews in %s; skipping", p)
			continue
		}

		if err := repo.UpsertMany(batch.Reviews); err != nil {
			return fmt.Errorf("upsert reviews from %s: %w", p, err)
		}

		lg.Printf("✅ Bootstrapped %d reviews from %s", len(batch.Reviews), p)
	}

	lg.Printf("🎉 Reviews bootstrap from CSV complete.")
	return nil
}
