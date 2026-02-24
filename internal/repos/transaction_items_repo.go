package repos

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/araquach/phorest-datahub/internal/models"
	"gorm.io/gorm"
)

type ItemsRepo struct {
	db *gorm.DB
	lg *log.Logger
}

func NewItemsRepo(db *gorm.DB, lg *log.Logger) *ItemsRepo {
	return &ItemsRepo{db: db, lg: lg}
}

func (r *ItemsRepo) UpsertBatch(rows []models.TransactionItem, batchSize int) error {
	if len(rows) == 0 {
		return nil
	}
	now := time.Now().UTC()

	// Column list matches your model tags; keep this long but explicit.
	cols := []string{
		"transaction_item_id", "transaction_id",
		"branch_id", "branch_name",
		"client_id", "client_first_name", "client_last_name", "client_source",
		"purchased_date", "purchase_time",
		"item_type", "description", "quantity",
		"purchase_voucher_discount_percentage", "purchase_online_deposit", "purchase_online_discount_amount",
		"service_id", "service_name", "service_category_id", "service_category_name",
		"package_id", "package_name", "special_offer_id", "special_offer_name",
		"product_id", "product_name", "product_brand_id", "product_brand_name",
		"product_category_id", "product_category_name", "product_barcode", "product_code",
		"course_id", "course_name", "client_course_name", "voucher_serial",
		"service_reward_id", "service_reward_name", "product_reward_id", "product_reward_name",
		"unit_price", "original_price", "discount_type", "discount_value",
		"item_online_deposit", "item_online_discount", "loyalty_points_awarded",
		"tax_rate", "total_amount", "total_amount_pre_vouch_disc", "net_total_amount",
		"gross_total_amount", "net_price", "gross_price", "discount_amount",
		"tax_amount", "staff_tips", "product_cost_price", "service_cost", "service_cost_type",
		"gross_total_with_discount", "gross_total_with_discount_minus_tax", "simple_discount_amount",
		"membership_benefit_used", "membership_discount_amount", "deal",
		"session_net_amount", "session_gross_amount", "phorest_tips",

		"payment_type", "payment_type_ids", "payment_type_amounts",
		"payment_type_codes", "payment_type_names", "payment_type_voucher_serials",
		"payment_type_prepaid_tax_amounts",

		"outstanding_balance_pmt", "open_sale", "open_sale_type", "purchase_type", "online_booking",
		"void", "voided_transaction_id", "void_reason",

		"department_id", "department_name",

		"staff_id", "staff_first_name", "staff_last_name",
		"staff_category_id", "staff_category_name", "is_requested_staff",
		"primary_staff_id", "preferred_staff_id", "preferred_staff_name",

		"appointment_id", "appointment_date", "appointment_created", "appointment_rating",

		"client_birthday", "client_gender", "client_email", "client_first_visit",

		"appt_client_id", "appt_client_first_name", "appt_client_last_name",
		"appt_client_birthday", "appt_client_gender", "appt_client_email", "appt_client_first_visit",

		"internet_category_ids", "internet_category_names", "branch_product_id",
		"fixed_discount_id", "fixed_discount_name", "client_course_id",
		"creating_user", "tax_rate_name", "sale_fee_id",

		"updated_at_phorest", "created_at", "updated_at",
	}

	placeholders := make([]string, 0, len(rows))
	args := make([]any, 0, len(rows)*len(cols))

	flush := func() error {
		if len(placeholders) == 0 {
			return nil
		}
		sql := fmt.Sprintf(`
INSERT INTO raw.transaction_items AS ti (%s)
VALUES %s
ON CONFLICT (transaction_item_id) DO UPDATE SET
  -- only the columns that should change on newer data:
  branch_id = EXCLUDED.branch_id,
  branch_name = EXCLUDED.branch_name,
  client_id = EXCLUDED.client_id,
  client_first_name = EXCLUDED.client_first_name,
  client_last_name = EXCLUDED.client_last_name,
  client_source = EXCLUDED.client_source,
  purchased_date = EXCLUDED.purchased_date,
  purchase_time = EXCLUDED.purchase_time,
  item_type = EXCLUDED.item_type,
  description = EXCLUDED.description,
  quantity = EXCLUDED.quantity,
  unit_price = EXCLUDED.unit_price,
  original_price = EXCLUDED.original_price,
  discount_type = EXCLUDED.discount_type,
  discount_value = EXCLUDED.discount_value,
  item_online_deposit = EXCLUDED.item_online_deposit,
  item_online_discount = EXCLUDED.item_online_discount,
  loyalty_points_awarded = EXCLUDED.loyalty_points_awarded,
  tax_rate = EXCLUDED.tax_rate,
  total_amount = EXCLUDED.total_amount,
  total_amount_pre_vouch_disc = EXCLUDED.total_amount_pre_vouch_disc,
  net_total_amount = EXCLUDED.net_total_amount,
  gross_total_amount = EXCLUDED.gross_total_amount,
  net_price = EXCLUDED.net_price,
  gross_price = EXCLUDED.gross_price,
  discount_amount = EXCLUDED.discount_amount,
  tax_amount = EXCLUDED.tax_amount,
  staff_tips = EXCLUDED.staff_tips,
  product_cost_price = EXCLUDED.product_cost_price,
  service_cost = EXCLUDED.service_cost,
  service_cost_type = EXCLUDED.service_cost_type,
  gross_total_with_discount = EXCLUDED.gross_total_with_discount,
  gross_total_with_discount_minus_tax = EXCLUDED.gross_total_with_discount_minus_tax,
  simple_discount_amount = EXCLUDED.simple_discount_amount,
  membership_benefit_used = EXCLUDED.membership_benefit_used,
  membership_discount_amount = EXCLUDED.membership_discount_amount,
  deal = EXCLUDED.deal,
  session_net_amount = EXCLUDED.session_net_amount,
  session_gross_amount = EXCLUDED.session_gross_amount,
  phorest_tips = EXCLUDED.phorest_tips,
  payment_type = EXCLUDED.payment_type,
  payment_type_ids = EXCLUDED.payment_type_ids,
  payment_type_amounts = EXCLUDED.payment_type_amounts,
  payment_type_codes = EXCLUDED.payment_type_codes,
  payment_type_names = EXCLUDED.payment_type_names,
  payment_type_voucher_serials = EXCLUDED.payment_type_voucher_serials,
  payment_type_prepaid_tax_amounts = EXCLUDED.payment_type_prepaid_tax_amounts,
  outstanding_balance_pmt = EXCLUDED.outstanding_balance_pmt,
  open_sale = EXCLUDED.open_sale,
  open_sale_type = EXCLUDED.open_sale_type,
  purchase_type = EXCLUDED.purchase_type,
  online_booking = EXCLUDED.online_booking,
  void = EXCLUDED.void,
  voided_transaction_id = EXCLUDED.voided_transaction_id,
  void_reason = EXCLUDED.void_reason,
  department_id = EXCLUDED.department_id,
  department_name = EXCLUDED.department_name,
  staff_id = EXCLUDED.staff_id,
  staff_first_name = EXCLUDED.staff_first_name,
  staff_last_name = EXCLUDED.staff_last_name,
  staff_category_id = EXCLUDED.staff_category_id,
  staff_category_name = EXCLUDED.staff_category_name,
  is_requested_staff = EXCLUDED.is_requested_staff,
  primary_staff_id = EXCLUDED.primary_staff_id,
  preferred_staff_id = EXCLUDED.preferred_staff_id,
  preferred_staff_name = EXCLUDED.preferred_staff_name,
  appointment_id = EXCLUDED.appointment_id,
  appointment_date = EXCLUDED.appointment_date,
  appointment_created = EXCLUDED.appointment_created,
  appointment_rating = EXCLUDED.appointment_rating,
  client_birthday = EXCLUDED.client_birthday,
  client_gender = EXCLUDED.client_gender,
  client_email = EXCLUDED.client_email,
  client_first_visit = EXCLUDED.client_first_visit,
  appt_client_id = EXCLUDED.appt_client_id,
  appt_client_first_name = EXCLUDED.appt_client_first_name,
  appt_client_last_name = EXCLUDED.appt_client_last_name,
  appt_client_birthday = EXCLUDED.appt_client_birthday,
  appt_client_gender = EXCLUDED.appt_client_gender,
  appt_client_email = EXCLUDED.appt_client_email,
  appt_client_first_visit = EXCLUDED.appt_client_first_visit,
  internet_category_ids = EXCLUDED.internet_category_ids,
  internet_category_names = EXCLUDED.internet_category_names,
  branch_product_id = EXCLUDED.branch_product_id,
  fixed_discount_id = EXCLUDED.fixed_discount_id,
  fixed_discount_name = EXCLUDED.fixed_discount_name,
  client_course_id = EXCLUDED.client_course_id,
  creating_user = EXCLUDED.creating_user,
  tax_rate_name = EXCLUDED.tax_rate_name,
  sale_fee_id = EXCLUDED.sale_fee_id,
  updated_at_phorest = EXCLUDED.updated_at_phorest,
  updated_at = EXCLUDED.updated_at
WHERE ti.updated_at_phorest IS NULL
   OR EXCLUDED.updated_at_phorest > ti.updated_at_phorest;`,
			strings.Join(cols, ", "),
			strings.Join(placeholders, ","),
		)
		if err := r.db.Exec(sql, args...).Error; err != nil {
			return err
		}
		r.lg.Printf("Upserted items: %d", len(placeholders))
		placeholders = placeholders[:0]
		args = args[:0]
		return nil
	}

	for _, it := range rows {
		placeholders = append(placeholders, "("+strings.Repeat("?,", len(cols)-1)+"?)")
		args = append(args,
			it.TransactionItemID, it.TransactionID,
			it.BranchID, it.BranchName,
			it.ClientID, it.ClientFirstName, it.ClientLastName, it.ClientSource,
			it.PurchasedDate, it.PurchaseTime,
			it.ItemType, it.Description, it.Quantity,
			it.PurchaseVoucherDiscountPercentage, it.PurchaseOnlineDeposit, it.PurchaseOnlineDiscountAmount,
			it.ServiceID, it.ServiceName, it.ServiceCategoryID, it.ServiceCategoryName,
			it.PackageID, it.PackageName, it.SpecialOfferID, it.SpecialOfferName,
			it.ProductID, it.ProductName, it.ProductBrandID, it.ProductBrandName,
			it.ProductCategoryID, it.ProductCategoryName, it.ProductBarcode, it.ProductCode,
			it.CourseID, it.CourseName, it.ClientCourseName, it.VoucherSerial,
			it.ServiceRewardID, it.ServiceRewardName, it.ProductRewardID, it.ProductRewardName,
			it.UnitPrice, it.OriginalPrice, it.DiscountType, it.DiscountValue,
			it.ItemOnlineDeposit, it.ItemOnlineDiscount, it.LoyaltyPointsAwarded,
			it.TaxRate, it.TotalAmount, it.TotalAmountPreVouchDisc, it.NetTotalAmount,
			it.GrossTotalAmount, it.NetPrice, it.GrossPrice, it.DiscountAmount,
			it.TaxAmount, it.StaffTips, it.ProductCostPrice, it.ServiceCost, it.ServiceCostType,
			it.GrossTotalWithDiscount, it.GrossTotalWithDiscountMinusTax, it.SimpleDiscountAmount,
			it.MembershipBenefitUsed, it.MembershipDiscountAmount, it.Deal,
			it.SessionNetAmount, it.SessionGrossAmount, it.PhorestTips,

			it.PaymentType, it.PaymentTypeIDs, it.PaymentTypeAmounts,
			it.PaymentTypeCodes, it.PaymentTypeNames, it.PaymentTypeVoucherSerials,
			it.PaymentTypePrepaidTaxAmounts,

			it.OutstandingBalancePMT, it.OpenSale, it.OpenSaleType, it.PurchaseType, it.OnlineBooking,
			it.Void, it.VoidedTransactionID, it.VoidReason,

			it.DepartmentID, it.DepartmentName,

			it.StaffID, it.StaffFirstName, it.StaffLastName,
			it.StaffCategoryID, it.StaffCategoryName, it.IsRequestedStaff,
			it.PrimaryStaffID, it.PreferredStaffID, it.PreferredStaffName,

			it.AppointmentID, it.AppointmentDate, it.AppointmentCreated, it.AppointmentRating,

			it.ClientBirthday, it.ClientGender, it.ClientEmail, it.ClientFirstVisit,

			it.ApptClientID, it.ApptClientFirstName, it.ApptClientLastName,
			it.ApptClientBirthday, it.ApptClientGender, it.ApptClientEmail, it.ApptClientFirstVisit,

			it.InternetCategoryIDs, it.InternetCategoryNames, it.BranchProductID,
			it.FixedDiscountID, it.FixedDiscountName, it.ClientCourseID,
			it.CreatingUser, it.TaxRateName, it.SaleFeeID,

			it.UpdatedAtPhorest, now, now,
		)
		if len(placeholders) >= batchSize {
			if err := flush(); err != nil {
				return err
			}
		}
	}
	return flush()
}
