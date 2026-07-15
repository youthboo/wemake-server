package helper

import (
	"fmt"
	"strings"
	"time"

	"github.com/yourusername/wemake/internal/domainutil"
)

var ThailandLocation = time.FixedZone("Asia/Bangkok", 7*60*60)

// AsThailandWallClock re-anchors a time.Time scanned from a Postgres
// "timestamp without time zone" column to the Asia/Bangkok location.
//
// Why this exists: the app's DB DSN pins the Postgres session TimeZone to
// Asia/Bangkok (see config.Config.GetDSN) specifically so NOW()/CURRENT_TIMESTAMP
// and naive "timestamp" columns hold Bangkok wall-clock values everywhere,
// regardless of which machine (dev laptop, container) runs the app. But the
// lib/pq driver always labels naive "timestamp" values it scans as UTC — the
// wall-clock digits are correct (e.g. "19:00:00"), but the Location is wrong
// (UTC instead of Asia/Bangkok). Comparing such a mislabeled time.Time against
// a correctly-zoned one (e.g. a timestamptz value, or a parsed external
// timestamp like a bank slip's transfer time) silently shifts the naive value
// by the Bangkok UTC offset (+7h), which can flip Before()/After() results
// for any two events less than 7 hours apart.
//
// Call this on any time.Time scanned from a naive "timestamp" column before
// comparing it (.Before/.After/.Sub) against a properly-zoned time.Time.
func AsThailandWallClock(t time.Time) time.Time {
	return time.Date(
		t.Year(), t.Month(), t.Day(),
		t.Hour(), t.Minute(), t.Second(), t.Nanosecond(),
		ThailandLocation,
	)
}

func RoundCurrency(v float64) float64 {
	return domainutil.RoundMoney(v)
}

func RoundMoney(v float64) float64 {
	return domainutil.RoundMoney(v)
}

func PercentOf(amount, total float64) float64 {
	if total <= 0 {
		return 0
	}
	return RoundCurrency((amount / total) * 100)
}

func DerefString(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func MaxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func FormatThaiShortDate(t time.Time) string {
	months := []string{"ม.ค.", "ก.พ.", "มี.ค.", "เม.ย.", "พ.ค.", "มิ.ย.", "ก.ค.", "ส.ค.", "ก.ย.", "ต.ค.", "พ.ย.", "ธ.ค."}
	return fmt.Sprintf("%d %s %02d", t.Day(), months[int(t.Month())-1], (t.Year()+543)%100)
}

func DereferenceString(ptr *string, defaultVal string) string {
	if ptr == nil {
		return defaultVal
	}
	return strings.TrimSpace(*ptr)
}

func DereferenceInt(ptr *int, defaultVal int) int {
	if ptr == nil {
		return defaultVal
	}
	return *ptr
}

func AssignIfNotNil[T any](target *T, ptr *T) {
	if ptr != nil {
		*target = *ptr
	}
}
