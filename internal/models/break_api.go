package models

import "time"

type BreakAPI struct {
	ID int64 `gorm:"primaryKey;column:id"`

	BranchID string `gorm:"column:branch_id"`

	BreakID   string    `gorm:"column:break_id"`
	Version   int64     `gorm:"column:version"`
	BreakDate time.Time `gorm:"column:break_date"` // date-only (UTC)

	StartTime string `gorm:"column:start_time"`
	EndTime   string `gorm:"column:end_time"`

	StaffID   string  `gorm:"column:staff_id"`
	RoomID    *string `gorm:"column:room_id"`
	MachineID *string `gorm:"column:machine_id"`
	Label     *string `gorm:"column:label"`
	PaidBreak bool    `gorm:"column:paid_break"`

	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (BreakAPI) TableName() string { return "raw.breaks_api" }
