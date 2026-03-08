package app

// Repository defines the data access interface for feedings.
type Repository interface {
	GetFeedings(date string, tz string) ([]Feeding, error)
	GetLastFeeding() (*Feeding, error)
	CreateFeeding(input FeedingInput) (Feeding, error)
	UpdateFeeding(id int, input FeedingInput) (Feeding, error)
	DeleteFeeding(id int) error
	GetDailyTotals(days int, tz string) ([]DailyTotal, error)
	GetDailyTotalsByMonth(month string, tz string) ([]DailyTotal, error)
}
