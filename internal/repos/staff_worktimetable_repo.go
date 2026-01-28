package repos

import (
	"log"
	"time"

	"github.com/araquach/phorest-datahub/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type StaffWorkTimetableRepo struct {
	db *gorm.DB
	lg *log.Logger
}

func NewStaffWorkTimetableRepo(db *gorm.DB, lg *log.Logger) *StaffWorkTimetableRepo {
	return &StaffWorkTimetableRepo{db: db, lg: lg}
}

func (r *StaffWorkTimetableRepo) DeleteWindow(branchID string, from, to time.Time) error {
	return r.db.
		Where("branch_id = ? AND slot_date BETWEEN ? AND ?", branchID, from, to).
		Delete(&models.StaffWorkTimetableSlot{}).
		Error
}

func (r *StaffWorkTimetableRepo) UpsertBatch(rows []models.StaffWorkTimetableSlot, batchSize int) error {
	if len(rows) == 0 {
		return nil
	}
	if batchSize <= 0 {
		batchSize = 1000
	}

	for i := 0; i < len(rows); i += batchSize {
		end := i + batchSize
		if end > len(rows) {
			end = len(rows)
		}
		chunk := rows[i:end]

		res := r.db.Clauses(clause.OnConflict{
			DoNothing: true,
		}).Create(&chunk)

		if res.Error != nil {
			return res.Error
		}
	}

	return nil
}
