// internal/phorest/staff_worktimetable_incremental.go
package phorest

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/araquach/phorest-datahub/internal/models"
	"github.com/araquach/phorest-datahub/internal/repos"
)

// Env keys (kept explicit so your .env matches reality)
const (
	EnvWorktimetablePastDays    = "WORKTIMETABLE_PAST_DAYS"    // preferred (your current env)
	EnvWorktimetableHistoryDays = "WORKTIMETABLE_HISTORY_DAYS" // legacy fallback
	EnvWorktimetableFutureDays  = "WORKTIMETABLE_FUTURE_DAYS"

	EnvWorktimetableFromDate = "WORKTIMETABLE_FROM_DATE" // optional override for rolling window
	EnvWorktimetableToDate   = "WORKTIMETABLE_TO_DATE"   // optional override for rolling window

	EnvWorktimetableBackfillEnabled = "WORKTIMETABLE_BACKFILL"      // true/false
	EnvWorktimetableBackfillFrom    = "WORKTIMETABLE_BACKFILL_FROM" // YYYY-MM-DD
	EnvWorktimetableBackfillTo      = "WORKTIMETABLE_BACKFILL_TO"   // YYYY-MM-DD (optional cap)

	EnvWorktimetableActivityType = "WORKTIMETABLE_ACTIVITY_TYPE" // optional
)

func (r *Runner) RunIncrementalStaffWorkTimetableSync(ctx context.Context) error {
	lg := r.Logger
	db := r.DB

	lg.Printf("▶️ Starting STAFF_WORKTIMETABLE sync...")

	client := NewStaffWorkTimetableClient(
		r.Cfg.PhorestUsername,
		r.Cfg.PhorestPassword,
		r.Cfg.PhorestBusiness,
	)

	const pageSize = 100

	// --- Rolling window config ---
	// Prefer WORKTIMETABLE_PAST_DAYS (what you set), but support legacy WORKTIMETABLE_HISTORY_DAYS too.
	historyDays := getIntEnv(EnvWorktimetablePastDays, 0)
	if historyDays <= 0 {
		historyDays = getIntEnv(EnvWorktimetableHistoryDays, 365)
	}
	futureDays := getIntEnv(EnvWorktimetableFutureDays, 120)

	now := time.Now().UTC()
	defaultStart := dateOnly(now.AddDate(0, 0, -historyDays))
	defaultEnd := dateOnly(now.AddDate(0, 0, futureDays))

	fromOverride, err := getDateEnv(EnvWorktimetableFromDate)
	if err != nil {
		return err
	}
	toOverride, err := getDateEnv(EnvWorktimetableToDate)
	if err != nil {
		return err
	}

	rollingStart := defaultStart
	rollingEnd := defaultEnd
	if fromOverride != nil {
		rollingStart = *fromOverride
	}
	if toOverride != nil {
		rollingEnd = *toOverride
	}

	if rollingStart.After(rollingEnd) {
		return fmt.Errorf("worktimetable: invalid rolling window: start=%s is after end=%s",
			rollingStart.Format("2006-01-02"), rollingEnd.Format("2006-01-02"),
		)
	}

	// Optional activity_type
	var activityType *string
	rawAct := strings.TrimSpace(os.Getenv(EnvWorktimetableActivityType))
	if rawAct != "" {
		activityType = &rawAct
		lg.Printf("🧭 worktimetable: activity_type=%s", rawAct)
	}

	// --- Backfill config (gated!) ---
	backfillEnabled := getBoolEnv(EnvWorktimetableBackfillEnabled, false)

	// Default backfill-from if not set: 2017-01-01 (UTC, date-only)
	backfillFrom := time.Date(2017, 1, 1, 0, 0, 0, 0, time.UTC)
	if s := strings.TrimSpace(os.Getenv(EnvWorktimetableBackfillFrom)); s != "" {
		t, err := time.Parse("2006-01-02", s)
		if err != nil {
			return fmt.Errorf("invalid %s: %w", EnvWorktimetableBackfillFrom, err)
		}
		backfillFrom = dateOnly(t.UTC())
	}

	var backfillTo *time.Time
	if s := strings.TrimSpace(os.Getenv(EnvWorktimetableBackfillTo)); s != "" {
		t, err := time.Parse("2006-01-02", s)
		if err != nil {
			return fmt.Errorf("invalid %s: %w", EnvWorktimetableBackfillTo, err)
		}
		bt := dateOnly(t.UTC())
		backfillTo = &bt
	}

	// Watermarks repo (rolling + backfill-done markers)
	wmRepo := repos.NewWatermarksRepo(db, lg)

	for _, br := range r.Cfg.Branches {
		branchID := br.BranchID

		// 1) BACKFILL (one-off) — ONLY when explicitly enabled AND not already done
		doneAt, err := wmRepo.GetWorktimetableBackfillDone(branchID)
		if err != nil {
			return fmt.Errorf("worktimetable/%s: read backfill watermark: %w", branchID, err)
		}

		if !backfillEnabled {
			lg.Printf("⏭  worktimetable/%s: backfill disabled (%s=false)", branchID, EnvWorktimetableBackfillEnabled)
		} else if doneAt != nil {
			lg.Printf("⏭  worktimetable/%s: backfill already done (%s)", branchID, doneAt.UTC().Format(time.RFC3339))
		} else {
			// Backfill end defaults to the day BEFORE rollingStart (so no overlap),
			// but can be capped by WORKTIMETABLE_BACKFILL_TO if provided.
			backfillEnd := dayBefore(rollingStart)
			if backfillTo != nil && backfillTo.Before(backfillEnd) {
				backfillEnd = *backfillTo
			}

			// Only run if the range is valid and actually before rollingStart.
			if backfillFrom.After(backfillEnd) {
				lg.Printf("⚠️  worktimetable/%s: backfill range invalid (%s > %s); skipping",
					branchID,
					backfillFrom.Format("2006-01-02"),
					backfillEnd.Format("2006-01-02"),
				)
			} else {
				lg.Printf("🕰️  worktimetable/%s: backfill starting %s → %s (one-off)",
					branchID,
					backfillFrom.Format("2006-01-02"),
					backfillEnd.Format("2006-01-02"),
				)

				if err := r.refreshWorktimetableRange(ctx, client, branchID, backfillFrom, backfillEnd, activityType, pageSize); err != nil {
					return err
				}
			}

			// Mark backfill done regardless (prevents infinite attempts if range is empty/invalid)
			if err := wmRepo.MarkWorktimetableBackfillDone(branchID, time.Now().UTC()); err != nil {
				return fmt.Errorf("worktimetable/%s: mark backfill done: %w", branchID, err)
			}
			lg.Printf("✅ worktimetable/%s: backfill marked done", branchID)
		}

		// 2) ROLLING WINDOW (every run)
		lg.Printf("📆 worktimetable/%s: rolling scan %s → %s",
			branchID,
			rollingStart.Format("2006-01-02"),
			rollingEnd.Format("2006-01-02"),
		)

		if err := r.refreshWorktimetableRange(ctx, client, branchID, rollingStart, rollingEnd, activityType, pageSize); err != nil {
			return err
		}

		// Optional: store a rolling “ran at” marker (separate from backfill-done)
		_ = wmRepo.UpsertLastUpdated(repos.WatermarkWorktimetableRolling, branchID, time.Now().UTC())
	}

	return nil
}

// refreshWorktimetableRange refreshes worktimetable data in month windows [start..end] inclusive.
// For each month window: fetch all pages -> delete window -> insert.
func (r *Runner) refreshWorktimetableRange(
	ctx context.Context,
	client *StaffWorkTimetableClient,
	branchID string,
	start time.Time,
	end time.Time,
	activityType *string,
	pageSize int,
) error {
	lg := r.Logger
	db := r.DB

	if start.After(end) {
		return fmt.Errorf("worktimetable/%s: invalid range: start=%s end=%s",
			branchID, start.Format("2006-01-02"), end.Format("2006-01-02"),
		)
	}

	windowStart := dateOnly(start.UTC())
	windowLimit := dateOnly(end.UTC())

	safetyCounter := 0
	totalTouched := 0

	for !windowStart.After(windowLimit) {
		safetyCounter++
		if safetyCounter > 5000 {
			return fmt.Errorf("worktimetable/%s: safety stop tripped (too many month iterations) start=%s end=%s",
				branchID,
				start.Format("2006-01-02"),
				end.Format("2006-01-02"),
			)
		}

		windowEnd := endOfMonth(windowStart)
		if windowEnd.After(windowLimit) {
			windowEnd = windowLimit
		}
		if windowEnd.Before(windowStart) {
			windowEnd = windowStart
		}

		// Fetch all pages for this window
		var fetched []models.StaffWorkTimetableSlot
		page := 0

		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			rows, totalPages, err := client.FetchWorkTimetablePage(
				ctx,
				branchID,
				windowStart,
				windowEnd,
				activityType,
				page,
				pageSize,
			)
			if err != nil {
				return fmt.Errorf(
					"fetch worktimetable branch=%s win=%s..%s page=%d: %w",
					branchID,
					windowStart.Format("2006-01-02"),
					windowEnd.Format("2006-01-02"),
					page,
					err,
				)
			}

			if len(rows) == 0 {
				break
			}

			fetched = append(fetched, rows...)

			page++
			if totalPages > 0 && page >= totalPages {
				break
			}
		}

		// Transactional replace for this month window
		tx := db.Begin()
		if tx.Error != nil {
			return tx.Error
		}
		defer func() {
			if p := recover(); p != nil {
				_ = tx.Rollback()
				panic(p)
			}
		}()

		monthRepo := repos.NewStaffWorkTimetableRepo(tx, lg)

		if err := monthRepo.DeleteWindow(branchID, windowStart, windowEnd); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf(
				"delete worktimetable branch=%s win=%s..%s: %w",
				branchID,
				windowStart.Format("2006-01-02"),
				windowEnd.Format("2006-01-02"),
				err,
			)
		}

		if err := monthRepo.UpsertBatch(fetched, 2000); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf(
				"insert worktimetable branch=%s win=%s..%s: %w",
				branchID,
				windowStart.Format("2006-01-02"),
				windowEnd.Format("2006-01-02"),
				err,
			)
		}

		if err := tx.Commit().Error; err != nil {
			return err
		}

		totalTouched += len(fetched)
		lg.Printf(
			"✅ worktimetable/%s: refreshed %s..%s (%d slots)",
			branchID,
			windowStart.Format("2006-01-02"),
			windowEnd.Format("2006-01-02"),
			len(fetched),
		)

		windowStart = firstDayOfNextMonth(windowStart)
	}

	lg.Printf("✅ worktimetable/%s: finished (%d slots total)", branchID, totalTouched)
	return nil
}
