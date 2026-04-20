package domain

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

type StringArray []string

func (a *StringArray) Scan(src interface{}) error {
	switch v := src.(type) {
	case nil:
		*a = StringArray{}
		return nil
	case []byte:
		if len(v) == 0 {
			*a = StringArray{}
			return nil
		}
		return json.Unmarshal(v, a)
	case string:
		if v == "" {
			*a = StringArray{}
			return nil
		}
		return json.Unmarshal([]byte(v), a)
	default:
		return fmt.Errorf("unsupported StringArray scan type %T", src)
	}
}

func (a StringArray) Value() (driver.Value, error) {
	if len(a) == 0 {
		return []byte("[]"), nil
	}
	b, err := json.Marshal(a)
	if err != nil {
		return nil, err
	}
	return b, nil
}

type ProductionStepTemplate struct {
	StepID           int64  `db:"step_id" json:"step_id"`
	StepCode         string `db:"step_code" json:"step_code"`
	StepNameTH       string `db:"step_name_th" json:"step_name_th"`
	StepNameEN       string `db:"step_name_en" json:"step_name_en"`
	SortOrder        int64  `db:"sort_order" json:"sort_order"`
	RequiresEvidence bool   `db:"requires_evidence" json:"requires_evidence"`
	MinPhotos        int64  `db:"min_photos" json:"min_photos"`
	IsPaymentTrigger bool   `db:"is_payment_trigger" json:"is_payment_trigger"`
	IconName         string `db:"icon_name" json:"icon_name"`
	Description      string `db:"description" json:"description"`
	IsActive         bool   `db:"is_active" json:"is_active"`
}

type ProductionUpdateAutoProgressed struct {
	StepID int64  `json:"step_id"`
	Status string `json:"status"`
}

type ProductionUpdateResult struct {
	Update         ProductionUpdate                `json:"update"`
	OrderStatus    string                          `json:"order_status"`
	AutoProgressed *ProductionUpdateAutoProgressed `json:"auto_progressed,omitempty"`
}

type ProductionUpdatesList struct {
	OrderID     int64              `json:"order_id"`
	Updates     []ProductionUpdate `json:"updates"`
	OrderStatus string             `json:"order_status"`
}

type DomainEvent struct {
	EventID     int64           `db:"event_id" json:"event_id"`
	EventType   string          `db:"event_type" json:"event_type"`
	Payload     json.RawMessage `db:"payload" json:"payload"`
	CreatedAt   time.Time       `db:"created_at" json:"created_at"`
	ProcessedAt *time.Time      `db:"processed_at" json:"processed_at,omitempty"`
}
