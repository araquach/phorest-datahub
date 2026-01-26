package phorest

import (
	"time"

	"gorm.io/gorm"
)

// small helper DTOs
type txWatermarkRow struct {
	BranchID       string     `gorm:"column:branch_id"`
	MaxUpdatedAtPh *time.Time `gorm:"column:max_updated_at"`
}

type clientsWatermarkRow struct {
	MaxUpdatedAtPh *time.Time `gorm:"column:max_updated_at"`
}

type apptWatermarkRow struct {
	BranchID       string     `gorm:"column:branch_id"`
	MaxUpdatedAtPh *time.Time `gorm:"column:max_updated_at"`
}

// BootstrapWatermarks inspects existing data and seeds sync_watermarks.
//
// - transactions_csv: per-branch, from transaction_items.updated_at_phorest
// - clients_csv: global (branch_id NULL), from clients.updated_at_phorest
func (r *Runner) BootstrapWatermarks() error {
	lg := r.Logger
	db := r.DB

	lg.Printf("🔧 Bootstrapping sync_watermarks from existing data...")

	// 1) Per-branch watermark for transaction_items
	var txRows []txWatermarkRow
	if err := db.
		Raw(`
			SELECT
				branch_id,
				MAX(updated_at_phorest) AS max_updated_at
			FROM transaction_items
			WHERE updated_at_phorest IS NOT NULL
			GROUP BY branch_id
		`).Scan(&txRows).Error; err != nil {
		return err
	}

	for _, row := range txRows {
		if row.MaxUpdatedAtPh == nil {
			continue
		}
		if err := upsertWatermark(db,
			"transactions_csv",
			&row.BranchID,
			*row.MaxUpdatedAtPh,
		); err != nil {
			return err
		}
		lg.Printf("  • seeded watermark for transactions_csv / branch %s at %s",
			row.BranchID, row.MaxUpdatedAtPh.UTC().Format(time.RFC3339))
	}

	// 2) Global watermark for clients (branch_id = 'ALL')
	var clientRow clientsWatermarkRow
	if err := db.Raw(`
    SELECT MAX(updated_at_phorest) AS max_updated_at
    FROM clients
    WHERE updated_at_phorest IS NOT NULL
`).Scan(&clientRow).Error; err != nil {
		return err
	}

	if clientRow.MaxUpdatedAtPh != nil {
		all := "ALL"
		if err := upsertWatermark(db,
			"clients_csv",
			&all, // <── HERE: use "ALL"
			*clientRow.MaxUpdatedAtPh,
		); err != nil {
			return err
		}
		lg.Printf("  • seeded watermark for clients_csv (ALL) at %s",
			clientRow.MaxUpdatedAtPh.UTC().Format(time.RFC3339))
	} else {
		lg.Printf("  • no clients.updated_at_phorest found; skipping clients_csv")
	}

	// 3) Per-branch watermark for appointments_api (seed CSV bootstrap watermark)
	var apptRows []apptWatermarkRow
	if err := db.
		Raw(`
			SELECT
				branch_id,
				MAX(updated_at_phorest) AS max_updated_at
			FROM appointments_api
			WHERE updated_at_phorest IS NOT NULL
			GROUP BY branch_id
		`).Scan(&apptRows).Error; err != nil {
		return err
	}

	for _, row := range apptRows {
		if row.MaxUpdatedAtPh == nil || row.BranchID == "" {
			continue
		}
		if err := upsertWatermark(db,
			"appointments_api",
			&row.BranchID,
			*row.MaxUpdatedAtPh,
		); err != nil {
			return err
		}
		lg.Printf("  • seeded watermark for appointments_api_csv / branch %s at %s",
			row.BranchID, row.MaxUpdatedAtPh.UTC().Format(time.RFC3339))
	}

	lg.Printf("✅ sync_watermarks bootstrap complete.")
	return nil
}

// upsertWatermark inserts or updates a single (entity, branch_id) row.
func upsertWatermark(db *gorm.DB, entity string, branchID *string, lastUpdatedPh time.Time) error {
	if lastUpdatedPh.IsZero() {
		return nil
	}

	// Normalise "global" to ALL
	if branchID == nil || *branchID == "" {
		all := "ALL"
		branchID = &all
	}

	return db.Exec(`
		INSERT INTO sync_watermarks (entity, branch_id, last_updated_phorest, created_at, updated_at)
		VALUES ($1, $2, $3, now(), now())
		ON CONFLICT (entity, branch_id)
		DO UPDATE SET
			last_updated_phorest = GREATEST(sync_watermarks.last_updated_phorest, EXCLUDED.last_updated_phorest),
			updated_at           = now()
	`, entity, *branchID, lastUpdatedPh).Error
}
