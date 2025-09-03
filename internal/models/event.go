package models

import "time"

type Event struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	Date        time.Time `json:"date"`
	TotalSeats  int       `json:"total_seats"`
	BookedSeats int       `json:"booked_seats"`
	Deadline    int       `json:"deadline_minutes"`
}
