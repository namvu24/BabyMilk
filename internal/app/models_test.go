package app

import (
	"testing"
)

func TestFeedingInput_Validate_Valid(t *testing.T) {
	input := FeedingInput{
		AmountML:  100,
		StartTime: "2025-01-15T08:00:00Z",
		EndTime:   "2025-01-15T08:15:00Z",
	}
	if err := input.Validate(); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestFeedingInput_Validate_MinAmount(t *testing.T) {
	input := FeedingInput{
		AmountML:  1,
		StartTime: "2025-01-15T08:00:00Z",
		EndTime:   "2025-01-15T08:15:00Z",
	}
	if err := input.Validate(); err != nil {
		t.Errorf("expected amount_ml=1 to be valid, got %v", err)
	}
}

func TestFeedingInput_Validate_ZeroAmount(t *testing.T) {
	input := FeedingInput{
		AmountML:  0,
		StartTime: "2025-01-15T08:00:00Z",
		EndTime:   "2025-01-15T08:15:00Z",
	}
	if err := input.Validate(); err == nil {
		t.Error("expected error for amount_ml=0")
	}
}

func TestFeedingInput_Validate_NegativeAmount(t *testing.T) {
	input := FeedingInput{
		AmountML:  -10,
		StartTime: "2025-01-15T08:00:00Z",
		EndTime:   "2025-01-15T08:15:00Z",
	}
	if err := input.Validate(); err == nil {
		t.Error("expected error for negative amount_ml")
	}
}

func TestFeedingInput_Validate_InvalidStartTime(t *testing.T) {
	input := FeedingInput{
		AmountML:  100,
		StartTime: "not-a-date",
		EndTime:   "2025-01-15T08:15:00Z",
	}
	err := input.Validate()
	if err == nil {
		t.Error("expected error for invalid start_time")
	}
}

func TestFeedingInput_Validate_InvalidEndTime(t *testing.T) {
	input := FeedingInput{
		AmountML:  100,
		StartTime: "2025-01-15T08:00:00Z",
		EndTime:   "not-a-date",
	}
	err := input.Validate()
	if err == nil {
		t.Error("expected error for invalid end_time")
	}
}

func TestFeedingInput_Validate_EndBeforeStart(t *testing.T) {
	input := FeedingInput{
		AmountML:  100,
		StartTime: "2025-01-15T08:15:00Z",
		EndTime:   "2025-01-15T08:00:00Z",
	}
	err := input.Validate()
	if err == nil {
		t.Error("expected error when end_time is before start_time")
	}
}

func TestFeedingInput_Validate_EndEqualsStart(t *testing.T) {
	input := FeedingInput{
		AmountML:  100,
		StartTime: "2025-01-15T08:00:00Z",
		EndTime:   "2025-01-15T08:00:00Z",
	}
	err := input.Validate()
	if err == nil {
		t.Error("expected error when end_time equals start_time")
	}
}
