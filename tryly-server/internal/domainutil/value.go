package domainutil

import (
	"math"
	"reflect"
	"strings"
)

func RoundMoney(v float64) float64 {
	return math.Round(v*100) / 100
}

func IntValue(v *int) int {
	if v == nil {
		return 0
	}
	return *v
}

func Int64Value(v *int64) int64 {
	if v == nil {
		return 0
	}
	return *v
}

func BoolValue(v *bool) bool {
	return v != nil && *v
}

func Nullable(v interface{}) interface{} {
	if v == nil {
		return nil
	}
	rv := reflect.ValueOf(v)
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return nil
		}
		rv = rv.Elem()
	}
	return rv.Interface()
}

func NullablePositiveInt64(v int64) interface{} {
	if v <= 0 {
		return nil
	}
	return v
}

// NullableString returns nil for empty/whitespace strings so they persist as SQL NULL.
func NullableString(v string) interface{} {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	return v
}

func NormalizeStatus(v string) string {
	return strings.ToUpper(strings.TrimSpace(v))
}

func NormalizeLower(v string) string {
	return strings.ToLower(strings.TrimSpace(v))
}

func StatusIn(value string, allowed ...string) bool {
	normalized := NormalizeStatus(value)
	for _, item := range allowed {
		if normalized == NormalizeStatus(item) {
			return true
		}
	}
	return false
}

func NormalizeUpperOrDefault(v string, fallback string) string {
	normalized := NormalizeStatus(v)
	if normalized == "" {
		return fallback
	}
	return normalized
}
