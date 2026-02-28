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

func (r *PostgresRepository) GetFeedings(dateFilter string) ([]Feeding, error) {
	query := `SELECT id, amount_ml, start_time, end_time, created_at, updated_at FROM feedings`
	var args []interface{}
	if len(dateFilter) == 7 {
		// YYYY-MM month filter
		query += ` WHERE to_char(start_time AT TIME ZONE 'UTC', 'YYYY-MM') = $1`
		args = append(args, dateFilter)
	} else if len(dateFilter) == 10 {
		// YYYY-MM-DD date filter
		query += ` WHERE start_time::date = $1`
		args = append(args, dateFilter)
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

func (r *PostgresRepository) GetDailyTotals(days int) ([]DailyTotal, error) {
	if days <= 0 {
		days = 7
	}
	rows, err := r.DB.Query(
		`SELECT start_time::date AS date, SUM(amount_ml) AS total_ml
		 FROM feedings
		 WHERE start_time >= NOW() - INTERVAL '1 day' * $1
		 GROUP BY start_time::date
		 ORDER BY date`, days)
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
