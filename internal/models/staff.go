package models

import "time"

type Staff struct {
	ID       uint   `gorm:"primaryKey"`
	StaffID  string `gorm:"column:staff_id;not null;index:ux_staff_branch,unique"`
	BranchID string `gorm:"column:branch_id;not null;index:ux_staff_branch,unique"`

	StaffCategoryID           string     `gorm:"column:staff_category_id"`
	UserID                    string     `gorm:"column:user_id"`
	StaffCategoryName         string     `gorm:"column:staff_category_name"`
	FirstName                 string     `gorm:"column:first_name"`
	LastName                  string     `gorm:"column:last_name"`
	BirthDate                 *time.Time `gorm:"column:birth_date;type:date"`
	StartDate                 *time.Time `gorm:"column:start_date;type:date"`
	SelfEmployed              bool       `gorm:"column:self_employed"`
	Archived                  bool       `gorm:"column:archived"`
	Mobile                    string     `gorm:"column:mobile"`
	Email                     string     `gorm:"column:email"`
	Gender                    string     `gorm:"column:gender"`
	Notes                     string     `gorm:"column:notes"`
	OnlineProfile             string     `gorm:"column:online_profile"`
	HideFromOnlineBookings    bool       `gorm:"column:hide_from_online_bookings"`
	HideFromAppointmentScreen bool       `gorm:"column:hide_from_appointment_screen"`
	ImageURL                  string     `gorm:"column:image_url"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (Staff) TableName() string { return "raw.staff" }
