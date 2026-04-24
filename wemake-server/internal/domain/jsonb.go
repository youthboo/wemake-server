package domain

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

type JSONB map[string]interface{}

func (j *JSONB) Scan(src interface{}) error {
	if src == nil {
		*j = JSONB{}
		return nil
	}
	var data []byte
	switch v := src.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return fmt.Errorf("JSONB: unsupported type %T", src)
	}
	if len(data) == 0 {
		*j = JSONB{}
		return nil
	}
	return json.Unmarshal(data, j)
}

func (j JSONB) Value() (driver.Value, error) {
	if j == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(j)
}
