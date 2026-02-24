package models

import "time"

type PhProduct struct {
	ID              string     `gorm:"column:id;primaryKey"` // productId
	ParentID        *string    `gorm:"column:parent_id"`
	Name            string     `gorm:"column:name"`
	BrandID         *string    `gorm:"column:brand_id"`
	BrandName       *string    `gorm:"column:brand_name"`
	CategoryID      *string    `gorm:"column:category_id"`
	CategoryName    *string    `gorm:"column:category_name"`
	Code            *string    `gorm:"column:code"`
	TypeRaw         *string    `gorm:"column:type_raw"` // e.g. "RETAIL, COLOUR, PROFESSIONAL"
	MeasurementQty  *float64   `gorm:"column:measurement_qty"`
	MeasurementUnit *string    `gorm:"column:measurement_unit"`
	Archived        bool       `gorm:"column:archived"`
	CreatedAtPh     *time.Time `gorm:"column:created_at_ph"`
	UpdatedAtPh     *time.Time `gorm:"column:updated_at_ph"`
	InsertedAt      time.Time  `gorm:"column:inserted_at;autoCreateTime"`
	UpdatedAt       time.Time  `gorm:"column:updated_at;autoUpdateTime"`
}

func (PhProduct) TableName() string {
	return "raw.ph_products"
}

type PhProductStock struct {
	ID              int64      `gorm:"column:id;primaryKey;autoIncrement"`
	ProductID       string     `gorm:"column:product_id;not null"`
	BranchID        string     `gorm:"column:branch_id;not null"`
	Price           *float64   `gorm:"column:price"` // retail price
	MinQuantity     *float64   `gorm:"column:min_quantity"`
	MaxQuantity     *float64   `gorm:"column:max_quantity"`
	QuantityInStock *float64   `gorm:"column:quantity_in_stock"`
	ReorderCount    *float64   `gorm:"column:reorder_count"`
	ReorderCost     *float64   `gorm:"column:reorder_cost"` // cost price
	Archived        bool       `gorm:"column:archived"`
	CreatedAtPh     *time.Time `gorm:"column:created_at_ph"`
	UpdatedAtPh     *time.Time `gorm:"column:updated_at_ph"`
	LastSyncedAt    time.Time  `gorm:"column:last_synced_at"`

	// Optional: relation to product if you want eager loading
	Product *PhProduct `gorm:"foreignKey:ProductID;references:ID"`
}

func (PhProductStock) TableName() string {
	return "raw.ph_product_stock"
}

type PhProductStockHistory struct {
	ID              int64     `gorm:"column:id;primaryKey;autoIncrement"`
	ProductID       string    `gorm:"column:product_id;not null"`
	BranchID        string    `gorm:"column:branch_id;not null"`
	SnapshotTime    time.Time `gorm:"column:snapshot_time"`
	QuantityInStock *float64  `gorm:"column:quantity_in_stock"`
	Price           *float64  `gorm:"column:price"`
	Source          string    `gorm:"column:source"` // 'sync', 'manual', etc.
}

func (PhProductStockHistory) TableName() string {
	return "raw.ph_product_stock_history"
}
