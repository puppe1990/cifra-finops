package scheduler

import "time"

// NextRun returns the next UTC instant the monthly job should fire: the last
// day of the month at hour. When now is already past that instant, it returns
// the last day of the following month.
func NextRun(now time.Time, hour int) time.Time {
	last := lastDayOfMonth(now.Year(), now.Month(), hour)
	if !now.Before(last) {
		last = lastDayOfMonth(now.Year(), now.Month()+1, hour)
	}
	return last
}

// lastDayOfMonth uses day 0 of the following month so month overflow and leap
// years fall out of time.Date normalization.
func lastDayOfMonth(year int, month time.Month, hour int) time.Time {
	return time.Date(year, month+1, 0, hour, 0, 0, 0, time.UTC)
}
