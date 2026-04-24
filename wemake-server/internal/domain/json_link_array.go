package domain

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strconv"
)

// JSONLinkArray accepts a JSON array of strings or numbers and stores them as strings.
type JSONLinkArray []string

func (a *JSONLinkArray) Scan(src interface{}) error {
	if src == nil {
		*a = JSONLinkArray{}
		return nil
	}
	var data []byte
	switch v := src.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return fmt.Errorf("JSONLinkArray: unsupported type %T", src)
	}
	if len(data) == 0 {
		*a = JSONLinkArray{}
		return nil
	}
	var raw []interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		switch v := item.(type) {
		case string:
			out = append(out, v)
		case float64:
			out = append(out, strconv.FormatInt(int64(v), 10))
		default:
			return fmt.Errorf("JSONLinkArray: unsupported element type %T", item)
		}
	}
	*a = JSONLinkArray(out)
	return nil
}

func (a JSONLinkArray) Value() (driver.Value, error) {
	if len(a) == 0 {
		return []byte("[]"), nil
	}
	return json.Marshal([]string(a))
}
