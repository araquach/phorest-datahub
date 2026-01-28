package phorest

import (
	"time"
)

func dateOnly(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func endOfMonth(t time.Time) time.Time {
	y, m, _ := t.Date()
	firstNext := time.Date(y, m, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 1, 0)
	return dateOnly(firstNext.AddDate(0, 0, -1))
}

func firstDayOfNextMonth(t time.Time) time.Time {
	y, m, _ := t.Date()
	first := time.Date(y, m, 1, 0, 0, 0, 0, time.UTC)
	return dateOnly(first.AddDate(0, 1, 0))
}

func dayBefore(d time.Time) time.Time {
	return dateOnly(d.AddDate(0, 0, -1))
}
