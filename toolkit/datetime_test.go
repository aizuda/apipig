package toolkit

import (
	"fmt"
	"testing"
)

func TestGetNowLocal(t *testing.T) {
	fmt.Println(GetNowLocal())
	fmt.Println(GetNowUnixMilli())
	now := GetNowLocal()
	fmt.Println(now.Hour())
	year, month, day := now.Date()
	fmt.Println(GetHourlyTimestamps(year, month, day, 0))
	fmt.Println(GetHourlyTimestamps(year, month, day, 3))
	fmt.Println(GetHourlyTimestamps(year, month, day, 10))
	fmt.Println(GetHourlyTimestamps(year, month, day, 23))
}
