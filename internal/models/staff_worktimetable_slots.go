// internal/models/staff_worktimetable_slot.go
package models

import "time"

type StaffWorkTimetableSlot struct {
	ID int64 `gorm:"primaryKey;column:id"`

	BranchID string `gorm:"column:branch_id;index"`
	StaffID  string `gorm:"column:staff_id;index"`

	SlotDate  time.Time `gorm:"column:slot_date;index"`
	StartTime string    `gorm:"column:start_time;type:time"`
	EndTime   string    `gorm:"column:end_time;type:time"`

	TimeOffStartTime *string `gorm:"column:time_off_start_time;type:time"`
	TimeOffEndTime   *string `gorm:"column:time_off_end_time;type:time"`

	Type           string  `gorm:"column:type"`
	Custom         *string `gorm:"column:custom"`
	SlotBranchID   *string `gorm:"column:slot_branch_id"`
	WorkActivityID *string `gorm:"column:work_activity_id"`

	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (StaffWorkTimetableSlot) TableName() string { return "raw.staff_worktimetable_slots" }
