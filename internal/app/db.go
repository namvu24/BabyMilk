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

// ── Baby profile & development cache ──

func (r *PostgresRepository) GetBabyProfile() (*BabyProfile, error) {
	var p BabyProfile
	var name, gender, milkType sql.NullString
	err := r.DB.QueryRow(`SELECT id, date_of_birth, name, gender, milk_type, created_at, updated_at FROM baby_profile ORDER BY id LIMIT 1`).
		Scan(&p.ID, &p.DateOfBirth, &name, &gender, &milkType, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, nil
		}
		return nil, err
	}
	if name.Valid {
		p.Name = name.String
	}
	if gender.Valid {
		p.Gender = gender.String
	}
	if milkType.Valid {
		p.MilkType = milkType.String
	}
	return &p, nil
}

func (r *PostgresRepository) SaveBabyProfile(input BabyProfileInput) (*BabyProfile, error) {
	var p BabyProfile
	var name, gender, milkType sql.NullString
	// Try update first (single-row table pattern)
	err := r.DB.QueryRow(`
		UPDATE baby_profile SET date_of_birth = $1, name = $2, gender = $3, milk_type = $4, updated_at = NOW()
		WHERE id = (SELECT id FROM baby_profile ORDER BY id LIMIT 1)
		RETURNING id, date_of_birth, name, gender, milk_type, created_at, updated_at`,
		input.DateOfBirth, toNullString(input.Name), toNullString(input.Gender), toNullString(input.MilkType)).
		Scan(&p.ID, &p.DateOfBirth, &name, &gender, &milkType, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			// No profile exists yet — insert
			err = r.DB.QueryRow(`
				INSERT INTO baby_profile (date_of_birth, name, gender, milk_type) VALUES ($1, $2, $3, $4)
				RETURNING id, date_of_birth, name, gender, milk_type, created_at, updated_at`,
				input.DateOfBirth, toNullString(input.Name), toNullString(input.Gender), toNullString(input.MilkType)).
				Scan(&p.ID, &p.DateOfBirth, &name, &gender, &milkType, &p.CreatedAt, &p.UpdatedAt)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	if name.Valid {
		p.Name = name.String
	}
	if gender.Valid {
		p.Gender = gender.String
	}
	if milkType.Valid {
		p.MilkType = milkType.String
	}
	return &p, nil
}

func toNullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

func (r *PostgresRepository) GetDevelopmentCache(weekNumber int) (*DevelopmentContent, error) {
	var c DevelopmentContent
	err := r.DB.QueryRow(`SELECT id, week_number, content, created_at, updated_at FROM development_cache WHERE week_number = $1`, weekNumber).
		Scan(&c.ID, &c.WeekNumber, &c.Content, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, nil
		}
		return nil, err
	}
	return &c, nil
}

func (r *PostgresRepository) SaveDevelopmentCache(weekNumber int, content string) error {
	_, err := r.DB.Exec(`
		INSERT INTO development_cache (week_number, content) VALUES ($1, $2)
		ON CONFLICT (week_number) DO UPDATE SET content = $2, updated_at = NOW()`,
		weekNumber, content)
	return err
}

// ── Growth measurement methods ──

func (r *PostgresRepository) GetGrowthMeasurements(limit int) ([]GrowthMeasurement, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.DB.Query(
		`SELECT id, date, weight_kg, length_cm, head_circumference_cm, notes, created_at, updated_at
		 FROM growth_measurements ORDER BY date DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanGrowthRows(rows)
}

func (r *PostgresRepository) GetGrowthMeasurementsByRange(from, to time.Time) ([]GrowthMeasurement, error) {
	rows, err := r.DB.Query(
		`SELECT id, date, weight_kg, length_cm, head_circumference_cm, notes, created_at, updated_at
		 FROM growth_measurements WHERE date >= $1 AND date <= $2 ORDER BY date`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanGrowthRows(rows)
}

func (r *PostgresRepository) GetLatestGrowthMeasurement() (*GrowthMeasurement, error) {
	var g GrowthMeasurement
	var headCirc sql.NullFloat64
	var notes sql.NullString
	err := r.DB.QueryRow(
		`SELECT id, date, weight_kg, length_cm, head_circumference_cm, notes, created_at, updated_at
		 FROM growth_measurements ORDER BY date DESC LIMIT 1`).
		Scan(&g.ID, &g.Date, &g.WeightKg, &g.LengthCm, &headCirc, &notes, &g.CreatedAt, &g.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if headCirc.Valid {
		g.HeadCircumferenceCm = &headCirc.Float64
	}
	if notes.Valid {
		g.Notes = notes.String
	}
	return &g, nil
}

func (r *PostgresRepository) CreateGrowthMeasurement(input GrowthMeasurementInput) (GrowthMeasurement, error) {
	var g GrowthMeasurement
	var headCirc sql.NullFloat64
	var notesScan sql.NullString
	if input.HeadCircumferenceCm != nil {
		headCirc = sql.NullFloat64{Float64: *input.HeadCircumferenceCm, Valid: true}
	}
	err := r.DB.QueryRow(
		`INSERT INTO growth_measurements (date, weight_kg, length_cm, head_circumference_cm, notes)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, date, weight_kg, length_cm, head_circumference_cm, notes, created_at, updated_at`,
		input.Date, input.WeightKg, input.LengthCm, headCirc, toNullString(input.Notes)).
		Scan(&g.ID, &g.Date, &g.WeightKg, &g.LengthCm, &headCirc, &notesScan, &g.CreatedAt, &g.UpdatedAt)
	if err != nil {
		return GrowthMeasurement{}, err
	}
	if headCirc.Valid {
		g.HeadCircumferenceCm = &headCirc.Float64
	}
	if notesScan.Valid {
		g.Notes = notesScan.String
	}
	return g, nil
}

func (r *PostgresRepository) UpdateGrowthMeasurement(id int, input GrowthMeasurementInput) (GrowthMeasurement, error) {
	var g GrowthMeasurement
	var headCirc sql.NullFloat64
	var notesScan sql.NullString
	if input.HeadCircumferenceCm != nil {
		headCirc = sql.NullFloat64{Float64: *input.HeadCircumferenceCm, Valid: true}
	}
	err := r.DB.QueryRow(
		`UPDATE growth_measurements SET date=$1, weight_kg=$2, length_cm=$3, head_circumference_cm=$4, notes=$5, updated_at=NOW()
		 WHERE id=$6
		 RETURNING id, date, weight_kg, length_cm, head_circumference_cm, notes, created_at, updated_at`,
		input.Date, input.WeightKg, input.LengthCm, headCirc, toNullString(input.Notes), id).
		Scan(&g.ID, &g.Date, &g.WeightKg, &g.LengthCm, &headCirc, &notesScan, &g.CreatedAt, &g.UpdatedAt)
	if err != nil {
		return GrowthMeasurement{}, err
	}
	if headCirc.Valid {
		g.HeadCircumferenceCm = &headCirc.Float64
	}
	if notesScan.Valid {
		g.Notes = notesScan.String
	}
	return g, nil
}

func (r *PostgresRepository) DeleteGrowthMeasurement(id int) error {
	result, err := r.DB.Exec(`DELETE FROM growth_measurements WHERE id=$1`, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("growth measurement not found")
	}
	return nil
}

func scanGrowthRows(rows *sql.Rows) ([]GrowthMeasurement, error) {
	var measurements []GrowthMeasurement
	for rows.Next() {
		var g GrowthMeasurement
		var headCirc sql.NullFloat64
		var notes sql.NullString
		if err := rows.Scan(&g.ID, &g.Date, &g.WeightKg, &g.LengthCm, &headCirc, &notes, &g.CreatedAt, &g.UpdatedAt); err != nil {
			return nil, err
		}
		if headCirc.Valid {
			g.HeadCircumferenceCm = &headCirc.Float64
		}
		if notes.Valid {
			g.Notes = notes.String
		}
		measurements = append(measurements, g)
	}
	return measurements, rows.Err()
}

// ── Insight cache methods ──

func (r *PostgresRepository) GetInsightCache(key string) (*InsightCache, error) {
	var c InsightCache
	err := r.DB.QueryRow(
		`SELECT id, cache_key, content, expires_at, created_at FROM insights_cache
		 WHERE cache_key = $1 AND (expires_at IS NULL OR expires_at > NOW())`, key).
		Scan(&c.ID, &c.CacheKey, &c.Content, &c.ExpiresAt, &c.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &c, nil
}

func (r *PostgresRepository) SaveInsightCache(key, content string, expiresAt time.Time) error {
	_, err := r.DB.Exec(`
		INSERT INTO insights_cache (cache_key, content, expires_at) VALUES ($1, $2, $3)
		ON CONFLICT (cache_key) DO UPDATE SET content = $2, expires_at = $3, created_at = NOW()`,
		key, content, expiresAt)
	return err
}

func (r *PostgresRepository) InvalidateInsightCache() error {
	_, err := r.DB.Exec(`DELETE FROM insights_cache`)
	return err
}

// ── Aggregation methods for insights ──

func (r *PostgresRepository) GetFeedingDailyAvg(days int) (int, error) {
	if days <= 0 {
		days = 7
	}
	var avg sql.NullFloat64
	err := r.DB.QueryRow(`
		SELECT AVG(daily_total) FROM (
			SELECT SUM(amount_ml) AS daily_total
			FROM feedings
			WHERE start_time >= NOW() - INTERVAL '1 day' * $1
			GROUP BY start_time::date
		) sub`, days).Scan(&avg)
	if err != nil {
		return 0, err
	}
	if !avg.Valid {
		return 0, nil
	}
	return int(avg.Float64), nil
}

func (r *PostgresRepository) GetSleepDailyAvg(days int) (int, error) {
	if days <= 0 {
		days = 7
	}
	var avg sql.NullFloat64
	err := r.DB.QueryRow(`
		SELECT AVG(daily_total) FROM (
			SELECT SUM(EXTRACT(EPOCH FROM (end_time - start_time)) / 60)::int AS daily_total
			FROM sleeps
			WHERE start_time >= NOW() - INTERVAL '1 day' * $1
			GROUP BY start_time::date
		) sub`, days).Scan(&avg)
	if err != nil {
		return 0, err
	}
	if !avg.Valid {
		return 0, nil
	}
	return int(avg.Float64), nil
}
