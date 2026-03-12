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
			start_time TIMESTAMPTZ NOT NULL,
			end_time   TIMESTAMPTZ NOT NULL,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW()
		)
	`)
	if err != nil {
		log.Fatalf("Failed to run feedings migration: %v", err)
	}

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
