package repos

import (
	"log"
	"time"

	"gorm.io/gorm"
)

const (
	WatermarkWorktimetableRolling      = "worktimetable"
	WatermarkWorktimetableBackfillDone = "worktimetable_backfill_done"
)

// WatermarksRepo provides access to the sync_watermarks table.
type WatermarksRepo struct {
	db *gorm.DB
	lg *log.Logger
}

func NewWatermarksRepo(db *gorm.DB, lg *log.Logger) *WatermarksRepo {
	return &WatermarksRepo{db: db, lg: lg}
}

// SyncWatermark matches the *current* sync_watermarks schema.
type SyncWatermark struct {
	ID                 int64      `gorm:"primaryKey;column:id"`
	Entity             string     `gorm:"column:entity"`               // e.g. "clients_csv", "transactions_csv"
	BranchID           *string    `gorm:"column:branch_id"`            // NULL or branch id, or "ALL" for global
	LastUpdatedPhorest *time.Time `gorm:"column:last_updated_phorest"` // watermark
	CreatedAt          time.Time  `gorm:"column:created_at"`
	UpdatedAt          time.Time  `gorm:"column:updated_at"`
}

func (SyncWatermark) TableName() string { return "sync_watermarks" }

// GetLastUpdated returns the last_updated_phorest for (entity, branchID).
// For global sources like clients, pass branchID = "ALL".
func (r *WatermarksRepo) GetLastUpdated(entity, branchID string) (*time.Time, error) {
	branchID = normaliseBranchID(branchID)

	var wm SyncWatermark
	err := r.db.
		Where("entity = ? AND branch_id = ?", entity, branchID).
		First(&wm).Error

	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return wm.LastUpdatedPhorest, nil
}

// UpsertLastUpdated advances the watermark for (entity, branchID) if candidate is newer.
// For global sources like clients, pass branchID = "ALL" (or "") – "" will be normalised.
func (r *WatermarksRepo) UpsertLastUpdated(entity, branchID string, candidate time.Time) error {
	if candidate.IsZero() {
		return nil
	}

	branchID = normaliseBranchID(branchID)

	r.lg.Printf("💾 Updating watermark for %s/%s → %s",
		entity, branchID, candidate.UTC().Format(time.RFC3339))

	return r.db.Exec(`
INSERT INTO sync_watermarks (entity, branch_id, last_updated_phorest, created_at, updated_at)
VALUES (?, ?, ?, now(), now())
ON CONFLICT (entity, branch_id) DO UPDATE
SET last_updated_phorest = GREATEST(sync_watermarks.last_updated_phorest, EXCLUDED.last_updated_phorest),
    updated_at           = now();
`, entity, branchID, candidate.UTC()).Error
}

func normaliseBranchID(branchID string) string {
	// Canonical "global" key
	if branchID == "" {
		return "ALL"
	}
	return branchID
}

func (r *WatermarksRepo) GetWorktimetableBackfillDone(branchID string) (*time.Time, error) {
	return r.GetLastUpdated(WatermarkWorktimetableBackfillDone, branchID)
}

func (r *WatermarksRepo) MarkWorktimetableBackfillDone(branchID string, doneAt time.Time) error {
	return r.UpsertLastUpdated(WatermarkWorktimetableBackfillDone, branchID, doneAt)
}
