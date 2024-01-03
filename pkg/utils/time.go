package utils

import "time"

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
