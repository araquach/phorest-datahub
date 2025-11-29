package phorest

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"staff-appraisals/internal/models"
)

// ParsedReviews wraps the slice for symmetry with other parsers
type ParsedReviews struct {
	Reviews []models.Review
}

// ParseReviewsCSV reads a reviews CSV we’ve previously written and converts it
// into []models.Review ready to upsert.
func ParseReviewsCSV(path string, lg *log.Logger) (*ParsedReviews, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open reviews csv %q: %w", path, err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.ReuseRecord = true

	header, err := r.Read()
	if err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}

	idx := make(map[string]int, len(header))
	for i, h := range header {
		h = strings.TrimSpace(strings.ToLower(h))
		idx[h] = i
	}

	get := func(rec []string, name string) string {
		i, ok := idx[name]
		if !ok || i >= len(rec) {
			return ""
		}
		return rec[i]
	}

	parseDate := func(s string) *time.Time {
		s = strings.TrimSpace(s)
		if s == "" {
			return nil
		}
		for _, layout := range []string{"2006-01-02", time.RFC3339, "2006-01-02T15:04:05"} {
			if t, err := time.Parse(layout, s); err == nil {
				return &t
			}
		}
		return nil
	}

	parseInt := func(s string) int {
		s = strings.TrimSpace(s)
		if s == "" {
			return 0
		}
		v, err := strconv.Atoi(s)
		if err != nil {
			return 0
		}
		return v
	}

	parseBool := func(s string) bool {
		s = strings.TrimSpace(strings.ToLower(s))
		switch s {
		case "1", "true", "t", "yes", "y":
			return true
		default:
			return false
		}
	}

	var out []models.Review
	row := 0

	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read row %d: %w", row, err)
		}
		row++

		reviewID := get(rec, "review_id")
		if reviewID == "" {
			continue
		}

		branchID := get(rec, "branch_id")
		if branchID == "" {
			// we *expect* branch_id, but skip rather than failing hard
			continue
		}

		rv := models.Review{
			ReviewID:        reviewID,
			BranchID:        branchID,
			ClientID:        get(rec, "client_id"),
			ClientFirstName: get(rec, "client_first_name"),
			ClientLastName:  get(rec, "client_last_name"),
			ReviewDate:      parseDate(get(rec, "review_date")),
			VisitDate:       parseDate(get(rec, "visit_date")),
			StaffID:         get(rec, "staff_id"),
			StaffFirstName:  get(rec, "staff_first_name"),
			StaffLastName:   get(rec, "staff_last_name"),
			Text:            get(rec, "text"),
			Rating:          parseInt(get(rec, "rating")),
			FacebookReview:  parseBool(get(rec, "facebook_review")),
			TwitterReview:   parseBool(get(rec, "twitter_review")),
		}

		out = append(out, rv)
	}

	lg.Printf("Parsed reviews CSV %s: %d reviews", path, len(out))
	return &ParsedReviews{Reviews: out}, nil
}
