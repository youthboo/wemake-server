package domain

import "time"

type FactoryReview struct {
	ReviewID       int64      `db:"review_id" json:"review_id"`
	FactoryID      int64      `db:"factory_id" json:"factory_id"`
	UserID         int64      `db:"user_id" json:"user_id"`
	OrderID        *int64     `db:"order_id" json:"order_id,omitempty"`
	Rating         int        `db:"rating" json:"rating"`
	Comment        string     `db:"comment" json:"comment"`
	CreatedAt      time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt      *time.Time `db:"updated_at" json:"updated_at,omitempty"`
	FactoryReply   *string    `db:"factory_reply" json:"factory_reply,omitempty"`
	FactoryReplyAt *time.Time `db:"factory_reply_at" json:"factory_reply_at,omitempty"`
	FactoryReplyBy *int64     `db:"factory_reply_by" json:"factory_reply_by,omitempty"`
	ReviewerName   *string    `db:"reviewer_name" json:"reviewer_name,omitempty"`
}

type FactoryReviewSummary struct {
	FactoryID       int64            `json:"factory_id"`
	AverageRating   float64          `json:"average_rating"`
	ReviewCount     int64            `json:"review_count"`
	RatingBreakdown map[string]int64 `json:"rating_breakdown"`
}

type OrderReviewState struct {
	OrderID         int64          `json:"order_id"`
	FactoryID       int64          `json:"factory_id"`
	FactoryName     string         `json:"factory_name"`
	Eligible        bool           `json:"eligible"`
	Reason          *string        `json:"reason,omitempty"`
	AlreadyReviewed bool           `json:"already_reviewed"`
	Review          *FactoryReview `json:"review,omitempty"`
}
