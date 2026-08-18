package shared

import (
	"fmt"
	"time"
)

const DefaultTimezone = "Asia/Kolkata"

func GetLocation() *time.Location {
	loc, err := time.LoadLocation(DefaultTimezone)
	if err != nil {
		return time.FixedZone("IST", 5*3600+30*60)
	}
	return loc
}

func GetCurrentBusinessDate() time.Time {
	now := time.Now().In(GetLocation())
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, GetLocation())
}

func FormatBusinessDate(t time.Time) string {
	return t.Format("2006-01-02")
}

func ParseBusinessDate(dateStr string) (time.Time, error) {
	if dateStr == "" {
		return GetCurrentBusinessDate(), nil
	}
	t, err := time.ParseInLocation("2006-01-02", dateStr, GetLocation())
	if err != nil {
		return time.Time{}, err
	}
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, GetLocation()), nil
}

// FinancialYearLabel returns the Indian financial year label for a date
// (April 1 - March 31), e.g. "26-27" for any date from 2026-04-01 through
// 2027-03-31.
func FinancialYearLabel(t time.Time) string {
	startYear := t.Year()
	if t.Month() < time.April {
		startYear--
	}
	return fmt.Sprintf("%02d-%02d", startYear%100, (startYear+1)%100)
}
