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
		log.Fatalf("Failed to run migrations: %v", err)
	}
	log.Println("Database migrations completed")
}
