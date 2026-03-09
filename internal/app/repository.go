package app

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
	SaveBabyProfile(dob string) (*BabyProfile, error)
	GetDevelopmentCache(weekNumber int) (*DevelopmentContent, error)
	SaveDevelopmentCache(weekNumber int, content string) error
}
