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
	testDB.Exec("DROP TABLE IF EXISTS sleeps")
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

// ═══════════════════════════════════════════════════════════════════════════
// ── Sleep integration tests ──
// ═══════════════════════════════════════════════════════════════════════════

func cleanSleepsTable(t *testing.T) {
	t.Helper()
	_, err := testDB.Exec("DELETE FROM sleeps")
	if err != nil {
		t.Fatalf("Failed to clean sleeps table: %v", err)
	}
}

func seedSleep(t *testing.T, startTime, endTime string) Sleep {
	t.Helper()
	input := SleepInput{
		StartTime: startTime,
		EndTime:   endTime,
	}
	s, err := testRepo.CreateSleep(input)
	if err != nil {
		t.Fatalf("Failed to seed sleep: %v", err)
	}
	return s
}

func TestCreateSleep_Integration(t *testing.T) {
	cleanSleepsTable(t)
	input := SleepInput{
		StartTime: "2025-01-15T22:00:00Z",
		EndTime:   "2025-01-16T06:00:00Z",
	}
	s, err := testRepo.CreateSleep(input)
	if err != nil {
		t.Fatalf("CreateSleep failed: %v", err)
	}
	if s.ID == 0 {
		t.Error("expected non-zero ID")
	}
}

func TestGetSleeps_Integration(t *testing.T) {
	cleanSleepsTable(t)
	seedSleep(t, "2025-01-15T22:00:00Z", "2025-01-16T06:00:00Z")
	seedSleep(t, "2025-01-16T13:00:00Z", "2025-01-16T14:30:00Z")

	sleeps, err := testRepo.GetSleeps("", "")
	if err != nil {
		t.Fatalf("GetSleeps failed: %v", err)
	}
	if len(sleeps) != 2 {
		t.Errorf("expected 2 sleeps, got %d", len(sleeps))
	}
}

func TestUpdateSleep_Integration(t *testing.T) {
	cleanSleepsTable(t)
	created := seedSleep(t, "2025-01-15T22:00:00Z", "2025-01-16T06:00:00Z")

	input := SleepInput{
		StartTime: "2025-01-15T21:00:00Z",
		EndTime:   "2025-01-16T07:00:00Z",
	}
	updated, err := testRepo.UpdateSleep(created.ID, input)
	if err != nil {
		t.Fatalf("UpdateSleep failed: %v", err)
	}
	if updated.ID != created.ID {
		t.Errorf("expected same ID %d, got %d", created.ID, updated.ID)
	}
}

func TestDeleteSleep_Integration(t *testing.T) {
	cleanSleepsTable(t)
	created := seedSleep(t, "2025-01-15T22:00:00Z", "2025-01-16T06:00:00Z")

	err := testRepo.DeleteSleep(created.ID)
	if err != nil {
		t.Fatalf("DeleteSleep failed: %v", err)
	}

	sleeps, _ := testRepo.GetSleeps("", "")
	if len(sleeps) != 0 {
		t.Errorf("expected 0 sleeps after delete, got %d", len(sleeps))
	}
}

func TestDeleteSleep_NotFound_Integration(t *testing.T) {
	cleanSleepsTable(t)
	err := testRepo.DeleteSleep(99999)
	if err == nil {
		t.Error("expected error for non-existent sleep")
	}
}

func TestGetSleepDailyTotals_Integration(t *testing.T) {
	cleanSleepsTable(t)

	now := time.Now()
	today := now.Format("2006-01-02")
	// Two naps today: 1h + 1.5h = 150 minutes
	seedSleep(t,
		fmt.Sprintf("%sT10:00:00Z", today),
		fmt.Sprintf("%sT11:00:00Z", today))
	seedSleep(t,
		fmt.Sprintf("%sT14:00:00Z", today),
		fmt.Sprintf("%sT15:30:00Z", today))

	totals, err := testRepo.GetSleepDailyTotals(7, "")
	if err != nil {
		t.Fatalf("GetSleepDailyTotals failed: %v", err)
	}
	if len(totals) == 0 {
		t.Fatal("expected at least 1 daily sleep total")
	}

	found := false
	for _, total := range totals {
		if total.TotalMinutes == 150 {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected daily sleep total of 150 minutes, got %v", totals)
	}
}

func TestGetLastSleep_Integration(t *testing.T) {
	cleanSleepsTable(t)

	// No sleeps yet
	last, err := testRepo.GetLastSleep()
	if err != nil {
		t.Fatalf("GetLastSleep failed: %v", err)
	}
	if last != nil {
		t.Error("expected nil when no sleeps exist")
	}

	seedSleep(t, "2025-01-15T22:00:00Z", "2025-01-16T06:00:00Z")
	last, err = testRepo.GetLastSleep()
	if err != nil {
		t.Fatalf("GetLastSleep failed: %v", err)
	}
	if last == nil {
		t.Error("expected non-nil sleep")
	}
}
