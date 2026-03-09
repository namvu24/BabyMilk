package app

import (
	"database/sql"
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

	log.Println("Database migrations completed")
}
