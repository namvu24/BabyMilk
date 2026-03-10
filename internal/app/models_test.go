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
	if err := input.Validate(); err != nil {
		t.Errorf("expected end_time equal to start_time to be valid, got %v", err)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// ── SleepInput validation tests ──
// ═══════════════════════════════════════════════════════════════════════════

func TestSleepInput_Validate_Valid(t *testing.T) {
	input := SleepInput{
		StartTime: "2025-01-15T22:00:00Z",
		EndTime:   "2025-01-16T06:00:00Z",
	}
	if err := input.Validate(); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestSleepInput_Validate_EndEqualsStart(t *testing.T) {
	input := SleepInput{
		StartTime: "2025-01-15T22:00:00Z",
		EndTime:   "2025-01-15T22:00:00Z",
	}
	if err := input.Validate(); err != nil {
		t.Errorf("expected end_time equal to start_time to be valid, got %v", err)
	}
}

func TestSleepInput_Validate_EndBeforeStart(t *testing.T) {
	input := SleepInput{
		StartTime: "2025-01-16T06:00:00Z",
		EndTime:   "2025-01-15T22:00:00Z",
	}
	if err := input.Validate(); err == nil {
		t.Error("expected error when end_time is before start_time")
	}
}

func TestSleepInput_Validate_InvalidStartTime(t *testing.T) {
	input := SleepInput{
		StartTime: "not-a-date",
		EndTime:   "2025-01-16T06:00:00Z",
	}
	if err := input.Validate(); err == nil {
		t.Error("expected error for invalid start_time")
	}
}

func TestSleepInput_Validate_InvalidEndTime(t *testing.T) {
	input := SleepInput{
		StartTime: "2025-01-15T22:00:00Z",
		EndTime:   "not-a-date",
	}
	if err := input.Validate(); err == nil {
		t.Error("expected error for invalid end_time")
	}
}

func TestSleepInput_Validate_EmptyTimes(t *testing.T) {
	input := SleepInput{
		StartTime: "",
		EndTime:   "",
	}
	if err := input.Validate(); err == nil {
		t.Error("expected error for empty times")
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// ── BabyProfileInput validation tests ──
// ═══════════════════════════════════════════════════════════════════════════

func TestBabyProfileInput_Validate_Valid(t *testing.T) {
	input := BabyProfileInput{DateOfBirth: "2025-01-15"}
	if err := input.Validate(); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestBabyProfileInput_Validate_WithGender(t *testing.T) {
	input := BabyProfileInput{DateOfBirth: "2025-01-15", Gender: "male"}
	if err := input.Validate(); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	input.Gender = "female"
	if err := input.Validate(); err != nil {
		t.Errorf("expected no error for female, got %v", err)
	}
}

func TestBabyProfileInput_Validate_InvalidGender(t *testing.T) {
	input := BabyProfileInput{DateOfBirth: "2025-01-15", Gender: "other"}
	if err := input.Validate(); err == nil {
		t.Error("expected error for invalid gender")
	}
}

func TestBabyProfileInput_Validate_MilkType(t *testing.T) {
	for _, mt := range []string{"formula", "breast", "mixed"} {
		input := BabyProfileInput{DateOfBirth: "2025-01-15", MilkType: mt}
		if err := input.Validate(); err != nil {
			t.Errorf("expected no error for milk_type=%s, got %v", mt, err)
		}
	}
}

func TestBabyProfileInput_Validate_InvalidMilkType(t *testing.T) {
	input := BabyProfileInput{DateOfBirth: "2025-01-15", MilkType: "cow"}
	if err := input.Validate(); err == nil {
		t.Error("expected error for invalid milk_type")
	}
}

func TestBabyProfileInput_Validate_MissingDOB(t *testing.T) {
	input := BabyProfileInput{}
	if err := input.Validate(); err == nil {
		t.Error("expected error for missing date_of_birth")
	}
}

func TestBabyProfileInput_Validate_InvalidDOBFormat(t *testing.T) {
	input := BabyProfileInput{DateOfBirth: "15-01-2025"}
	if err := input.Validate(); err == nil {
		t.Error("expected error for invalid date format")
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// ── GrowthMeasurementInput validation tests ──
// ═══════════════════════════════════════════════════════════════════════════

func TestGrowthInput_Validate_Valid(t *testing.T) {
	input := GrowthMeasurementInput{
		Date:     "2025-06-15",
		WeightKg: 5.5,
		LengthCm: 55.0,
	}
	if err := input.Validate(); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestGrowthInput_Validate_WithHead(t *testing.T) {
	head := 37.5
	input := GrowthMeasurementInput{
		Date:                "2025-06-15",
		WeightKg:            5.5,
		LengthCm:            55.0,
		HeadCircumferenceCm: &head,
	}
	if err := input.Validate(); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestGrowthInput_Validate_MissingDate(t *testing.T) {
	input := GrowthMeasurementInput{WeightKg: 5.5, LengthCm: 55.0}
	if err := input.Validate(); err == nil {
		t.Error("expected error for missing date")
	}
}

func TestGrowthInput_Validate_InvalidDateFormat(t *testing.T) {
	input := GrowthMeasurementInput{Date: "15-06-2025", WeightKg: 5.5, LengthCm: 55.0}
	if err := input.Validate(); err == nil {
		t.Error("expected error for invalid date format")
	}
}

func TestGrowthInput_Validate_WeightTooLow(t *testing.T) {
	input := GrowthMeasurementInput{Date: "2025-06-15", WeightKg: 0.1, LengthCm: 55.0}
	if err := input.Validate(); err == nil {
		t.Error("expected error for weight too low")
	}
}

func TestGrowthInput_Validate_WeightTooHigh(t *testing.T) {
	input := GrowthMeasurementInput{Date: "2025-06-15", WeightKg: 31, LengthCm: 55.0}
	if err := input.Validate(); err == nil {
		t.Error("expected error for weight too high")
	}
}

func TestGrowthInput_Validate_LengthTooLow(t *testing.T) {
	input := GrowthMeasurementInput{Date: "2025-06-15", WeightKg: 5.5, LengthCm: 20}
	if err := input.Validate(); err == nil {
		t.Error("expected error for length too low")
	}
}

func TestGrowthInput_Validate_LengthTooHigh(t *testing.T) {
	input := GrowthMeasurementInput{Date: "2025-06-15", WeightKg: 5.5, LengthCm: 140}
	if err := input.Validate(); err == nil {
		t.Error("expected error for length too high")
	}
}

func TestGrowthInput_Validate_HeadTooLow(t *testing.T) {
	head := 15.0
	input := GrowthMeasurementInput{Date: "2025-06-15", WeightKg: 5.5, LengthCm: 55.0, HeadCircumferenceCm: &head}
	if err := input.Validate(); err == nil {
		t.Error("expected error for head circumference too low")
	}
}

func TestGrowthInput_Validate_HeadTooHigh(t *testing.T) {
	head := 65.0
	input := GrowthMeasurementInput{Date: "2025-06-15", WeightKg: 5.5, LengthCm: 55.0, HeadCircumferenceCm: &head}
	if err := input.Validate(); err == nil {
		t.Error("expected error for head circumference too high")
	}
}

func TestGrowthInput_Validate_BoundaryWeight(t *testing.T) {
	// At boundary: 0.5 and 30 should be valid
	input := GrowthMeasurementInput{Date: "2025-06-15", WeightKg: 0.5, LengthCm: 50}
	if err := input.Validate(); err != nil {
		t.Errorf("expected 0.5 kg to be valid, got %v", err)
	}
	input.WeightKg = 30
	if err := input.Validate(); err != nil {
		t.Errorf("expected 30 kg to be valid, got %v", err)
	}
}
