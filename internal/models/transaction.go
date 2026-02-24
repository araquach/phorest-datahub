package models

import "time"

// Transaction is the header record for a Phorest sale.
// Keep only columns that are constant across all rows sharing the same transaction_id.
type Transaction struct {
	ID              uint       `gorm:"primaryKey"`
	TransactionID   string     `gorm:"column:transaction_id;uniqueIndex"`
	BranchID        string     `gorm:"column:branch_id;index"`
	BranchName      string     `gorm:"column:branch_name"`
	ClientID        string     `gorm:"column:client_id;index"`
	ClientFirstName string     `gorm:"column:client_first_name"`
	ClientLastName  string     `gorm:"column:client_last_name"`
	ClientSource    string     `gorm:"column:client_source"`
	PurchasedDate   *time.Time `gorm:"column:purchased_date;type:date"`
	PurchaseTime    *time.Time `gorm:"column:purchase_time;type:time without time zone"`

	// CSV column 'purchase_updated_at' → stored as updated_at_phorest for clarity / delta-sync
	UpdatedAtPhorest *time.Time `gorm:"column:updated_at_phorest;type:timestamptz;index"`

	// Relation
	Items []TransactionItem `gorm:"foreignKey:TransactionID;references:TransactionID"`

	// GORM metadata
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (Transaction) TableName() string { return "raw.transactions" }
