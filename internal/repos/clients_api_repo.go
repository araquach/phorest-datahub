package repos

import (
	"log"

	"github.com/araquach/phorest-datahub/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ClientsAPIRepo struct {
	db *gorm.DB
	lg *log.Logger
}

func NewClientsAPIRepo(db *gorm.DB, lg *log.Logger) *ClientsAPIRepo {
	return &ClientsAPIRepo{db: db, lg: lg}
}

func (r *ClientsAPIRepo) UpsertBatch(rows []models.ClientAPI, batchSize int) error {
	if len(rows) == 0 {
		return nil
	}
	if batchSize <= 0 {
		batchSize = 500
	}

	// “newer wins” based on updated_at_phorest
	for i := 0; i < len(rows); i += batchSize {
		end := i + batchSize
		if end > len(rows) {
			end = len(rows)
		}
		chunk := rows[i:end]

		res := r.db.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "client_id"}},
			DoUpdates: clause.Assignments(map[string]interface{}{
				"version":                    gorm.Expr("CASE WHEN EXCLUDED.updated_at_phorest > clients_api.updated_at_phorest OR clients_api.updated_at_phorest IS NULL THEN EXCLUDED.version ELSE clients_api.version END"),
				"first_name":                 gorm.Expr("CASE WHEN EXCLUDED.updated_at_phorest > clients_api.updated_at_phorest OR clients_api.updated_at_phorest IS NULL THEN EXCLUDED.first_name ELSE clients_api.first_name END"),
				"last_name":                  gorm.Expr("CASE WHEN EXCLUDED.updated_at_phorest > clients_api.updated_at_phorest OR clients_api.updated_at_phorest IS NULL THEN EXCLUDED.last_name ELSE clients_api.last_name END"),
				"mobile":                     gorm.Expr("CASE WHEN EXCLUDED.updated_at_phorest > clients_api.updated_at_phorest OR clients_api.updated_at_phorest IS NULL THEN EXCLUDED.mobile ELSE clients_api.mobile END"),
				"email":                      gorm.Expr("CASE WHEN EXCLUDED.updated_at_phorest > clients_api.updated_at_phorest OR clients_api.updated_at_phorest IS NULL THEN EXCLUDED.email ELSE clients_api.email END"),
				"street_address_1":           gorm.Expr("CASE WHEN EXCLUDED.updated_at_phorest > clients_api.updated_at_phorest OR clients_api.updated_at_phorest IS NULL THEN EXCLUDED.street_address_1 ELSE clients_api.street_address_1 END"),
				"street_address_2":           gorm.Expr("CASE WHEN EXCLUDED.updated_at_phorest > clients_api.updated_at_phorest OR clients_api.updated_at_phorest IS NULL THEN EXCLUDED.street_address_2 ELSE clients_api.street_address_2 END"),
				"city":                       gorm.Expr("CASE WHEN EXCLUDED.updated_at_phorest > clients_api.updated_at_phorest OR clients_api.updated_at_phorest IS NULL THEN EXCLUDED.city ELSE clients_api.city END"),
				"postal_code":                gorm.Expr("CASE WHEN EXCLUDED.updated_at_phorest > clients_api.updated_at_phorest OR clients_api.updated_at_phorest IS NULL THEN EXCLUDED.postal_code ELSE clients_api.postal_code END"),
				"country":                    gorm.Expr("CASE WHEN EXCLUDED.updated_at_phorest > clients_api.updated_at_phorest OR clients_api.updated_at_phorest IS NULL THEN EXCLUDED.country ELSE clients_api.country END"),
				"gender":                     gorm.Expr("CASE WHEN EXCLUDED.updated_at_phorest > clients_api.updated_at_phorest OR clients_api.updated_at_phorest IS NULL THEN EXCLUDED.gender ELSE clients_api.gender END"),
				"loyalty_card_serial":        gorm.Expr("CASE WHEN EXCLUDED.updated_at_phorest > clients_api.updated_at_phorest OR clients_api.updated_at_phorest IS NULL THEN EXCLUDED.loyalty_card_serial ELSE clients_api.loyalty_card_serial END"),
				"loyalty_points":             gorm.Expr("CASE WHEN EXCLUDED.updated_at_phorest > clients_api.updated_at_phorest OR clients_api.updated_at_phorest IS NULL THEN EXCLUDED.loyalty_points ELSE clients_api.loyalty_points END"),
				"credit_outstanding_balance": gorm.Expr("CASE WHEN EXCLUDED.updated_at_phorest > clients_api.updated_at_phorest OR clients_api.updated_at_phorest IS NULL THEN EXCLUDED.credit_outstanding_balance ELSE clients_api.credit_outstanding_balance END"),
				"credit_days":                gorm.Expr("CASE WHEN EXCLUDED.updated_at_phorest > clients_api.updated_at_phorest OR clients_api.updated_at_phorest IS NULL THEN EXCLUDED.credit_days ELSE clients_api.credit_days END"),
				"credit_limit":               gorm.Expr("CASE WHEN EXCLUDED.updated_at_phorest > clients_api.updated_at_phorest OR clients_api.updated_at_phorest IS NULL THEN EXCLUDED.credit_limit ELSE clients_api.credit_limit END"),
				"updated_at_phorest":         gorm.Expr("GREATEST(clients_api.updated_at_phorest, EXCLUDED.updated_at_phorest)"),
				"updated_at":                 gorm.Expr("now()"),
			}),
		}).Create(&chunk)

		if res.Error != nil {
			return res.Error
		}
	}

	r.lg.Printf("Upserted %d clients_api rows", len(rows))
	return nil
}
