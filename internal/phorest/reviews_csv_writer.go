package phorest

import (
	"encoding/csv"
	"fmt"
	"os"
	"time"

	"staff-appraisals/internal/models"
)

func writeReviewsCSV(path string, rows []models.Review) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create reviews csv %q: %w", path, err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	// Must match ParseReviewsCSV headers (all lowercase, snake_case)
	header := []string{
		"review_id",
		"branch_id",
		"client_id",
		"client_first_name",
		"client_last_name",
		"review_date",
		"visit_date",
		"staff_id",
		"staff_first_name",
		"staff_last_name",
		"text",
		"rating",
		"facebook_review",
		"twitter_review",
	}

	if err := w.Write(header); err != nil {
		return fmt.Errorf("write header: %w", err)
	}

	dateFmt := func(t *time.Time) string {
		if t == nil {
			return ""
		}
		return t.Format("2006-01-02")
	}

	for _, rv := range rows {
		rec := []string{
			rv.ReviewID,
			rv.BranchID,
			rv.ClientID,
			rv.ClientFirstName,
			rv.ClientLastName,
			dateFmt(rv.ReviewDate),
			dateFmt(rv.VisitDate),
			rv.StaffID,
			rv.StaffFirstName,
			rv.StaffLastName,
			rv.Text,
			fmt.Sprintf("%d", rv.Rating),
			fmt.Sprintf("%t", rv.FacebookReview),
			fmt.Sprintf("%t", rv.TwitterReview),
		}

		if err := w.Write(rec); err != nil {
			return fmt.Errorf("write row for review %s: %w", rv.ReviewID, err)
		}
	}

	return nil
}
