package models

import (
	"fmt"
	"time"
)

// PeriodCutoff returns the cutoff time for the given period relative to now.
// Valid periods: "day" (24h), "week" (7 days), "month" (30 days).
func PeriodCutoff(period string, now time.Time) (time.Time, error) {
	switch period {
	case "day":
		return now.AddDate(0, 0, -1), nil
	case "week":
		return now.AddDate(0, 0, -7), nil
	case "month":
		return now.AddDate(0, 0, -30), nil
	default:
		return time.Time{}, fmt.Errorf("invalid period %q: must be day, week, or month", period)
	}
}
