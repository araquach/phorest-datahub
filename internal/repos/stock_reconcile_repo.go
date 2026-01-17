package repos

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type PKStockRow struct {
	TransactionItemID string
	Barcode           string
	Quantity          int
	StaffID           string
	PhysicalBranchID  sql.NullString
	UpdatedAtPhorest  time.Time
	PurchasedAt       sql.NullTime
}

type StockVirtualTransfer struct {
	TransactionItemID string
	FromBranchID      string
	ToBranchID        string
	Barcode           string
	Quantity          int
}

type StockVirtualTransferRow struct {
	TransactionItemID string
	FromBranchID      string
	ToBranchID        string
	Barcode           string
	Quantity          int
}

type StockReconcileRepo struct {
	DB *sql.DB
}

func (r *StockReconcileRepo) FetchUnprocessedPKItems(
	ctx context.Context,
	pkBranchID string,
	fromTS, toTS time.Time,
	limit int,
	testBarcode string,
) ([]PKStockRow, error) {

	const q = `
SELECT
  ti.transaction_item_id,
  ti.product_barcode AS barcode,
  ti.quantity::int   AS quantity,
  ti.staff_id,
  spbo.physical_branch_id,
  ti.updated_at_phorest,
  (t.purchased_date::date + t.purchase_time::time) AS purchased_at
FROM transactions t
JOIN transaction_items ti ON ti.transaction_id = t.transaction_id
LEFT JOIN staff_physical_branch_overrides spbo
  ON spbo.staff_id = ti.staff_id AND spbo.active = true
WHERE t.branch_id = $1
  AND ti.product_barcode IS NOT NULL
  AND ti.product_barcode <> ''
  AND ti.quantity > 0
  AND ti.updated_at_phorest >= $2
  AND ti.updated_at_phorest <  $3
  AND ($5 = '' OR ti.product_barcode = $5)
  AND NOT EXISTS (
    SELECT 1
    FROM stock_virtual_transfers svt
    WHERE svt.transaction_item_id = ti.transaction_item_id
  )
  AND NOT EXISTS (
    SELECT 1
    FROM stock_virtual_transfer_exceptions svte
    WHERE svte.transaction_item_id = ti.transaction_item_id
  )
ORDER BY ti.updated_at_phorest ASC
LIMIT $4;
`

	rows, err := r.DB.QueryContext(ctx, q, pkBranchID, fromTS, toTS, limit, testBarcode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []PKStockRow
	for rows.Next() {
		var r PKStockRow
		if err := rows.Scan(
			&r.TransactionItemID,
			&r.Barcode,
			&r.Quantity,
			&r.StaffID,
			&r.PhysicalBranchID,
			&r.UpdatedAtPhorest,
			&r.PurchasedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, r)
	}

	return out, rows.Err()
}

func (r *StockReconcileRepo) InsertStockVirtualTransfers(
	ctx context.Context,
	transfers []StockVirtualTransferRow,
) error {
	if len(transfers) == 0 {
		return nil
	}

	const q = `
INSERT INTO stock_virtual_transfers (
  transaction_item_id,
  processed_at,
  from_branch_id,
  to_branch_id,
  barcode,
  quantity
) VALUES (
  $1, now(), $2, $3, $4, $5
)
ON CONFLICT (transaction_item_id) DO NOTHING;
`

	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.PrepareContext(ctx, q)
	if err != nil {
		return fmt.Errorf("prepare: %w", err)
	}
	defer stmt.Close()

	for _, t := range transfers {
		if t.FromBranchID == "" || t.ToBranchID == "" || t.Barcode == "" || t.Quantity <= 0 {
			return fmt.Errorf("invalid transfer row: %+v", t)
		}
		if _, err := stmt.ExecContext(ctx,
			t.TransactionItemID,
			t.FromBranchID,
			t.ToBranchID,
			t.Barcode,
			t.Quantity,
		); err != nil {
			return fmt.Errorf("insert transfer item_id=%s: %w", t.TransactionItemID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

func (r *StockReconcileRepo) InsertStockVirtualTransferExceptions(
	ctx context.Context,
	rows []PKStockRow,
	reason string,
	// optional extras if you can provide them (otherwise pass empty strings)
	productNameByBarcode map[string]string,
	staffNameByID map[string][2]string, // [first,last]
) error {
	if len(rows) == 0 {
		return nil
	}
	if reason == "" {
		reason = "UNMAPPED_STAFF"
	}

	const q = `
INSERT INTO stock_virtual_transfer_exceptions (
  transaction_item_id,
  reason,
  purchased_at,
  product_barcode,
  product_name,
  staff_id,
  staff_first_name,
  staff_last_name
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8
)
ON CONFLICT (transaction_item_id) DO NOTHING;
`

	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.PrepareContext(ctx, q)
	if err != nil {
		return fmt.Errorf("prepare: %w", err)
	}
	defer stmt.Close()

	for _, row := range rows {
		var purchasedAt any = nil
		if row.PurchasedAt.Valid {
			purchasedAt = row.PurchasedAt.Time
		}

		productName := ""
		if productNameByBarcode != nil {
			productName = productNameByBarcode[row.Barcode]
		}

		first, last := "", ""
		if staffNameByID != nil {
			if nm, ok := staffNameByID[row.StaffID]; ok {
				first, last = nm[0], nm[1]
			}
		}

		if _, err := stmt.ExecContext(
			ctx,
			row.TransactionItemID,
			reason,
			purchasedAt,
			row.Barcode,
			productName,
			row.StaffID,
			first,
			last,
		); err != nil {
			return fmt.Errorf("insert exception item_id=%s: %w", row.TransactionItemID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}
