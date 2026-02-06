package phorest

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/araquach/phorest-datahub/internal/repos"
)

// RunIncrementalBreaksAPISync
//
// Default behaviour (every run):
//   - rolling window: today-60 days -> today+180 days (configurable)
//
// Optional backfill (only when instructed):
//   - set BREAKS_BACKFILL=true AND provide BREAKS_FROM_DATE + BREAKS_TO_DATE (YYYY-MM-DD)
//
// Notes:
//   - Breaks API has no updated_from, so we re-fetch windows and upsert by (branch_id, break_id)
//   - Repo upsert should be version-gated (only overwrite when EXCLUDED.version >= existing.version)
func (r *Runner) RunIncrementalBreaksAPISync(ctx context.Context) error {
	lg := r.Logger
	db := r.DB

	lg.Printf("▶️ Starting BREAKS_API sync...")

	client := NewBreaksAPIClient(
		r.Cfg.PhorestUsername,
		r.Cfg.PhorestPassword,
		r.Cfg.PhorestBusiness,
	)

	repo := repos.NewBreaksAPIRepo(db, lg)

	now := time.Now().UTC()

	backDays := getIntEnv("BREAKS_BACK_DAYS", 60)
	forwardDays := getIntEnv("BREAKS_FORWARD_DAYS", 180)

	rollingStart := dateOnly(now.AddDate(0, 0, -backDays))
	rollingEnd := dateOnly(now.AddDate(0, 0, forwardDays))

	backfillEnabled := strings.EqualFold(strings.TrimSpace(os.Getenv("BREAKS_BACKFILL")), "true")

	fromOverride, err := getDateEnv("BREAKS_FROM_DATE")
	if err != nil {
		return err
	}
	toOverride, err := getDateEnv("BREAKS_TO_DATE")
	if err != nil {
		return err
	}

	if backfillEnabled && (fromOverride == nil || toOverride == nil) {
		return fmt.Errorf("BREAKS_BACKFILL=true requires BREAKS_FROM_DATE and BREAKS_TO_DATE (YYYY-MM-DD)")
	}
	if backfillEnabled && fromOverride.After(*toOverride) {
		return fmt.Errorf("breaks backfill invalid window: %s after %s",
			fromOverride.Format("2006-01-02"), toOverride.Format("2006-01-02"))
	}

	lg.Printf("🧭 Rolling window: %s → %s (back=%dd forward=%dd)",
		rollingStart.Format("2006-01-02"),
		rollingEnd.Format("2006-01-02"),
		backDays,
		forwardDays,
	)

	if backfillEnabled {
		lg.Printf("🧱 Backfill window: %s → %s",
			fromOverride.Format("2006-01-02"),
			toOverride.Format("2006-01-02"),
		)
	}

	for _, br := range r.Cfg.Branches {
		branchID := br.BranchID

		// Always do rolling window
		if err := scanBreaksRange(ctx, client, repo, branchID, rollingStart, rollingEnd, lg); err != nil {
			return err
		}

		// Optional backfill
		if backfillEnabled {
			if err := scanBreaksRange(ctx, client, repo, branchID, *fromOverride, *toOverride, lg); err != nil {
				return err
			}
		}

		lg.Printf("✅ breaks_api/%s: done", branchID)
	}

	return nil
}

func scanBreaksRange(
	ctx context.Context,
	c *BreaksAPIClient,
	repo *repos.BreaksAPIRepo,
	branchID string,
	start, end time.Time,
	lg *log.Logger,
) error {
	if start.After(end) {
		return nil
	}

	lg.Printf("📆 breaks_api/%s: scanning %s → %s",
		branchID,
		start.Format("2006-01-02"),
		end.Format("2006-01-02"),
	)

	windowStart := start
	safety := 0

	for !windowStart.After(end) {
		safety++
		if safety > 500 {
			return fmt.Errorf("breaks_api/%s: safety stop tripped start=%s end=%s",
				branchID, start.Format("2006-01-02"), end.Format("2006-01-02"))
		}

		windowEnd := endOfMonth(windowStart)
		if windowEnd.After(end) {
			windowEnd = end
		}

		page := 0
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			rows, totalPages, err := c.FetchBreaksPage(
				ctx,
				branchID,
				windowStart,
				windowEnd,
				page,
				100,           // size (max 100)
				nil, nil, nil, // staff_id, room_id, machine_id filters (unused here)
			)
			if err != nil {
				return fmt.Errorf("fetch breaks branch=%s win=%s..%s page=%d: %w",
					branchID, windowStart.Format("2006-01-02"), windowEnd.Format("2006-01-02"), page, err)
			}

			if len(rows) > 0 {
				if err := repo.UpsertBatch(rows, 500); err != nil {
					return fmt.Errorf("upsert breaks branch=%s page=%d: %w", branchID, page, err)
				}
			}

			page++
			// stop conditions
			if totalPages > 0 && page >= totalPages {
				break
			}
			if len(rows) == 0 {
				break
			}
		}

		windowStart = firstDayOfNextMonth(windowStart)
	}

	return nil
}
