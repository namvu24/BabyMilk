package app

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/lib/pq"
)

// InitDB opens a PostgreSQL connection and verifies connectivity.
func InitDB() (*sql.DB, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/babymilk?sslmode=disable"
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}

// PostgresRepository implements Repository using PostgreSQL.
type PostgresRepository struct {
	DB *sql.DB
}

// NewPostgresRepository creates a new PostgresRepository.
func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{DB: db}
}

func (r *PostgresRepository) GetFeedings(dateFilter string, tz string) ([]Feeding, error) {
	if tz == "" {
		tz = "UTC"
	}
	query := `SELECT id, amount_ml, start_time, end_time, created_at, updated_at FROM feedings`
	var args []interface{}
	if len(dateFilter) == 7 {
		// YYYY-MM month filter — convert to client timezone before comparing
		query += ` WHERE to_char(start_time AT TIME ZONE $1, 'YYYY-MM') = $2`
		args = append(args, tz, dateFilter)
	} else if len(dateFilter) == 10 {
		// YYYY-MM-DD date filter — convert to client timezone before comparing
		query += ` WHERE (start_time AT TIME ZONE $1)::date = $2`
		args = append(args, tz, dateFilter)
	}
	query += ` ORDER BY start_time DESC`

	rows, err := r.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var feedings []Feeding
	for rows.Next() {
		var f Feeding
		if err := rows.Scan(&f.ID, &f.AmountML, &f.StartTime, &f.EndTime, &f.CreatedAt, &f.UpdatedAt); err != nil {
			return nil, err
		}
		feedings = append(feedings, f)
	}
	return feedings, rows.Err()
}

func (r *PostgresRepository) GetLastFeeding() (*Feeding, error) {
	var f Feeding
	err := r.DB.QueryRow(
		`SELECT id, amount_ml, start_time, end_time, created_at, updated_at
		 FROM feedings ORDER BY created_at DESC LIMIT 1`,
	).Scan(&f.ID, &f.AmountML, &f.StartTime, &f.EndTime, &f.CreatedAt, &f.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &f, nil
}

func (r *PostgresRepository) CreateFeeding(input FeedingInput) (Feeding, error) {
	start, _ := time.Parse(time.RFC3339, input.StartTime)
	end, _ := time.Parse(time.RFC3339, input.EndTime)
	var f Feeding
	err := r.DB.QueryRow(
		`INSERT INTO feedings (amount_ml, start_time, end_time) VALUES ($1, $2, $3)
		 RETURNING id, amount_ml, start_time, end_time, created_at, updated_at`,
		input.AmountML, start, end,
	).Scan(&f.ID, &f.AmountML, &f.StartTime, &f.EndTime, &f.CreatedAt, &f.UpdatedAt)
	return f, err
}

func (r *PostgresRepository) UpdateFeeding(id int, input FeedingInput) (Feeding, error) {
	start, _ := time.Parse(time.RFC3339, input.StartTime)
	end, _ := time.Parse(time.RFC3339, input.EndTime)
	var f Feeding
	err := r.DB.QueryRow(
		`UPDATE feedings SET amount_ml=$1, start_time=$2, end_time=$3, updated_at=NOW()
		 WHERE id=$4
		 RETURNING id, amount_ml, start_time, end_time, created_at, updated_at`,
		input.AmountML, start, end, id,
	).Scan(&f.ID, &f.AmountML, &f.StartTime, &f.EndTime, &f.CreatedAt, &f.UpdatedAt)
	return f, err
}

func (r *PostgresRepository) DeleteFeeding(id int) error {
	result, err := r.DB.Exec(`DELETE FROM feedings WHERE id=$1`, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("feeding not found")
	}
	return nil
}

func (r *PostgresRepository) GetDailyTotals(days int, tz string) ([]DailyTotal, error) {
	if days <= 0 {
		days = 7
	}
	if tz == "" {
		tz = "UTC"
	}
	// Convert UTC start_time to client timezone before extracting the date
	rows, err := r.DB.Query(
		`SELECT (start_time AT TIME ZONE $1)::date AS date, SUM(amount_ml) AS total_ml
		 FROM feedings
		 WHERE start_time >= NOW() - INTERVAL '1 day' * $2
		 GROUP BY (start_time AT TIME ZONE $1)::date
		 ORDER BY date`, tz, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var totals []DailyTotal
	for rows.Next() {
		var t DailyTotal
		if err := rows.Scan(&t.Date, &t.TotalML); err != nil {
			return nil, err
		}
		totals = append(totals, t)
	}
	return totals, rows.Err()
}

func (r *PostgresRepository) GetDailyTotalsByMonth(month string, tz string) ([]DailyTotal, error) {
	if tz == "" {
		tz = "UTC"
	}
	// Convert UTC start_time to client timezone before extracting date and month
	rows, err := r.DB.Query(
		`SELECT (start_time AT TIME ZONE $1)::date AS date, SUM(amount_ml) AS total_ml
		 FROM feedings
		 WHERE to_char(start_time AT TIME ZONE $1, 'YYYY-MM') = $2
		 GROUP BY (start_time AT TIME ZONE $1)::date
		 ORDER BY date`, tz, month)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var totals []DailyTotal
	for rows.Next() {
		var t DailyTotal
		if err := rows.Scan(&t.Date, &t.TotalML); err != nil {
			return nil, err
		}
		totals = append(totals, t)
	}
	return totals, rows.Err()
}

// ── Sleep repository methods ──

func (r *PostgresRepository) GetSleeps(dateFilter string, tz string) ([]Sleep, error) {
	if tz == "" {
		tz = "UTC"
	}
	query := `SELECT id, start_time, end_time, created_at, updated_at FROM sleeps`
	var args []interface{}
	if len(dateFilter) == 7 {
		query += ` WHERE to_char(start_time AT TIME ZONE $1, 'YYYY-MM') = $2`
		args = append(args, tz, dateFilter)
	} else if len(dateFilter) == 10 {
		query += ` WHERE (start_time AT TIME ZONE $1)::date = $2`
		args = append(args, tz, dateFilter)
	}
	query += ` ORDER BY start_time DESC`

	rows, err := r.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sleeps []Sleep
	for rows.Next() {
		var s Sleep
		if err := rows.Scan(&s.ID, &s.StartTime, &s.EndTime, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		sleeps = append(sleeps, s)
	}
	return sleeps, rows.Err()
}

func (r *PostgresRepository) GetLastSleep() (*Sleep, error) {
	var s Sleep
	err := r.DB.QueryRow(
		`SELECT id, start_time, end_time, created_at, updated_at
		 FROM sleeps ORDER BY created_at DESC LIMIT 1`,
	).Scan(&s.ID, &s.StartTime, &s.EndTime, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

func (r *PostgresRepository) CreateSleep(input SleepInput) (Sleep, error) {
	start, _ := time.Parse(time.RFC3339, input.StartTime)
	end, _ := time.Parse(time.RFC3339, input.EndTime)
	var s Sleep
	err := r.DB.QueryRow(
		`INSERT INTO sleeps (start_time, end_time) VALUES ($1, $2)
		 RETURNING id, start_time, end_time, created_at, updated_at`,
		start, end,
	).Scan(&s.ID, &s.StartTime, &s.EndTime, &s.CreatedAt, &s.UpdatedAt)
	return s, err
}

func (r *PostgresRepository) UpdateSleep(id int, input SleepInput) (Sleep, error) {
	start, _ := time.Parse(time.RFC3339, input.StartTime)
	end, _ := time.Parse(time.RFC3339, input.EndTime)
	var s Sleep
	err := r.DB.QueryRow(
		`UPDATE sleeps SET start_time=$1, end_time=$2, updated_at=NOW()
		 WHERE id=$3
		 RETURNING id, start_time, end_time, created_at, updated_at`,
		start, end, id,
	).Scan(&s.ID, &s.StartTime, &s.EndTime, &s.CreatedAt, &s.UpdatedAt)
	return s, err
}

func (r *PostgresRepository) DeleteSleep(id int) error {
	result, err := r.DB.Exec(`DELETE FROM sleeps WHERE id=$1`, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("sleep not found")
	}
	return nil
}

func (r *PostgresRepository) GetSleepDailyTotals(days int, tz string) ([]DailySleepTotal, error) {
	if days <= 0 {
		days = 7
	}
	if tz == "" {
		tz = "UTC"
	}
	rows, err := r.DB.Query(
		`SELECT (start_time AT TIME ZONE $1)::date AS date,
		        COALESCE(SUM(EXTRACT(EPOCH FROM (end_time - start_time)) / 60)::int, 0) AS total_minutes
		 FROM sleeps
		 WHERE start_time >= NOW() - INTERVAL '1 day' * $2
		 GROUP BY (start_time AT TIME ZONE $1)::date
		 ORDER BY date`, tz, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var totals []DailySleepTotal
	for rows.Next() {
		var t DailySleepTotal
		if err := rows.Scan(&t.Date, &t.TotalMinutes); err != nil {
			return nil, err
		}
		totals = append(totals, t)
	}
	return totals, rows.Err()
}

func (r *PostgresRepository) GetSleepDailyTotalsByMonth(month string, tz string) ([]DailySleepTotal, error) {
	if tz == "" {
		tz = "UTC"
	}
	rows, err := r.DB.Query(
		`SELECT (start_time AT TIME ZONE $1)::date AS date,
		        COALESCE(SUM(EXTRACT(EPOCH FROM (end_time - start_time)) / 60)::int, 0) AS total_minutes
		 FROM sleeps
		 WHERE to_char(start_time AT TIME ZONE $1, 'YYYY-MM') = $2
		 GROUP BY (start_time AT TIME ZONE $1)::date
		 ORDER BY date`, tz, month)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var totals []DailySleepTotal
	for rows.Next() {
		var t DailySleepTotal
		if err := rows.Scan(&t.Date, &t.TotalMinutes); err != nil {
			return nil, err
		}
		totals = append(totals, t)
	}
	return totals, rows.Err()
}
