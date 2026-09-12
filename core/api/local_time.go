package api

import (
	"bytes"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

const TimeFormat = "2006-01-02 15:04:05"

type LocalTime time.Time

func (t *LocalTime) UnmarshalJSON(data []byte) (err error) {
	if t == nil {
		return fmt.Errorf("local time: nil destination")
	}
	data = bytes.TrimSpace(data)
	if bytes.Equal(data, []byte("null")) || bytes.Equal(data, []byte(`""`)) {
		*t = LocalTime(time.Time{})
		return
	}

	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf("local time: %w", err)
	}
	now, err := time.Parse(TimeFormat, value)
	if err != nil {
		return fmt.Errorf("local time: %w", err)
	}
	*t = LocalTime(now)
	return nil
}

func (t LocalTime) MarshalJSON() ([]byte, error) {
	b := make([]byte, 0, len(TimeFormat)+2)
	b = append(b, '"')
	b = time.Time(t).AppendFormat(b, TimeFormat)
	b = append(b, '"')
	return b, nil
}

func (t LocalTime) Value() (driver.Value, error) {
	if t.String() == "0001-01-01 00:00:00" {
		return nil, nil
	}
	return []byte(time.Time(t).Format(TimeFormat)), nil
}

func (t *LocalTime) Scan(v interface{}) error {
	if t == nil {
		return fmt.Errorf("local time: nil destination")
	}
	switch value := v.(type) {
	case nil:
		*t = LocalTime(time.Time{})
		return nil
	case time.Time:
		*t = LocalTime(value)
		return nil
	case []byte:
		return t.parseDatabaseString(string(value))
	case string:
		return t.parseDatabaseString(value)
	default:
		return fmt.Errorf("local time: unsupported database value %T", v)
	}
}

func (t *LocalTime) parseDatabaseString(value string) error {
	if value == "" {
		*t = LocalTime(time.Time{})
		return nil
	}
	parsed, err := time.ParseInLocation(TimeFormat, value, time.Local)
	if err != nil {
		return fmt.Errorf("local time: %w", err)
	}
	*t = LocalTime(parsed)
	return nil
}

func (t LocalTime) String() string {
	return time.Time(t).Format(TimeFormat)
}
