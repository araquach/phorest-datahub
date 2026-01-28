package phorest

import (
	"context"
	"fmt"
	"time"

	"github.com/araquach/phorest-datahub/internal/repos"
)

func (r *Runner) RunIncrementalAppointmentsAPISync(ctx context.Context) error {
	lg := r.Logger
	db := r.DB

	lg.Printf("▶️ Starting APPOINTMENTS_API sync...")

	c := NewAppointmentsAPIClient(
		r.Cfg.PhorestUsername,
		r.Cfg.PhorestPassword,
		r.Cfg.PhorestBusiness,
	)

	repo := repos.NewAppointmentsAPIRepo(db, lg)
	wr := repos.NewWatermarksRepo(db, lg)

	const (
		pageSize = 100

		// include these by default (you wanted complete coverage)
		fetchCanceled       = true
		fetchDeleted        = true
		fetchArchived       = true
		fetchOnlineCategory = true

		// small overlap to avoid missing boundary updates
		overlap = 2 * time.Minute
	)

	// Tunables (cron-friendly)
	historyDays := getIntEnv("APPOINTMENTS_HISTORY_DAYS", 365)
	futureDays := getIntEnv("APPOINTMENTS_FUTURE_DAYS", 120)

	// Backfill mode (do not use updated_from AND do not update watermark)
	ignoreWM := getBoolEnv("APPOINTMENTS_IGNORE_WATERMARK", false)

	now := time.Now().UTC()
	defaultStart := dateOnly(now.AddDate(0, 0, -historyDays))
	defaultEnd := dateOnly(now.AddDate(0, 0, futureDays))

	fromOverride, err := getDateEnv("APPOINTMENTS_FROM_DATE")
	if err != nil {
		return err
	}
	toOverride, err := getDateEnv("APPOINTMENTS_TO_DATE")
	if err != nil {
		return err
	}

	appointmentStart := defaultStart
	appointmentEnd := defaultEnd

	if fromOverride != nil {
		appointmentStart = *fromOverride
	}
	if toOverride != nil {
		appointmentEnd = *toOverride
	}

	if appointmentStart.After(appointmentEnd) {
		return fmt.Errorf("appointments_api: invalid window: start=%s is after end=%s",
			appointmentStart.Format("2006-01-02"),
			appointmentEnd.Format("2006-01-02"),
		)
	}

	if ignoreWM {
		lg.Printf("🟥 APPOINTMENTS_IGNORE_WATERMARK=1 → BACKFILL MODE (no updated_from, no watermark updates)")
	} else {
		lg.Printf("🟩 Normal incremental mode (uses watermark + overlap, updates watermark)")
	}

	if fromOverride != nil || toOverride != nil {
		lg.Printf("🧭 Date overrides: start=%s end=%s",
			appointmentStart.Format("2006-01-02"),
			appointmentEnd.Format("2006-01-02"),
		)
	}

	for _, br := range r.Cfg.Branches {
		branchID := br.BranchID

		// Determine updated_from (nil = full fetch within window)
		var updatedFrom *time.Time
		var last *time.Time

		if !ignoreWM {
			var err error
			last, err = wr.GetLastUpdated("appointments_api", branchID)
			if err != nil {
				return fmt.Errorf("get appointments_api watermark branch=%s: %w", branchID, err)
			}
		}

		if ignoreWM {
			updatedFrom = nil
			lg.Printf("ℹ️ appointments_api/%s: IGNORE watermark enabled → no updated_from filter", branchID)
		} else if last != nil {
			uf := last.UTC().Add(-overlap)
			updatedFrom = &uf
			lg.Printf("ℹ️ appointments_api/%s: watermark=%s (updated_from=%s)",
				branchID,
				last.UTC().Format(time.RFC3339),
				updatedFrom.UTC().Format(time.RFC3339),
			)
		} else {
			updatedFrom = nil
			lg.Printf("ℹ️ appointments_api/%s: no watermark yet → bootstrap window (no updated_from filter)", branchID)
		}

		lg.Printf("📆 appointments_api/%s: scanning appointment_date %s → %s",
			branchID,
			appointmentStart.Format("2006-01-02"),
			appointmentEnd.Format("2006-01-02"),
		)

		var (
			maxUpdated    *time.Time
			touchedCount  int
			windowStart   = appointmentStart
			safetyCounter = 0
		)

		// month-chunked loop (API supports max one month)
		for !windowStart.After(appointmentEnd) {
			safetyCounter++
			if safetyCounter > 500 {
				return fmt.Errorf("appointments_api/%s: safety stop tripped (too many month iterations) start=%s end=%s",
					branchID,
					appointmentStart.Format("2006-01-02"),
					appointmentEnd.Format("2006-01-02"),
				)
			}

			windowEnd := endOfMonth(windowStart)
			if windowEnd.After(appointmentEnd) {
				windowEnd = appointmentEnd
			}
			// defensive
			if windowEnd.Before(windowStart) {
				windowEnd = windowStart
			}

			page := 0
			for {
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
				}

				rows, totalPages, err := c.FetchAppointmentsPage(
					ctx,
					branchID,
					windowStart,
					windowEnd,
					updatedFrom,
					page,
					pageSize,
					fetchCanceled,
					fetchDeleted,
					fetchArchived,
					fetchOnlineCategory,
				)
				if err != nil {
					return fmt.Errorf("fetch appointments branch=%s win=%s..%s page=%d: %w",
						branchID, windowStart.Format("2006-01-02"), windowEnd.Format("2006-01-02"), page, err)
				}

				if len(rows) == 0 {
					break
				}

				if err := repo.UpsertBatch(rows, 500); err != nil {
					return fmt.Errorf("upsert appointments branch=%s page=%d: %w", branchID, page, err)
				}

				touchedCount += len(rows)

				for i := range rows {
					if rows[i].UpdatedAtPhorest != nil {
						if maxUpdated == nil || rows[i].UpdatedAtPhorest.After(*maxUpdated) {
							maxUpdated = rows[i].UpdatedAtPhorest
						}
					}
				}

				page++
				if totalPages > 0 && page >= totalPages {
					break
				}
			}

			// move to first day of next month
			windowStart = firstDayOfNextMonth(windowStart)
		}

		if touchedCount == 0 {
			lg.Printf("✅ appointments_api/%s: no rows returned (nothing to do)", branchID)
			continue
		}

		// ✅ watermark update ONLY in normal mode
		if ignoreWM {
			lg.Printf("🧱 appointments_api/%s: BACKFILL MODE → watermark not updated", branchID)
		} else {
			if maxUpdated != nil {
				if err := wr.UpsertLastUpdated("appointments_api", branchID, *maxUpdated); err != nil {
					return fmt.Errorf("update appointments_api watermark branch=%s: %w", branchID, err)
				}
				lg.Printf("💾 appointments_api/%s: watermark → %s", branchID, maxUpdated.UTC().Format(time.RFC3339))
			} else {
				lg.Printf("⚠️ appointments_api/%s: touched rows but no updated_at_phorest seen; watermark unchanged", branchID)
			}
		}

		lg.Printf("✅ appointments_api/%s: finished (%d rows touched)", branchID, touchedCount)
	}

	return nil
}
