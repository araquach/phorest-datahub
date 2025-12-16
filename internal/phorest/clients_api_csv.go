package phorest

import (
	"encoding/csv"
	"os"
	"strconv"
	"time"

	"github.com/araquach/phorest-datahub/internal/models"
)

func writeClientsAPICSV(path string, rows []models.ClientAPI) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	header := []string{
		"client_id",
		"version",
		"first_name",
		"last_name",
		"mobile",
		"email",
		"street_address_1",
		"street_address_2",
		"city",
		"postal_code",
		"country",
		"birth_date",
		"client_since",
		"gender",
		"notes",
		"loyalty_card_serial",
		"loyalty_points",
		"credit_outstanding_balance",
		"credit_days",
		"credit_limit",
		"updated_at_phorest",
	}
	if err := w.Write(header); err != nil {
		return err
	}

	formatDate := func(t *time.Time) string {
		if t == nil {
			return ""
		}
		return t.UTC().Format(time.RFC3339)
	}

	for _, c := range rows {
		var lp, bal, lim, days string
		if c.LoyaltyPoints != nil {
			lp = strconv.FormatFloat(*c.LoyaltyPoints, 'f', 2, 64)
		}
		if c.CreditOutstandingBalance != nil {
			bal = strconv.FormatFloat(*c.CreditOutstandingBalance, 'f', 2, 64)
		}
		if c.CreditLimit != nil {
			lim = strconv.FormatFloat(*c.CreditLimit, 'f', 2, 64)
		}
		if c.CreditDays != nil {
			days = strconv.FormatInt(*c.CreditDays, 10)
		}

		rec := []string{
			c.ClientID,
			strconv.FormatInt(c.Version, 10),
			c.FirstName,
			c.LastName,
			c.Mobile,
			c.Email,
			c.StreetAddress1,
			c.StreetAddress2,
			c.City,
			c.PostalCode,
			c.Country,
			"", // birth_date (you can format if you like)
			formatDate(c.ClientSince),
			c.Gender,
			c.Notes,
			c.LoyaltyCardSerial,
			lp,
			bal,
			days,
			lim,
			formatDate(c.UpdatedAtPhorest),
		}
		if err := w.Write(rec); err != nil {
			return err
		}
	}

	return w.Error()
}
