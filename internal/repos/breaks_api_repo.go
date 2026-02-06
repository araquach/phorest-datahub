package repos

import (
	"log"

	"github.com/araquach/phorest-datahub/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type BreaksAPIRepo struct {
	db *gorm.DB
	lg *log.Logger
}

func NewBreaksAPIRepo(db *gorm.DB, lg *log.Logger) *BreaksAPIRepo {
	return &BreaksAPIRepo{db: db, lg: lg}
}

func (r *BreaksAPIRepo) UpsertBatch(rows []models.BreakAPI, batchSize int) error {
	if len(rows) == 0 {
		return nil
	}
	if batchSize <= 0 {
		batchSize = 500
	}

	for i := 0; i < len(rows); i += batchSize {
		end := i + batchSize
		if end > len(rows) {
			end = len(rows)
		}
		chunk := rows[i:end]

		res := r.db.Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "branch_id"},
				{Name: "break_id"},
			},
			DoUpdates: clause.Assignments(map[string]interface{}{
				"version": gorm.Expr(`
					CASE
					  WHEN EXCLUDED.version >= breaks_api.version OR breaks_api.version IS NULL
					  THEN EXCLUDED.version
					  ELSE breaks_api.version
					END
				`),

				"break_date": gorm.Expr(`CASE WHEN EXCLUDED.version >= breaks_api.version OR breaks_api.version IS NULL THEN EXCLUDED.break_date ELSE breaks_api.break_date END`),
				"start_time": gorm.Expr(`CASE WHEN EXCLUDED.version >= breaks_api.version OR breaks_api.version IS NULL THEN EXCLUDED.start_time ELSE breaks_api.start_time END`),
				"end_time":   gorm.Expr(`CASE WHEN EXCLUDED.version >= breaks_api.version OR breaks_api.version IS NULL THEN EXCLUDED.end_time ELSE breaks_api.end_time END`),

				"staff_id":   gorm.Expr(`CASE WHEN EXCLUDED.version >= breaks_api.version OR breaks_api.version IS NULL THEN EXCLUDED.staff_id ELSE breaks_api.staff_id END`),
				"room_id":    gorm.Expr(`CASE WHEN EXCLUDED.version >= breaks_api.version OR breaks_api.version IS NULL THEN EXCLUDED.room_id ELSE breaks_api.room_id END`),
				"machine_id": gorm.Expr(`CASE WHEN EXCLUDED.version >= breaks_api.version OR breaks_api.version IS NULL THEN EXCLUDED.machine_id ELSE breaks_api.machine_id END`),
				"label":      gorm.Expr(`CASE WHEN EXCLUDED.version >= breaks_api.version OR breaks_api.version IS NULL THEN EXCLUDED.label ELSE breaks_api.label END`),
				"paid_break": gorm.Expr(`CASE WHEN EXCLUDED.version >= breaks_api.version OR breaks_api.version IS NULL THEN EXCLUDED.paid_break ELSE breaks_api.paid_break END`),

				"updated_at": gorm.Expr(`now()`),
			}),
		}).Create(&chunk)

		if res.Error != nil {
			return res.Error
		}
	}

	r.lg.Printf("Upserted %d breaks_api rows", len(rows))
	return nil
}
