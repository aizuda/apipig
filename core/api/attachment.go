package api

import (
	"apipig/toolkit/snowflake"
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

type Attachments []Attachment

func (a Attachments) Value() (driver.Value, error) {
	b, err := json.Marshal(a)
	return string(b), err
}

func (a *Attachments) Scan(src any) error {
	if a == nil {
		return fmt.Errorf("attachments: nil destination")
	}
	switch value := src.(type) {
	case nil:
		*a = nil
		return nil
	case []byte:
		if len(value) == 0 {
			*a = nil
			return nil
		}
		return json.Unmarshal(value, a)
	case string:
		if value == "" {
			*a = nil
			return nil
		}
		return json.Unmarshal([]byte(value), a)
	default:
		return fmt.Errorf("attachments: unsupported database value %T", src)
	}
}

type Attachment struct {
	FileId   snowflake.ID `json:"fileId" swaggertype:"string"` // 文件 ID
	FileName string       `json:"fileName"`                    // 文件名称
}
