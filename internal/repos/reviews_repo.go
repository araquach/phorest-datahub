package repos

import (
	"log"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"staff-appraisals/internal/models"
)

type ReviewsRepo struct {
	db *gorm.DB
	lg *log.Logger
}

func NewReviewsRepo(db *gorm.DB, lg *log.Logger) *ReviewsRepo {
	return &ReviewsRepo{db: db, lg: lg}
}

// UpsertMany: reviews are immutable, so we can safely DO NOTHING on conflict.
// We batch inserts to avoid Postgres' 65535-parameter limit.
func (r *ReviewsRepo) UpsertMany(rows []models.Review) error {
	if len(rows) == 0 {
		return nil
	}

	const batchSize = 500 // safely under parameter limit even with many columns
	total := 0

	for start := 0; start < len(rows); start += batchSize {
		end := start + batchSize
		if end > len(rows) {
			end = len(rows)
		}
		chunk := rows[start:end]

		res := r.db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "review_id"}},
			DoNothing: true,
		}).Create(&chunk)

		if res.Error != nil {
			return res.Error
		}
		total += len(chunk)
	}

	r.lg.Printf("Upserted/inserted reviews (batched): %d", total)
	return nil
}

// Watermark helpers (for incremental fetches by branch)
func (r *ReviewsRepo) MaxReviewDate(branchID string) (*string, error) {
	var ts *string
	err := r.db.
		Table("reviews").
		Select("MAX(review_date)::text").
		Where("branch_id = ?", branchID).
		Scan(&ts).Error
	return ts, err
}

func (r *ReviewsRepo) CountExistingByIDs(branchID string, ids []string) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}

	var count int64
	if err := r.db.
		Model(&models.Review{}).
		Where("branch_id = ? AND review_id IN (?)", branchID, ids).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}
