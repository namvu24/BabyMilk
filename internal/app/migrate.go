package app

import (
	"database/sql"
	"fmt"
	"log"
)

func RunMigrations(db *sql.DB) {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS feedings (
			id         SERIAL PRIMARY KEY,
			amount_ml  INTEGER NOT NULL,
			end_time   TIMESTAMPTZ NOT NULL,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW()
		)
	`)
	if err != nil {
		log.Fatalf("Failed to run feedings migration: %v", err)
	}

	// Remove start_time from feedings if it exists
	_, _ = db.Exec(`ALTER TABLE feedings DROP COLUMN IF EXISTS start_time`)

	// Ensure index on feedings(end_time) for efficient filtering and sorting
	_, _ = db.Exec(`CREATE INDEX IF NOT EXISTS idx_feedings_end_time ON feedings (end_time)`)

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS sleeps (
			id         SERIAL PRIMARY KEY,
			start_time TIMESTAMPTZ NOT NULL,
			end_time   TIMESTAMPTZ NOT NULL,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW()
		)
	`)
	if err != nil {
		log.Fatalf("Failed to run sleeps migration: %v", err)
	}

	// Add new columns to sleeps (idempotent)
	for _, col := range []struct{ name, def string }{
		{"sleep_type", "VARCHAR(10) DEFAULT 'nap'"},
		{"status", "VARCHAR(10) DEFAULT 'completed'"},
	} {
		_, err = db.Exec(fmt.Sprintf(
			`ALTER TABLE sleeps ADD COLUMN IF NOT EXISTS %s %s`, col.name, col.def))
		if err != nil {
			log.Fatalf("Failed to add column %s to sleeps: %v", col.name, err)
		}
	}

	// Allow end_time to be NULL for active sleep sessions
	if _, err = db.Exec(`ALTER TABLE sleeps ALTER COLUMN end_time DROP NOT NULL`); err != nil {
		log.Printf("Warning: failed to alter end_time nullable: %v", err)
	}

	// Ensure new columns are NOT NULL
	if _, err = db.Exec(`ALTER TABLE sleeps ALTER COLUMN sleep_type SET NOT NULL`); err != nil {
		log.Printf("Warning: failed to set sleep_type NOT NULL: %v", err)
	}
	if _, err = db.Exec(`ALTER TABLE sleeps ALTER COLUMN status SET NOT NULL`); err != nil {
		log.Printf("Warning: failed to set status NOT NULL: %v", err)
	}

	// Prevent multiple active sleep sessions
	_, _ = db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_sleeps_single_active ON sleeps (status) WHERE status = 'active'`)
	_, _ = db.Exec(`CREATE INDEX IF NOT EXISTS idx_sleeps_status ON sleeps (status)`)
	_, _ = db.Exec(`CREATE INDEX IF NOT EXISTS idx_sleeps_start_time ON sleeps (start_time)`)

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS baby_profile (
			id            SERIAL PRIMARY KEY,
			date_of_birth DATE NOT NULL,
			created_at    TIMESTAMPTZ DEFAULT NOW(),
			updated_at    TIMESTAMPTZ DEFAULT NOW()
		)
	`)
	if err != nil {
		log.Fatalf("Failed to run baby_profile migration: %v", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS development_cache (
			id          SERIAL PRIMARY KEY,
			week_number INTEGER UNIQUE NOT NULL,
			content     JSONB NOT NULL,
			created_at  TIMESTAMPTZ DEFAULT NOW(),
			updated_at  TIMESTAMPTZ DEFAULT NOW()
		)
	`)
	if err != nil {
		log.Fatalf("Failed to run development_cache migration: %v", err)
	}

	// Add new columns to baby_profile (idempotent)
	for _, col := range []struct{ name, def string }{
		{"name", "VARCHAR(255)"},
		{"gender", "VARCHAR(10)"},
		{"milk_type", "VARCHAR(10) DEFAULT 'formula'"},
	} {
		_, err = db.Exec(fmt.Sprintf(
			`ALTER TABLE baby_profile ADD COLUMN IF NOT EXISTS %s %s`, col.name, col.def))
		if err != nil {
			log.Fatalf("Failed to add column %s to baby_profile: %v", col.name, err)
		}
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS growth_measurements (
			id                    SERIAL PRIMARY KEY,
			date                  DATE NOT NULL,
			weight_kg             NUMERIC(5,2),
			length_cm             NUMERIC(5,1),
			head_circumference_cm NUMERIC(4,1),
			notes                 TEXT,
			created_at            TIMESTAMPTZ DEFAULT NOW(),
			updated_at            TIMESTAMPTZ DEFAULT NOW()
		)
	`)
	if err != nil {
		log.Fatalf("Failed to run growth_measurements migration: %v", err)
	}

	// Index on date for growth measurements
	_, _ = db.Exec(`CREATE INDEX IF NOT EXISTS idx_growth_measurements_date ON growth_measurements (date)`)

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS insights_cache (
			id         SERIAL PRIMARY KEY,
			cache_key  VARCHAR(255) UNIQUE NOT NULL,
			content    JSONB NOT NULL,
			expires_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ DEFAULT NOW()
		)
	`)
	if err != nil {
		log.Fatalf("Failed to run insights_cache migration: %v", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS diapers (
			id         SERIAL PRIMARY KEY,
			type       VARCHAR(10) NOT NULL,
			time       TIMESTAMPTZ NOT NULL,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW()
		)
	`)
	if err != nil {
		log.Fatalf("Failed to run diapers migration: %v", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS baths (
			id         SERIAL PRIMARY KEY,
			time       TIMESTAMPTZ NOT NULL,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW()
		)
	`)
	if err != nil {
		log.Fatalf("Failed to run baths migration: %v", err)
	}

	log.Println("Database migrations completed")
}
