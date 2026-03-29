package domain

import "time"

type FactoryReview struct {
	ReviewID  int64     `db:"review_id" json:"review_id"`
	FactoryID int64     `db:"factory_id" json:"factory_id"`
	UserID    int64     `db:"user_id" json:"user_id"`
	Rating    int       `db:"rating" json:"rating"`
	Comment   string    `db:"comment" json:"comment"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}
