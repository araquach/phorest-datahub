package models

import "time"

type Review struct {
	ID              uint       `gorm:"primaryKey"`
	ReviewID        string     `gorm:"column:review_id;not null;uniqueIndex"` // Phorest ID
	BranchID        string     `gorm:"column:branch_id;not null;index"`       // for per-branch syncs
	ClientID        string     `gorm:"column:client_id;index"`
	ClientFirstName string     `gorm:"column:client_first_name"`
	ClientLastName  string     `gorm:"column:client_last_name"`
	ReviewDate      *time.Time `gorm:"column:review_date;type:date;index"`
	VisitDate       *time.Time `gorm:"column:visit_date;type:date;index"`
	StaffID         string     `gorm:"column:staff_id;index"`
	StaffFirstName  string     `gorm:"column:staff_first_name"`
	StaffLastName   string     `gorm:"column:staff_last_name"`
	Text            string     `gorm:"column:text"`
	Rating          int        `gorm:"column:rating;index"` // 1..5
	FacebookReview  bool       `gorm:"column:facebook_review"`
	TwitterReview   bool       `gorm:"column:twitter_review"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (Review) TableName() string { return "raw.reviews" }
