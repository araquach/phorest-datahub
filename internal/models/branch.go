package models

import "time"

type Branch struct {
	ID           uint     `gorm:"primaryKey"`
	BranchID     string   `gorm:"column:branch_id;not null;uniqueIndex"`
	Name         string   `gorm:"column:name"`
	TimeZone     string   `gorm:"column:time_zone"`
	Latitude     *float64 `gorm:"column:latitude"`
	Longitude    *float64 `gorm:"column:longitude"`
	Street1      string   `gorm:"column:street_address_1"`
	Street2      string   `gorm:"column:street_address_2"`
	City         string   `gorm:"column:city"`
	State        string   `gorm:"column:state"`
	PostalCode   string   `gorm:"column:postal_code"`
	Country      string   `gorm:"column:country"`
	CurrencyCode string   `gorm:"column:currency_code"`
	AccountID    *int64   `gorm:"column:account_id"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (Branch) TableName() string { return "raw.branches" }
