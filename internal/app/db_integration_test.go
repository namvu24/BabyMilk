//go:build integration

package app

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

var testDB *sql.DB
var testRepo *PostgresRepository

func TestMain(m *testing.M) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://testuser:testpass@localhost:5433/babymilk_test?sslmode=disable"
	}

	var err error
	testDB, err = sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Failed to connect to test database: %v", err)
	}

	// Wait for database to be ready
	for i := 0; i < 30; i++ {
		if err = testDB.Ping(); err == nil {
			break
		}
		time.Sleep(time.Second)
	}
	if err != nil {
		log.Fatalf("Test database not ready: %v", err)
	}

	RunMigrations(testDB)
	testRepo = NewPostgresRepository(testDB)

	code := m.Run()

	testDB.Exec("DROP TABLE IF EXISTS feedings")
	testDB.Close()
	os.Exit(code)
}

func cleanTable(t *testing.T) {
	t.Helper()
	_, err := testDB.Exec("DELETE FROM feedings")
	if err != nil {
		t.Fatalf("Failed to clean feedings table: %v", err)
	}
}

func seedFeeding(t *testing.T, amountML int, startTime, endTime string) Feeding {
	t.Helper()
	input := FeedingInput{
		AmountML:  amountML,
		StartTime: startTime,
		EndTime:   endTime,
	}
	f, err := testRepo.CreateFeeding(input)
	if err != nil {
		t.Fatalf("Failed to seed feeding: %v", err)
	}
	return f
}

func TestCreateFeeding_Integration(t *testing.T) {
	cleanTable(t)
	input := FeedingInput{
		AmountML:  150,
		StartTime: "2025-01-15T08:00:00Z",
		EndTime:   "2025-01-15T08:20:00Z",
	}
	f, err := testRepo.CreateFeeding(input)
	if err != nil {
		t.Fatalf("CreateFeeding failed: %v", err)
	}
	if f.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if f.AmountML != 150 {
		t.Errorf("expected amount_ml=150, got %d", f.AmountML)
	}
}

func TestGetFeedings_Integration(t *testing.T) {
	cleanTable(t)
	seedFeeding(t, 100, "2025-01-15T08:00:00Z", "2025-01-15T08:10:00Z")
	seedFeeding(t, 200, "2025-01-15T09:00:00Z", "2025-01-15T09:15:00Z")

	feedings, err := testRepo.GetFeedings("", "")
	if err != nil {
		t.Fatalf("GetFeedings failed: %v", err)
	}
	if len(feedings) != 2 {
		t.Errorf("expected 2 feedings, got %d", len(feedings))
	}
}

func TestGetFeedings_WithDateFilter_Integration(t *testing.T) {
	cleanTable(t)
	seedFeeding(t, 100, "2025-01-15T08:00:00Z", "2025-01-15T08:10:00Z")
	seedFeeding(t, 200, "2025-01-16T09:00:00Z", "2025-01-16T09:15:00Z")

	feedings, err := testRepo.GetFeedings("2025-01-15", "")
	if err != nil {
		t.Fatalf("GetFeedings with date filter failed: %v", err)
	}
	if len(feedings) != 1 {
		t.Errorf("expected 1 feeding for 2025-01-15, got %d", len(feedings))
	}
}

func TestUpdateFeeding_Integration(t *testing.T) {
	cleanTable(t)
	created := seedFeeding(t, 100, "2025-01-15T08:00:00Z", "2025-01-15T08:10:00Z")

	input := FeedingInput{
		AmountML:  250,
		StartTime: "2025-01-15T08:00:00Z",
		EndTime:   "2025-01-15T08:30:00Z",
	}
	updated, err := testRepo.UpdateFeeding(created.ID, input)
	if err != nil {
		t.Fatalf("UpdateFeeding failed: %v", err)
	}
	if updated.AmountML != 250 {
		t.Errorf("expected amount_ml=250, got %d", updated.AmountML)
	}
	if updated.ID != created.ID {
		t.Errorf("expected same ID %d, got %d", created.ID, updated.ID)
	}
}

func TestDeleteFeeding_Integration(t *testing.T) {
	cleanTable(t)
	created := seedFeeding(t, 100, "2025-01-15T08:00:00Z", "2025-01-15T08:10:00Z")

	err := testRepo.DeleteFeeding(created.ID)
	if err != nil {
		t.Fatalf("DeleteFeeding failed: %v", err)
	}

	feedings, _ := testRepo.GetFeedings("", "")
	if len(feedings) != 0 {
		t.Errorf("expected 0 feedings after delete, got %d", len(feedings))
	}
}

func TestDeleteFeeding_NotFound_Integration(t *testing.T) {
	cleanTable(t)
	err := testRepo.DeleteFeeding(99999)
	if err == nil {
		t.Error("expected error for non-existent feeding")
	}
	if err.Error() != "feeding not found" {
		t.Errorf("expected 'feeding not found', got '%s'", err.Error())
	}
}

func TestGetDailyTotals_Integration(t *testing.T) {
	cleanTable(t)

	now := time.Now()
	today := now.Format("2006-01-02")
	seedFeeding(t, 100,
		fmt.Sprintf("%sT08:00:00Z", today),
		fmt.Sprintf("%sT08:10:00Z", today))
	seedFeeding(t, 200,
		fmt.Sprintf("%sT09:00:00Z", today),
		fmt.Sprintf("%sT09:15:00Z", today))

	totals, err := testRepo.GetDailyTotals(7, "")
	if err != nil {
		t.Fatalf("GetDailyTotals failed: %v", err)
	}
	if len(totals) == 0 {
		t.Fatal("expected at least 1 daily total")
	}

	found := false
	for _, total := range totals {
		if total.TotalML == 300 {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected daily total of 300ml, got %v", totals)
	}
}
