package utils

import (
	"fmt"
	"time"
)

func ConvertUnixTimeToTime(timeUnix float64) time.Time {
	unixTimeInt := int64(timeUnix)

	fraction := timeUnix - float64(unixTimeInt)
	nanoSeconds := int64(fraction * float64(time.Second))

	return time.Unix(unixTimeInt, nanoSeconds)
}

func IsDateAfterDaysOffset(date time.Time, daysOffset int64) bool {
	return date.After(time.Now().AddDate(0, 0, -int(daysOffset)))
}

func DisplayTime(time time.Time) string {
	return time.Format("2006-01-02 15:04:05")
}

func FormatTimeAgo(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		mins := int(diff.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", mins)
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	case diff < 7*24*time.Hour:
		days := int(diff.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	default:
		weeks := int(diff.Hours() / 24 / 7)
		if weeks == 1 {
			return "1 week ago"
		}
		return fmt.Sprintf("%d weeks ago", weeks)
	}
}
