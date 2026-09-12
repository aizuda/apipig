package api

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

type MapConfig map[string]interface{}

func (c MapConfig) Value() (driver.Value, error) {
	b, err := json.Marshal(c)
	return string(b), err
}

func (c *MapConfig) Scan(src any) error {
	if c == nil {
		return fmt.Errorf("map config: nil destination")
	}
	switch value := src.(type) {
	case nil:
		*c = nil
		return nil
	case []byte:
		if len(value) == 0 {
			*c = nil
			return nil
		}
		return json.Unmarshal(value, c)
	case string:
		if value == "" {
			*c = nil
			return nil
		}
		return json.Unmarshal([]byte(value), c)
	default:
		return fmt.Errorf("map config: unsupported database value %T", src)
	}
}
