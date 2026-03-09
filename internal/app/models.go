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

// ── Sleep models ──

type Sleep struct {
	ID        int       `json:"id"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type SleepInput struct {
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
}

type DailySleepTotal struct {
	Date         string `json:"date"`
	TotalMinutes int    `json:"total_minutes"`
}

func (s *SleepInput) Validate() error {
	start, err := time.Parse(time.RFC3339, s.StartTime)
	if err != nil {
		return fmt.Errorf("invalid start_time format, use RFC3339")
	}
	end, err := time.Parse(time.RFC3339, s.EndTime)
	if err != nil {
		return fmt.Errorf("invalid end_time format, use RFC3339")
	}
	if end.Before(start) {
		return fmt.Errorf("end_time must not be before start_time")
	}
	return nil
}

// ── Baby profile & development models ──

type BabyProfile struct {
	ID          int       `json:"id"`
	DateOfBirth time.Time `json:"date_of_birth"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type BabyProfileInput struct {
	DateOfBirth string `json:"date_of_birth"`
}

func (b *BabyProfileInput) Validate() error {
	if b.DateOfBirth == "" {
		return fmt.Errorf("date_of_birth is required")
	}
	_, err := time.Parse("2006-01-02", b.DateOfBirth)
	if err != nil {
		return fmt.Errorf("invalid date_of_birth format, use YYYY-MM-DD")
	}
	return nil
}

type DevelopmentContent struct {
	ID         int       `json:"id"`
	WeekNumber int       `json:"week_number"`
	Content    string    `json:"content"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
