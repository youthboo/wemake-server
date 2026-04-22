package domain

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// JSONInt64Array is a PostgreSQL JSONB array of integer IDs.
type JSONInt64Array []int64

func (a *JSONInt64Array) Scan(src interface{}) error {
	if src == nil {
		*a = JSONInt64Array{}
		return nil
	}
	var data []byte
	switch v := src.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return fmt.Errorf("JSONInt64Array: unsupported type %T", src)
	}
	if len(data) == 0 {
		*a = JSONInt64Array{}
		return nil
	}
	var slice []int64
	if err := json.Unmarshal(data, &slice); err != nil {
		return err
	}
	*a = slice
	return nil
}

func (a JSONInt64Array) Value() (driver.Value, error) {
	if len(a) == 0 {
		return []byte("[]"), nil
	}
	return json.Marshal([]int64(a))
}
