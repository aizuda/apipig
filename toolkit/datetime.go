package toolkit

import (
	"fmt"
	"strconv"
	"time"
)

const DateLayout1 = "2006-01-02 15:04:05.000"

func GetTimeNow() string {
	return GetTimeNowFormat(DateLayout1)
}
func GetSqlTimeNow() string {
	return fmt.Sprintf("'%s'", GetTimeNowFormat(DateLayout1))
}

func GetTimeNowFormat(layout string) string {
	return GetNowLocal().Format(layout)
}

func GetNowUnix() int64 {
	return GetNowLocal().Unix()
}

func GetNowUnixMilli() int64 {
	return GetNowLocal().UnixMilli()
}

func GetNowLocal() time.Time {
	return time.Now().Local()
}

func Get3MinutesLaterUnix() int64 {
	return GetNowLocal().Add(3 * time.Minute).Unix()
}

func GetMinute(duration int64) time.Duration {
	return time.Duration(duration) * time.Minute
}

func GetStrTimestampFormat(timestamp string) string {
	millis, err := strconv.ParseInt(timestamp, 10, 64)
	if err == nil {
		return GetTimestampFormat(millis)
	}
	return ""
}

func GetTimestampFormat(timestamp int64) string {
	if timestamp == 0 {
		return ""
	}
	var sec, nano int64
	if timestamp > 1e9 {
		for timestamp < 1e18 {
			timestamp *= 10
		}
		sec = timestamp / 1e9
		nano = timestamp % 1e9
	} else {
		sec = timestamp
	}
	return time.Unix(sec, nano).Format(DateLayout1)
}

func ParseInt64(str string) (int64, error) {
	return strconv.ParseInt(str, 10, 64)
}

func GetHourlyTimestamps(year int, month time.Month, day, hour int) int64 {
	baseTime := time.Date(year, month, day, hour, 0, 0, 0, time.Local)
	return baseTime.UnixMilli()
}
