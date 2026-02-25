package app

// Repository defines the data access interface for feedings.
type Repository interface {
	GetFeedings(date string) ([]Feeding, error)
	CreateFeeding(input FeedingInput) (Feeding, error)
	UpdateFeeding(id int, input FeedingInput) (Feeding, error)
	DeleteFeeding(id int) error
	GetDailyTotals(days int) ([]DailyTotal, error)
}
