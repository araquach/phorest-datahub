package models

import "time"

type AppointmentAPI struct {
	ID int64 `gorm:"primaryKey;column:id"`

	BranchID      string `gorm:"column:branch_id"`
	AppointmentID string `gorm:"column:appointment_id"`
	Version       int64  `gorm:"column:version"`

	AppointmentDate time.Time `gorm:"column:appointment_date"`
	StartTime       string    `gorm:"column:start_time"`
	EndTime         string    `gorm:"column:end_time"`

	Price         float64  `gorm:"column:price"`
	DepositAmount *float64 `gorm:"column:deposit_amount"`

	StaffID          string     `gorm:"column:staff_id"`
	Confirmed        bool       `gorm:"column:confirmed"`
	ServiceID        string     `gorm:"column:service_id"`
	CreatedAtPhorest *time.Time `gorm:"column:created_at_phorest"`
	UpdatedAtPhorest *time.Time `gorm:"column:updated_at_phorest"`

	StaffRequest   bool   `gorm:"column:staff_request"`
	PreferredStaff bool   `gorm:"column:preferred_staff"`
	ClientID       string `gorm:"column:client_id"`

	ServiceRewardID    string `gorm:"column:service_reward_id"`
	PurchasingBranchID string `gorm:"column:purchasing_branch_id"`
	ServiceName        string `gorm:"column:service_name"`

	State           string `gorm:"column:state"`
	ActivationState string `gorm:"column:activation_state"`

	DepositDateTime *time.Time `gorm:"column:deposit_datetime"`
	BookingID       string     `gorm:"column:booking_id"`
	Source          string     `gorm:"column:source"`

	Deleted bool `gorm:"column:deleted"`

	// Stored as JSON string (you chose to keep it; we’ll store losslessly)
	InternetServiceCategories string `gorm:"column:internet_service_categories"`

	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (AppointmentAPI) TableName() string { return "appointments_api" }
