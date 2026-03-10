package app

import "time"

// Repository defines the data access interface for feedings and sleeps.
type Repository interface {
	// Feeding methods
	GetFeedings(date string, tz string) ([]Feeding, error)
	GetLastFeeding() (*Feeding, error)
	CreateFeeding(input FeedingInput) (Feeding, error)
	UpdateFeeding(id int, input FeedingInput) (Feeding, error)
	DeleteFeeding(id int) error
	GetDailyTotals(days int, tz string) ([]DailyTotal, error)
	GetDailyTotalsByMonth(month string, tz string) ([]DailyTotal, error)

	// Sleep methods
	GetSleeps(date string, tz string) ([]Sleep, error)
	GetLastSleep() (*Sleep, error)
	CreateSleep(input SleepInput) (Sleep, error)
	UpdateSleep(id int, input SleepInput) (Sleep, error)
	DeleteSleep(id int) error
	GetSleepDailyTotals(days int, tz string) ([]DailySleepTotal, error)
	GetSleepDailyTotalsByMonth(month string, tz string) ([]DailySleepTotal, error)

	// Baby profile & development methods
	GetBabyProfile() (*BabyProfile, error)
	SaveBabyProfile(input BabyProfileInput) (*BabyProfile, error)
	GetDevelopmentCache(weekNumber int) (*DevelopmentContent, error)
	SaveDevelopmentCache(weekNumber int, content string) error

	// Growth measurement methods
	GetGrowthMeasurements(limit int) ([]GrowthMeasurement, error)
	GetGrowthMeasurementsByRange(from, to time.Time) ([]GrowthMeasurement, error)
	GetLatestGrowthMeasurement() (*GrowthMeasurement, error)
	CreateGrowthMeasurement(input GrowthMeasurementInput) (GrowthMeasurement, error)
	UpdateGrowthMeasurement(id int, input GrowthMeasurementInput) (GrowthMeasurement, error)
	DeleteGrowthMeasurement(id int) error

	// Insight cache methods
	GetInsightCache(key string) (*InsightCache, error)
	SaveInsightCache(key, content string, expiresAt time.Time) error
	InvalidateInsightCache() error

	// Aggregation methods for insights
	GetFeedingDailyAvg(days int) (int, error)
	GetSleepDailyAvg(days int) (int, error)
}
