package domain

import "time"

type Favorite struct {
	FavID      int64     `db:"fav_id" json:"fav_id"`
	UserID     int64     `db:"user_id" json:"user_id"`
	ShowcaseID int64     `db:"showcase_id" json:"showcase_id"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
}
