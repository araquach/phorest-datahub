package repos

import (
	"log"

	"github.com/araquach/phorest-datahub/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type AppointmentsAPIRepo struct {
	db *gorm.DB
	lg *log.Logger
}

func (r *AppointmentsAPIRepo) Count() (int64, error) {
	var n int64
	err := r.db.Model(&models.AppointmentAPI{}).Count(&n).Error
	return n, err
}

func NewAppointmentsAPIRepo(db *gorm.DB, lg *log.Logger) *AppointmentsAPIRepo {
	return &AppointmentsAPIRepo{
		db: db,
		lg: lg,
	}
}

func (r *AppointmentsAPIRepo) UpsertBatch(rows []models.AppointmentAPI, batchSize int) error {
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
				{Name: "appointment_id"},
			},
			DoUpdates: clause.Assignments(map[string]interface{}{
				"version": gorm.Expr(`
					CASE
					  WHEN EXCLUDED.updated_at_phorest > appointments_api.updated_at_phorest
					    OR appointments_api.updated_at_phorest IS NULL
					  THEN EXCLUDED.version
					  ELSE appointments_api.version
					END
				`),

				"appointment_date": gorm.Expr(`CASE WHEN EXCLUDED.updated_at_phorest > appointments_api.updated_at_phorest OR appointments_api.updated_at_phorest IS NULL THEN EXCLUDED.appointment_date ELSE appointments_api.appointment_date END`),
				"start_time":       gorm.Expr(`CASE WHEN EXCLUDED.updated_at_phorest > appointments_api.updated_at_phorest OR appointments_api.updated_at_phorest IS NULL THEN EXCLUDED.start_time ELSE appointments_api.start_time END`),
				"end_time":         gorm.Expr(`CASE WHEN EXCLUDED.updated_at_phorest > appointments_api.updated_at_phorest OR appointments_api.updated_at_phorest IS NULL THEN EXCLUDED.end_time ELSE appointments_api.end_time END`),

				"price":          gorm.Expr(`CASE WHEN EXCLUDED.updated_at_phorest > appointments_api.updated_at_phorest OR appointments_api.updated_at_phorest IS NULL THEN EXCLUDED.price ELSE appointments_api.price END`),
				"deposit_amount": gorm.Expr(`CASE WHEN EXCLUDED.updated_at_phorest > appointments_api.updated_at_phorest OR appointments_api.updated_at_phorest IS NULL THEN EXCLUDED.deposit_amount ELSE appointments_api.deposit_amount END`),

				"staff_id":   gorm.Expr(`CASE WHEN EXCLUDED.updated_at_phorest > appointments_api.updated_at_phorest OR appointments_api.updated_at_phorest IS NULL THEN EXCLUDED.staff_id ELSE appointments_api.staff_id END`),
				"confirmed":  gorm.Expr(`CASE WHEN EXCLUDED.updated_at_phorest > appointments_api.updated_at_phorest OR appointments_api.updated_at_phorest IS NULL THEN EXCLUDED.confirmed ELSE appointments_api.confirmed END`),
				"service_id": gorm.Expr(`CASE WHEN EXCLUDED.updated_at_phorest > appointments_api.updated_at_phorest OR appointments_api.updated_at_phorest IS NULL THEN EXCLUDED.service_id ELSE appointments_api.service_id END`),
				"client_id":  gorm.Expr(`CASE WHEN EXCLUDED.updated_at_phorest > appointments_api.updated_at_phorest OR appointments_api.updated_at_phorest IS NULL THEN EXCLUDED.client_id ELSE appointments_api.client_id END`),

				"staff_request":   gorm.Expr(`CASE WHEN EXCLUDED.updated_at_phorest > appointments_api.updated_at_phorest OR appointments_api.updated_at_phorest IS NULL THEN EXCLUDED.staff_request ELSE appointments_api.staff_request END`),
				"preferred_staff": gorm.Expr(`CASE WHEN EXCLUDED.updated_at_phorest > appointments_api.updated_at_phorest OR appointments_api.updated_at_phorest IS NULL THEN EXCLUDED.preferred_staff ELSE appointments_api.preferred_staff END`),

				"service_reward_id":    gorm.Expr(`CASE WHEN EXCLUDED.updated_at_phorest > appointments_api.updated_at_phorest OR appointments_api.updated_at_phorest IS NULL THEN EXCLUDED.service_reward_id ELSE appointments_api.service_reward_id END`),
				"purchasing_branch_id": gorm.Expr(`CASE WHEN EXCLUDED.updated_at_phorest > appointments_api.updated_at_phorest OR appointments_api.updated_at_phorest IS NULL THEN EXCLUDED.purchasing_branch_id ELSE appointments_api.purchasing_branch_id END`),
				"service_name":         gorm.Expr(`CASE WHEN EXCLUDED.updated_at_phorest > appointments_api.updated_at_phorest OR appointments_api.updated_at_phorest IS NULL THEN EXCLUDED.service_name ELSE appointments_api.service_name END`),

				"state":            gorm.Expr(`CASE WHEN EXCLUDED.updated_at_phorest > appointments_api.updated_at_phorest OR appointments_api.updated_at_phorest IS NULL THEN EXCLUDED.state ELSE appointments_api.state END`),
				"activation_state": gorm.Expr(`CASE WHEN EXCLUDED.updated_at_phorest > appointments_api.updated_at_phorest OR appointments_api.updated_at_phorest IS NULL THEN EXCLUDED.activation_state ELSE appointments_api.activation_state END`),

				"deposit_datetime": gorm.Expr(`CASE WHEN EXCLUDED.updated_at_phorest > appointments_api.updated_at_phorest OR appointments_api.updated_at_phorest IS NULL THEN EXCLUDED.deposit_datetime ELSE appointments_api.deposit_datetime END`),
				"booking_id":       gorm.Expr(`CASE WHEN EXCLUDED.updated_at_phorest > appointments_api.updated_at_phorest OR appointments_api.updated_at_phorest IS NULL THEN EXCLUDED.booking_id ELSE appointments_api.booking_id END`),
				"source":           gorm.Expr(`CASE WHEN EXCLUDED.updated_at_phorest > appointments_api.updated_at_phorest OR appointments_api.updated_at_phorest IS NULL THEN EXCLUDED.source ELSE appointments_api.source END`),
				"deleted":          gorm.Expr(`CASE WHEN EXCLUDED.updated_at_phorest > appointments_api.updated_at_phorest OR appointments_api.updated_at_phorest IS NULL THEN EXCLUDED.deleted ELSE appointments_api.deleted END`),

				"internet_service_categories": gorm.Expr(`CASE WHEN EXCLUDED.updated_at_phorest > appointments_api.updated_at_phorest OR appointments_api.updated_at_phorest IS NULL THEN EXCLUDED.internet_service_categories ELSE appointments_api.internet_service_categories END`),

				"created_at_phorest": gorm.Expr(`CASE WHEN appointments_api.created_at_phorest IS NULL THEN EXCLUDED.created_at_phorest ELSE appointments_api.created_at_phorest END`),
				"updated_at_phorest": gorm.Expr(`GREATEST(appointments_api.updated_at_phorest, EXCLUDED.updated_at_phorest)`),
				"updated_at":         gorm.Expr(`now()`),
			}),
		}).Create(&chunk)

		if res.Error != nil {
			return res.Error
		}
	}

	r.lg.Printf("Upserted %d appointments_api rows", len(rows))
	return nil
}
