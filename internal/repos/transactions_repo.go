package repos

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/araquach/phorest-datahub/internal/models"
	"gorm.io/gorm"
)

type TransactionsRepo struct {
	db *gorm.DB
	lg *log.Logger
}

func NewTransactionsRepo(db *gorm.DB, lg *log.Logger) *TransactionsRepo {
	return &TransactionsRepo{db: db, lg: lg}
}

func (r *TransactionsRepo) UpsertBatch(rows []models.Transaction, batchSize int) error {
	if len(rows) == 0 {
		return nil
	}
	now := time.Now().UTC()

	cols := []string{
		"transaction_id", "branch_id", "branch_name",
		"client_id", "client_first_name", "client_last_name", "client_source",
		"purchased_date", "purchase_time",
		"updated_at_phorest",
		"created_at", "updated_at",
	}

	placeholders := make([]string, 0, len(rows))
	args := make([]any, 0, len(rows)*len(cols))

	flush := func() error {
		if len(placeholders) == 0 {
			return nil
		}
		sql := fmt.Sprintf(`
INSERT INTO raw.transactions AS t (%s)
VALUES %s
ON CONFLICT (transaction_id) DO UPDATE SET
  branch_id = EXCLUDED.branch_id,
  branch_name = EXCLUDED.branch_name,
  client_id = EXCLUDED.client_id,
  client_first_name = EXCLUDED.client_first_name,
  client_last_name = EXCLUDED.client_last_name,
  client_source = EXCLUDED.client_source,
  purchased_date = EXCLUDED.purchased_date,
  purchase_time = EXCLUDED.purchase_time,
  updated_at_phorest = EXCLUDED.updated_at_phorest,
  updated_at = EXCLUDED.updated_at
WHERE t.updated_at_phorest IS NULL
   OR EXCLUDED.updated_at_phorest > t.updated_at_phorest;`,
			strings.Join(cols, ", "),
			strings.Join(placeholders, ","),
		)
		if err := r.db.Exec(sql, args...).Error; err != nil {
			return err
		}
		r.lg.Printf("Upserted transactions: %d", len(placeholders))
		placeholders = placeholders[:0]
		args = args[:0]
		return nil
	}

	for _, row := range rows {
		placeholders = append(placeholders, "("+strings.Repeat("?,", len(cols)-1)+"?)")
		args = append(args,
			row.TransactionID, row.BranchID, row.BranchName,
			row.ClientID, row.ClientFirstName, row.ClientLastName, row.ClientSource,
			row.PurchasedDate, row.PurchaseTime,
			row.UpdatedAtPhorest,
			now, now,
		)
		if len(placeholders) >= batchSize {
			if err := flush(); err != nil {
				return err
			}
		}
	}
	return flush()
}
