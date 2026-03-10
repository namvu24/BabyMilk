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
	Name        string    `json:"name,omitempty"`
	Gender      string    `json:"gender,omitempty"`
	MilkType    string    `json:"milk_type,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type BabyProfileInput struct {
	DateOfBirth string `json:"date_of_birth"`
	Name        string `json:"name,omitempty"`
	Gender      string `json:"gender,omitempty"`
	MilkType    string `json:"milk_type,omitempty"`
}

func (b *BabyProfileInput) Validate() error {
	if b.DateOfBirth == "" {
		return fmt.Errorf("date_of_birth is required")
	}
	_, err := time.Parse("2006-01-02", b.DateOfBirth)
	if err != nil {
		return fmt.Errorf("invalid date_of_birth format, use YYYY-MM-DD")
	}
	if b.Gender != "" && b.Gender != "male" && b.Gender != "female" {
		return fmt.Errorf("gender must be 'male' or 'female'")
	}
	if b.MilkType != "" && b.MilkType != "formula" && b.MilkType != "breast" && b.MilkType != "mixed" {
		return fmt.Errorf("milk_type must be 'formula', 'breast', or 'mixed'")
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

// ── Growth measurement models ──

type GrowthMeasurement struct {
	ID                  int       `json:"id"`
	Date                time.Time `json:"date"`
	WeightKg            float64   `json:"weight_kg"`
	LengthCm            float64   `json:"length_cm"`
	HeadCircumferenceCm *float64  `json:"head_circumference_cm,omitempty"`
	Notes               string    `json:"notes,omitempty"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type GrowthMeasurementInput struct {
	Date                string   `json:"date"`
	WeightKg            float64  `json:"weight_kg"`
	LengthCm            float64  `json:"length_cm"`
	HeadCircumferenceCm *float64 `json:"head_circumference_cm,omitempty"`
	Notes               string   `json:"notes,omitempty"`
}

func (g *GrowthMeasurementInput) Validate() error {
	if g.Date == "" {
		return fmt.Errorf("date is required")
	}
	_, err := time.Parse("2006-01-02", g.Date)
	if err != nil {
		return fmt.Errorf("invalid date format, use YYYY-MM-DD")
	}
	if g.WeightKg < 0.5 || g.WeightKg > 30 {
		return fmt.Errorf("weight_kg must be between 0.5 and 30")
	}
	if g.LengthCm < 30 || g.LengthCm > 130 {
		return fmt.Errorf("length_cm must be between 30 and 130")
	}
	if g.HeadCircumferenceCm != nil && (*g.HeadCircumferenceCm < 20 || *g.HeadCircumferenceCm > 60) {
		return fmt.Errorf("head_circumference_cm must be between 20 and 60")
	}
	return nil
}

// ── Personalized insights models ──

type PersonalizedInsight struct {
	GrowthAssessment GrowthAssessment `json:"growth_assessment"`
	FeedingAnalysis  FeedingAnalysis  `json:"feeding_analysis"`
	SleepAnalysis    SleepAnalysis    `json:"sleep_analysis"`
	Activities       []Activity       `json:"activities"`
	Alerts           []Alert          `json:"alerts"`
	Summary          string           `json:"summary"`
}

type GrowthAssessment struct {
	Percentile int    `json:"percentile"`
	Status     string `json:"status"`
	Reasoning  string `json:"reasoning"`
}

type FeedingAnalysis struct {
	DailyAvgML      int    `json:"daily_avg_ml"`
	RecommendedML   int    `json:"recommended_ml"`
	MilkTypeGuidance string `json:"milk_type_guidance"`
	Recommendations []string `json:"recommendations"`
}

type SleepAnalysis struct {
	DailyAvgMinutes    int    `json:"daily_avg_minutes"`
	RecommendedMinutes int    `json:"recommended_minutes"`
	PatternObservations string `json:"pattern_observations"`
}

type Activity struct {
	Name         string `json:"name"`
	Icon         string `json:"icon"`
	Instructions string `json:"instructions"`
	Benefits     string `json:"benefits"`
	Duration     string `json:"duration"`
	Difficulty   string `json:"difficulty"`
}

type Alert struct {
	Severity string `json:"severity"`
	Message  string `json:"message"`
	Action   string `json:"action"`
}

// ── Insights cache model ──

type InsightCache struct {
	ID        int       `json:"id"`
	CacheKey  string    `json:"cache_key"`
	Content   string    `json:"content"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}
