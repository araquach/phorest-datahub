package models

import "time"

type ClientAPI struct {
	ID                 int64  `gorm:"primaryKey;column:id"`
	ClientID           string `gorm:"column:client_id"`
	Version            int64  `gorm:"column:version"`
	FirstName          string `gorm:"column:first_name"`
	LastName           string `gorm:"column:last_name"`
	Mobile             string `gorm:"column:mobile"`
	LinkedClientMobile string `gorm:"column:linked_client_mobile"`
	LandLine           string `gorm:"column:land_line"`
	Email              string `gorm:"column:email"`

	StreetAddress1 string `gorm:"column:street_address_1"`
	StreetAddress2 string `gorm:"column:street_address_2"`
	City           string `gorm:"column:city"`
	State          string `gorm:"column:state"`
	PostalCode     string `gorm:"column:postal_code"`
	Country        string `gorm:"column:country"`

	BirthDate   *time.Time `gorm:"column:birth_date"`
	ClientSince *time.Time `gorm:"column:client_since"`
	Gender      string     `gorm:"column:gender"`
	Notes       string     `gorm:"column:notes"`

	SMSMarketingConsent   bool `gorm:"column:sms_marketing_consent"`
	EmailMarketingConsent bool `gorm:"column:email_marketing_consent"`
	SMSReminderConsent    bool `gorm:"column:sms_reminder_consent"`
	EmailReminderConsent  bool `gorm:"column:email_reminder_consent"`

	PreferredStaffID string `gorm:"column:preferred_staff_id"`
	ExternalID       string `gorm:"column:external_id"`
	CreatingBranchID string `gorm:"column:creating_branch_id"`

	Archived         bool   `gorm:"column:archived"`
	Banned           bool   `gorm:"column:banned"`
	MergedToClientID string `gorm:"column:merged_to_client_id"`
	Deleted          bool   `gorm:"column:deleted"`

	ClientCategoryIDs string `gorm:"column:client_category_ids"`

	FirstVisit *time.Time `gorm:"column:first_visit"`
	LastVisit  *time.Time `gorm:"column:last_visit"`

	CreatedAtPhorest *time.Time `gorm:"column:created_at_phorest"`
	UpdatedAtPhorest *time.Time `gorm:"column:updated_at_phorest"`

	PhotoURL string `gorm:"column:photo_url"`

	LoyaltyCardSerial string   `gorm:"column:loyalty_card_serial"`
	LoyaltyPoints     *float64 `gorm:"column:loyalty_points"`

	CreditOutstandingBalance *float64 `gorm:"column:credit_outstanding_balance"`
	CreditDays               *int64   `gorm:"column:credit_days"`
	CreditLimit              *float64 `gorm:"column:credit_limit"`

	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (ClientAPI) TableName() string { return "clients_api" }
