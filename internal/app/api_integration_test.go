//go:build integration

package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func setupTestServer(t *testing.T) *Server {
	t.Helper()
	cleanTable(t)
	return NewServer(testRepo)
}

func TestAPI_CreateFeeding(t *testing.T) {
	srv := setupTestServer(t)
	body := `{"amount_ml":120,"start_time":"2025-01-15T08:00:00Z","end_time":"2025-01-15T08:15:00Z"}`

	req := httptest.NewRequest(http.MethodPost, "/api/feedings", strings.NewReader(body))
	w := httptest.NewRecorder()
	srv.HandleFeedings(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var f Feeding
	json.Unmarshal(w.Body.Bytes(), &f)
	if f.ID == 0 {
		t.Error("expected non-zero ID in response")
	}
	if f.AmountML != 120 {
		t.Errorf("expected amount_ml=120, got %d", f.AmountML)
	}
}

func TestAPI_ListFeedings(t *testing.T) {
	srv := setupTestServer(t)

	// Create two feedings
	for _, amt := range []int{100, 200} {
		body := fmt.Sprintf(`{"amount_ml":%d,"start_time":"2025-01-15T08:00:00Z","end_time":"2025-01-15T08:15:00Z"}`, amt)
		req := httptest.NewRequest(http.MethodPost, "/api/feedings", strings.NewReader(body))
		w := httptest.NewRecorder()
		srv.HandleFeedings(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("setup: expected 201, got %d", w.Code)
		}
	}

	// List feedings
	req := httptest.NewRequest(http.MethodGet, "/api/feedings", nil)
	w := httptest.NewRecorder()
	srv.HandleFeedings(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var feedings []Feeding
	json.Unmarshal(w.Body.Bytes(), &feedings)
	if len(feedings) != 2 {
		t.Errorf("expected 2 feedings, got %d", len(feedings))
	}
}

func TestAPI_ListFeedings_WithDateFilter(t *testing.T) {
	srv := setupTestServer(t)

	// Create feedings on different dates
	bodies := []string{
		`{"amount_ml":100,"start_time":"2025-01-15T08:00:00Z","end_time":"2025-01-15T08:15:00Z"}`,
		`{"amount_ml":200,"start_time":"2025-01-16T08:00:00Z","end_time":"2025-01-16T08:15:00Z"}`,
	}
	for _, body := range bodies {
		req := httptest.NewRequest(http.MethodPost, "/api/feedings", strings.NewReader(body))
		w := httptest.NewRecorder()
		srv.HandleFeedings(w, req)
	}

	// Filter by date
	req := httptest.NewRequest(http.MethodGet, "/api/feedings?date=2025-01-15", nil)
	w := httptest.NewRecorder()
	srv.HandleFeedings(w, req)

	var feedings []Feeding
	json.Unmarshal(w.Body.Bytes(), &feedings)
	if len(feedings) != 1 {
		t.Errorf("expected 1 feeding for 2025-01-15, got %d", len(feedings))
	}
}

func TestAPI_UpdateFeeding(t *testing.T) {
	srv := setupTestServer(t)

	// Create a feeding
	createBody := `{"amount_ml":100,"start_time":"2025-01-15T08:00:00Z","end_time":"2025-01-15T08:15:00Z"}`
	req := httptest.NewRequest(http.MethodPost, "/api/feedings", strings.NewReader(createBody))
	w := httptest.NewRecorder()
	srv.HandleFeedings(w, req)

	var created Feeding
	json.Unmarshal(w.Body.Bytes(), &created)

	// Update it
	updateBody := `{"amount_ml":250,"start_time":"2025-01-15T08:00:00Z","end_time":"2025-01-15T08:30:00Z"}`
	req = httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/feedings/%d", created.ID), strings.NewReader(updateBody))
	w = httptest.NewRecorder()
	srv.HandleFeedingByID(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var updated Feeding
	json.Unmarshal(w.Body.Bytes(), &updated)
	if updated.AmountML != 250 {
		t.Errorf("expected amount_ml=250, got %d", updated.AmountML)
	}
}

func TestAPI_DeleteFeeding(t *testing.T) {
	srv := setupTestServer(t)

	// Create a feeding
	createBody := `{"amount_ml":100,"start_time":"2025-01-15T08:00:00Z","end_time":"2025-01-15T08:15:00Z"}`
	req := httptest.NewRequest(http.MethodPost, "/api/feedings", strings.NewReader(createBody))
	w := httptest.NewRecorder()
	srv.HandleFeedings(w, req)

	var created Feeding
	json.Unmarshal(w.Body.Bytes(), &created)

	// Delete it
	req = httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/feedings/%d", created.ID), nil)
	w = httptest.NewRecorder()
	srv.HandleFeedingByID(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}

	// Verify it's gone
	req = httptest.NewRequest(http.MethodGet, "/api/feedings", nil)
	w = httptest.NewRecorder()
	srv.HandleFeedings(w, req)

	var feedings []Feeding
	json.Unmarshal(w.Body.Bytes(), &feedings)
	if len(feedings) != 0 {
		t.Errorf("expected 0 feedings after delete, got %d", len(feedings))
	}
}

func TestAPI_DeleteFeeding_NotFound(t *testing.T) {
	srv := setupTestServer(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/feedings/99999", nil)
	w := httptest.NewRecorder()
	srv.HandleFeedingByID(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestAPI_CreateFeeding_InvalidJSON(t *testing.T) {
	srv := setupTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/feedings", strings.NewReader("{bad"))
	w := httptest.NewRecorder()
	srv.HandleFeedings(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestAPI_CreateFeeding_ValidationError(t *testing.T) {
	srv := setupTestServer(t)

	body := `{"amount_ml":0,"start_time":"2025-01-15T08:00:00Z","end_time":"2025-01-15T08:15:00Z"}`
	req := httptest.NewRequest(http.MethodPost, "/api/feedings", strings.NewReader(body))
	w := httptest.NewRecorder()
	srv.HandleFeedings(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestAPI_DailyTotals(t *testing.T) {
	srv := setupTestServer(t)

	now := time.Now()
	today := now.Format("2006-01-02")

	// Create feedings for today
	for _, amt := range []int{100, 200, 150} {
		body := fmt.Sprintf(`{"amount_ml":%d,"start_time":"%sT08:00:00Z","end_time":"%sT08:15:00Z"}`, amt, today, today)
		req := httptest.NewRequest(http.MethodPost, "/api/feedings", strings.NewReader(body))
		w := httptest.NewRecorder()
		srv.HandleFeedings(w, req)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/feedings/daily?days=7", nil)
	w := httptest.NewRecorder()
	srv.HandleDailyTotals(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var totals []DailyTotal
	json.Unmarshal(w.Body.Bytes(), &totals)
	if len(totals) == 0 {
		t.Fatal("expected at least 1 daily total")
	}

	found := false
	for _, total := range totals {
		if total.TotalML == 450 {
			found = true
		}
	}
	if !found {
		t.Errorf("expected daily total of 450, got %v", totals)
	}
}

func TestAPI_MethodNotAllowed(t *testing.T) {
	srv := setupTestServer(t)

	req := httptest.NewRequest(http.MethodPatch, "/api/feedings", nil)
	w := httptest.NewRecorder()
	srv.HandleFeedings(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}
