package models

import "time"

type Reservation struct {
	ID        int       `json:"id"`
	BookID    int       `json:"book_id"`
	UserEmail string    `json:"user_email"`
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
	Status    string    `json:"status"` // "active", "completed", "cancelled"
}
