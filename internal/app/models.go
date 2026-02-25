package app

import (
	"fmt"
	"time"
)

type Feeding struct {
	ID        int       `json:"id"`
	AmountML  int       `json:"amount_ml"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type FeedingInput struct {
	AmountML  int    `json:"amount_ml"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
}

type DailyTotal struct {
	Date    string `json:"date"`
	TotalML int    `json:"total_ml"`
}

func (f *FeedingInput) Validate() error {
	if f.AmountML <= 0 {
		return fmt.Errorf("amount_ml must be greater than 0")
	}
	start, err := time.Parse(time.RFC3339, f.StartTime)
	if err != nil {
		return fmt.Errorf("invalid start_time format, use RFC3339")
	}
	end, err := time.Parse(time.RFC3339, f.EndTime)
	if err != nil {
		return fmt.Errorf("invalid end_time format, use RFC3339")
	}
	if end.Before(start) {
		return fmt.Errorf("end_time must not be before start_time")
	}
	return nil
}
