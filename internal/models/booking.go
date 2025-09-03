package models

import "time"

type Booking struct {
	ID        int       `json:"id"`
	EventID   int       `json:"event_id"`
	UserID    string    `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
	Confirmed bool      `json:"confirmed"`
}
