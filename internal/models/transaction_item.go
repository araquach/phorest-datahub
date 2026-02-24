package models

import "time"

// TransactionItem represents one CSV row (service/product/etc.) within a sale.
// It carries all per-line values including money, staff, appointment, payment, flags.
type TransactionItem struct {
	ID                uint   `gorm:"primaryKey"`
	TransactionItemID string `gorm:"column:transaction_item_id;uniqueIndex"`
	TransactionID     string `gorm:"column:transaction_id;index"`

	// Echoed header (some exports repeat these per row; we keep them line-scoped for correctness)
	BranchID        string     `gorm:"column:branch_id;index"`
	BranchName      string     `gorm:"column:branch_name"`
	ClientID        string     `gorm:"column:client_id;index"`
	ClientFirstName string     `gorm:"column:client_first_name"`
	ClientLastName  string     `gorm:"column:client_last_name"`
	ClientSource    string     `gorm:"column:client_source"`
	PurchasedDate   *time.Time `gorm:"column:purchased_date;type:date"`
	PurchaseTime    *time.Time `gorm:"column:purchase_time;type:time without time zone"`

	// Item meta / classification
	ItemType    string  `gorm:"column:item_type;index"` // SERVICE, PRODUCT, etc.
	Description string  `gorm:"column:description"`
	Quantity    float64 `gorm:"column:quantity"`

	// Discounts at purchase level (as echoed per-line in CSV)
	PurchaseVoucherDiscountPercentage float64 `gorm:"column:purchase_voucher_discount_percentage"`
	PurchaseOnlineDeposit             float64 `gorm:"column:purchase_online_deposit"`
	PurchaseOnlineDiscountAmount      float64 `gorm:"column:purchase_online_discount_amount"`

	// Service fields
	ServiceID           string `gorm:"column:service_id"`
	ServiceName         string `gorm:"column:service_name"`
	ServiceCategoryID   string `gorm:"column:service_category_id"`
	ServiceCategoryName string `gorm:"column:service_category_name"`

	// Package / Special offer
	PackageID        string `gorm:"column:package_id"`
	PackageName      string `gorm:"column:package_name"`
	SpecialOfferID   string `gorm:"column:special_offer_id"`
	SpecialOfferName string `gorm:"column:special_offer_name"`

	// Product fields
	ProductID           string `gorm:"column:product_id"`
	ProductName         string `gorm:"column:product_name"`
	ProductBrandID      string `gorm:"column:product_brand_id"`
	ProductBrandName    string `gorm:"column:product_brand_name"`
	ProductCategoryID   string `gorm:"column:product_category_id"`
	ProductCategoryName string `gorm:"column:product_category_name"`
	ProductBarcode      string `gorm:"column:product_barcode"`
	ProductCode         string `gorm:"column:product_code"`

	// Courses / vouchers / rewards
	CourseID          string `gorm:"column:course_id"`
	CourseName        string `gorm:"column:course_name"`
	ClientCourseName  string `gorm:"column:client_course_name"`
	VoucherSerial     string `gorm:"column:voucher_serial"`
	ServiceRewardID   string `gorm:"column:service_reward_id"`
	ServiceRewardName string `gorm:"column:service_reward_name"`
	ProductRewardID   string `gorm:"column:product_reward_id"`
	ProductRewardName string `gorm:"column:product_reward_name"`

	// Monetary (per line)
	UnitPrice                      float64 `gorm:"column:unit_price"`
	OriginalPrice                  float64 `gorm:"column:original_price"`
	DiscountType                   float64 `gorm:"column:discount_type"`
	DiscountValue                  float64 `gorm:"column:discount_value"`
	ItemOnlineDeposit              float64 `gorm:"column:item_online_deposit"`
	ItemOnlineDiscount             float64 `gorm:"column:item_online_discount"`
	LoyaltyPointsAwarded           float64 `gorm:"column:loyalty_points_awarded"`
	TaxRate                        float64 `gorm:"column:tax_rate"`
	TotalAmount                    float64 `gorm:"column:total_amount"`
	TotalAmountPreVouchDisc        float64 `gorm:"column:total_amount_pre_vouch_disc"`
	NetTotalAmount                 float64 `gorm:"column:net_total_amount"`
	GrossTotalAmount               float64 `gorm:"column:gross_total_amount"`
	NetPrice                       float64 `gorm:"column:net_price"`
	GrossPrice                     float64 `gorm:"column:gross_price"`
	DiscountAmount                 float64 `gorm:"column:discount_amount"`
	TaxAmount                      float64 `gorm:"column:tax_amount"`
	StaffTips                      float64 `gorm:"column:staff_tips"`
	ProductCostPrice               float64 `gorm:"column:product_cost_price"`
	ServiceCost                    float64 `gorm:"column:service_cost"`
	ServiceCostType                string  `gorm:"column:service_cost_type"`
	GrossTotalWithDiscount         float64 `gorm:"column:gross_total_with_discount"`
	GrossTotalWithDiscountMinusTax float64 `gorm:"column:gross_total_with_discount_minus_tax"`
	SimpleDiscountAmount           float64 `gorm:"column:simple_discount_amount"`
	MembershipBenefitUsed          int     `gorm:"column:membership_benefit_used"`
	MembershipDiscountAmount       float64 `gorm:"column:membership_discount_amount"`
	Deal                           float64 `gorm:"column:deal"`
	SessionNetAmount               float64 `gorm:"column:session_net_amount"`
	SessionGrossAmount             float64 `gorm:"column:session_gross_amount"`
	PhorestTips                    float64 `gorm:"column:phorest_tips"`

	// Payment details (often echoed per line)
	PaymentType                  string  `gorm:"column:payment_type"`
	PaymentTypeIDs               string  `gorm:"column:payment_type_ids"`
	PaymentTypeAmounts           float64 `gorm:"column:payment_type_amounts"`
	PaymentTypeCodes             string  `gorm:"column:payment_type_codes"`
	PaymentTypeNames             string  `gorm:"column:payment_type_names"`
	PaymentTypeVoucherSerials    string  `gorm:"column:payment_type_voucher_serials"`
	PaymentTypePrepaidTaxAmounts string  `gorm:"column:payment_type_prepaid_tax_amounts"`

	// Flags / types / misc
	OutstandingBalancePMT int64  `gorm:"column:outstanding_balance_pmt"`
	OpenSale              bool   `gorm:"column:open_sale"`
	OpenSaleType          string `gorm:"column:open_sale_type"`
	PurchaseType          string `gorm:"column:purchase_type"`
	OnlineBooking         int64  `gorm:"column:online_booking"`
	Void                  int64  `gorm:"column:void"`
	VoidedTransactionID   string `gorm:"column:voided_transaction_id"`
	VoidReason            string `gorm:"column:void_reason"`

	// Department
	DepartmentID   string `gorm:"column:department_id"`
	DepartmentName string `gorm:"column:department_name"`

	// Staff per-line
	StaffID            string `gorm:"column:staff_id;index"`
	StaffFirstName     string `gorm:"column:staff_first_name"`
	StaffLastName      string `gorm:"column:staff_last_name"`
	StaffCategoryID    string `gorm:"column:staff_category_id"`
	StaffCategoryName  string `gorm:"column:staff_category_name"`
	IsRequestedStaff   int    `gorm:"column:is_requested_staff"`
	PrimaryStaffID     string `gorm:"column:primary_staff_id"`
	PreferredStaffID   string `gorm:"column:preferred_staff_id"`
	PreferredStaffName string `gorm:"column:preferred_staff_name"`

	// Appointment linkage (varies by line)
	AppointmentID      string     `gorm:"column:appointment_id;index"`
	AppointmentDate    *time.Time `gorm:"column:appointment_date;type:date"`
	AppointmentCreated *time.Time `gorm:"column:appointment_created;type:timestamptz"`
	AppointmentRating  int64      `gorm:"column:appointment_rating"`

	// Client echoes (line-scoped in CSV)
	ClientBirthday   *time.Time `gorm:"column:client_birthday;type:date"`
	ClientGender     string     `gorm:"column:client_gender"`
	ClientEmail      string     `gorm:"column:client_email"`
	ClientFirstVisit *time.Time `gorm:"column:client_first_visit;type:date"`

	// “Appointment client” echoes
	ApptClientID         string     `gorm:"column:appt_client_id"`
	ApptClientFirstName  string     `gorm:"column:appt_client_first_name"`
	ApptClientLastName   string     `gorm:"column:appt_client_last_name"`
	ApptClientBirthday   *time.Time `gorm:"column:appt_client_birthday;type:date"`
	ApptClientGender     string     `gorm:"column:appt_client_gender"`
	ApptClientEmail      string     `gorm:"column:appt_client_email"`
	ApptClientFirstVisit *time.Time `gorm:"column:appt_client_first_visit;type:date"`

	// Internet / categories / misc ids
	InternetCategoryIDs   string `gorm:"column:internet_category_ids"`
	InternetCategoryNames string `gorm:"column:internet_category_names"`
	BranchProductID       string `gorm:"column:branch_product_id"`
	FixedDiscountID       string `gorm:"column:fixed_discount_id"`
	FixedDiscountName     string `gorm:"column:fixed_discount_name"`
	ClientCourseID        string `gorm:"column:client_course_id"`
	CreatingUser          string `gorm:"column:creating_user"`
	TaxRateName           string `gorm:"column:tax_rate_name"`
	SaleFeeID             string `gorm:"column:sale_fee_id"`

	// Sync watermark from CSV (purchase_updated_at) → stored as updated_at_phorest
	UpdatedAtPhorest *time.Time `gorm:"column:updated_at_phorest;type:timestamptz;index"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (TransactionItem) TableName() string { return "raw.transaction_items" }
