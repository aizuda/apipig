package api

import (
	"database/sql/driver"
	"encoding/json"
)

type MapConfig map[string]interface{}

func (c MapConfig) Value() (driver.Value, error) {
	b, err := json.Marshal(c)
	return string(b), err
}

func (c *MapConfig) Scan(src any) error {
	return json.Unmarshal(src.([]byte), c)
}
