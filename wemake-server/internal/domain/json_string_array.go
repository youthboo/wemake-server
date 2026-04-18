package domain

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// JSONStringArray is a PostgreSQL JSONB array of strings for database/sql and JSON APIs.
type JSONStringArray []string

func (a *JSONStringArray) Scan(src interface{}) error {
	if src == nil {
		*a = JSONStringArray{}
		return nil
	}
	var data []byte
	switch v := src.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return fmt.Errorf("JSONStringArray: unsupported type %T", src)
	}
	if len(data) == 0 {
		*a = JSONStringArray{}
		return nil
	}
	var slice []string
	if err := json.Unmarshal(data, &slice); err != nil {
		return err
	}
	*a = slice
	return nil
}

func (a JSONStringArray) Value() (driver.Value, error) {
	if len(a) == 0 {
		return []byte("[]"), nil
	}
	return json.Marshal([]string(a))
}
