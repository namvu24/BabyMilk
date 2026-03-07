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

// mockRepository implements Repository for testing.
type mockRepository struct {
	feedings    []Feeding
	dailyTotals []DailyTotal
	getErr      error
	createErr   error
	updateErr   error
	deleteErr   error
	dailyErr    error
	lastDate    string
	lastDays    int
	lastID      int
	lastInput   FeedingInput
}

func (m *mockRepository) GetFeedings(date string) ([]Feeding, error) {
	m.lastDate = date
	return m.feedings, m.getErr
}

func (m *mockRepository) CreateFeeding(input FeedingInput) (Feeding, error) {
	m.lastInput = input
	if m.createErr != nil {
		return Feeding{}, m.createErr
	}
	f := Feeding{
		ID:        1,
		AmountML:  input.AmountML,
		StartTime: time.Now(),
		EndTime:   time.Now(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	return f, nil
}

func (m *mockRepository) UpdateFeeding(id int, input FeedingInput) (Feeding, error) {
	m.lastID = id
	m.lastInput = input
	if m.updateErr != nil {
		return Feeding{}, m.updateErr
	}
	f := Feeding{
		ID:        id,
		AmountML:  input.AmountML,
		StartTime: time.Now(),
		EndTime:   time.Now(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	return f, nil
}

func (m *mockRepository) DeleteFeeding(id int) error {
	m.lastID = id
	return m.deleteErr
}

func (m *mockRepository) GetDailyTotals(days int) ([]DailyTotal, error) {
	m.lastDays = days
	return m.dailyTotals, m.dailyErr
}

func (m *mockRepository) GetDailyTotalsByMonth(month string) ([]DailyTotal, error) {
	m.lastDate = month
	return m.dailyTotals, m.dailyErr
}

func (m *mockRepository) GetLastFeeding() (*Feeding, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	if len(m.feedings) == 0 {
		return nil, nil
	}
	return &m.feedings[0], nil
}

func validFeedingJSON() string {
	return `{"amount_ml":120,"start_time":"2025-01-15T08:00:00Z","end_time":"2025-01-15T08:15:00Z"}`
}

// --- HandleFeedings tests ---

func TestHandleFeedings_GET_Success(t *testing.T) {
	mock := &mockRepository{
		feedings: []Feeding{
			{ID: 1, AmountML: 100},
			{ID: 2, AmountML: 200},
		},
	}
	srv := NewServer(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/feedings", nil)
	w := httptest.NewRecorder()
	srv.HandleFeedings(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var result []Feeding
	json.Unmarshal(w.Body.Bytes(), &result)
	if len(result) != 2 {
		t.Errorf("expected 2 feedings, got %d", len(result))
	}
}

func TestHandleFeedings_GET_WithDateFilter(t *testing.T) {
	mock := &mockRepository{}
	srv := NewServer(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/feedings?date=2025-01-15", nil)
	w := httptest.NewRecorder()
	srv.HandleFeedings(w, req)

	if mock.lastDate != "2025-01-15" {
		t.Errorf("expected date filter '2025-01-15', got '%s'", mock.lastDate)
	}
}

func TestHandleFeedings_GET_EmptyList(t *testing.T) {
	mock := &mockRepository{}
	srv := NewServer(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/feedings", nil)
	w := httptest.NewRecorder()
	srv.HandleFeedings(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var result []Feeding
	json.Unmarshal(w.Body.Bytes(), &result)
	if len(result) != 0 {
		t.Errorf("expected empty array, got %d feedings", len(result))
	}
}

func TestHandleFeedings_GET_RepoError(t *testing.T) {
	mock := &mockRepository{getErr: fmt.Errorf("db connection lost")}
	srv := NewServer(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/feedings", nil)
	w := httptest.NewRecorder()
	srv.HandleFeedings(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestHandleFeedings_POST_Success(t *testing.T) {
	mock := &mockRepository{}
	srv := NewServer(mock)

	body := strings.NewReader(validFeedingJSON())
	req := httptest.NewRequest(http.MethodPost, "/api/feedings", body)
	w := httptest.NewRecorder()
	srv.HandleFeedings(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", w.Code)
	}
	if mock.lastInput.AmountML != 120 {
		t.Errorf("expected amount_ml=120, got %d", mock.lastInput.AmountML)
	}
}

func TestHandleFeedings_POST_InvalidJSON(t *testing.T) {
	mock := &mockRepository{}
	srv := NewServer(mock)

	body := strings.NewReader("{invalid")
	req := httptest.NewRequest(http.MethodPost, "/api/feedings", body)
	w := httptest.NewRecorder()
	srv.HandleFeedings(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleFeedings_POST_ValidationError(t *testing.T) {
	mock := &mockRepository{}
	srv := NewServer(mock)

	body := strings.NewReader(`{"amount_ml":0,"start_time":"2025-01-15T08:00:00Z","end_time":"2025-01-15T08:15:00Z"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/feedings", body)
	w := httptest.NewRecorder()
	srv.HandleFeedings(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleFeedings_POST_RepoError(t *testing.T) {
	mock := &mockRepository{createErr: fmt.Errorf("insert failed")}
	srv := NewServer(mock)

	body := strings.NewReader(validFeedingJSON())
	req := httptest.NewRequest(http.MethodPost, "/api/feedings", body)
	w := httptest.NewRecorder()
	srv.HandleFeedings(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestHandleFeedings_MethodNotAllowed(t *testing.T) {
	mock := &mockRepository{}
	srv := NewServer(mock)

	req := httptest.NewRequest(http.MethodDelete, "/api/feedings", nil)
	w := httptest.NewRecorder()
	srv.HandleFeedings(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

// --- HandleFeedingByID tests ---

func TestHandleFeedingByID_PUT_Success(t *testing.T) {
	mock := &mockRepository{}
	srv := NewServer(mock)

	body := strings.NewReader(validFeedingJSON())
	req := httptest.NewRequest(http.MethodPut, "/api/feedings/42", body)
	w := httptest.NewRecorder()
	srv.HandleFeedingByID(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if mock.lastID != 42 {
		t.Errorf("expected id=42, got %d", mock.lastID)
	}
}

func TestHandleFeedingByID_PUT_InvalidJSON(t *testing.T) {
	mock := &mockRepository{}
	srv := NewServer(mock)

	body := strings.NewReader("{bad")
	req := httptest.NewRequest(http.MethodPut, "/api/feedings/1", body)
	w := httptest.NewRecorder()
	srv.HandleFeedingByID(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleFeedingByID_PUT_ValidationError(t *testing.T) {
	mock := &mockRepository{}
	srv := NewServer(mock)

	body := strings.NewReader(`{"amount_ml":-1,"start_time":"2025-01-15T08:00:00Z","end_time":"2025-01-15T08:15:00Z"}`)
	req := httptest.NewRequest(http.MethodPut, "/api/feedings/1", body)
	w := httptest.NewRecorder()
	srv.HandleFeedingByID(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleFeedingByID_DELETE_Success(t *testing.T) {
	mock := &mockRepository{}
	srv := NewServer(mock)

	req := httptest.NewRequest(http.MethodDelete, "/api/feedings/5", nil)
	w := httptest.NewRecorder()
	srv.HandleFeedingByID(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
	if mock.lastID != 5 {
		t.Errorf("expected id=5, got %d", mock.lastID)
	}
}

func TestHandleFeedingByID_DELETE_NotFound(t *testing.T) {
	mock := &mockRepository{deleteErr: fmt.Errorf("feeding not found")}
	srv := NewServer(mock)

	req := httptest.NewRequest(http.MethodDelete, "/api/feedings/999", nil)
	w := httptest.NewRecorder()
	srv.HandleFeedingByID(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestHandleFeedingByID_DELETE_RepoError(t *testing.T) {
	mock := &mockRepository{deleteErr: fmt.Errorf("db error")}
	srv := NewServer(mock)

	req := httptest.NewRequest(http.MethodDelete, "/api/feedings/1", nil)
	w := httptest.NewRecorder()
	srv.HandleFeedingByID(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestHandleFeedingByID_InvalidID(t *testing.T) {
	mock := &mockRepository{}
	srv := NewServer(mock)

	req := httptest.NewRequest(http.MethodPut, "/api/feedings/abc", nil)
	w := httptest.NewRecorder()
	srv.HandleFeedingByID(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleFeedingByID_MethodNotAllowed(t *testing.T) {
	mock := &mockRepository{}
	srv := NewServer(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/feedings/1", nil)
	w := httptest.NewRecorder()
	srv.HandleFeedingByID(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

// --- HandleDailyTotals tests ---

func TestHandleDailyTotals_GET_Success(t *testing.T) {
	mock := &mockRepository{
		dailyTotals: []DailyTotal{
			{Date: "2025-01-15", TotalML: 500},
			{Date: "2025-01-14", TotalML: 450},
		},
	}
	srv := NewServer(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/feedings/daily", nil)
	w := httptest.NewRecorder()
	srv.HandleDailyTotals(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if mock.lastDays != 7 {
		t.Errorf("expected default days=7, got %d", mock.lastDays)
	}
}

func TestHandleDailyTotals_GET_WithDaysParam(t *testing.T) {
	mock := &mockRepository{}
	srv := NewServer(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/feedings/daily?days=14", nil)
	w := httptest.NewRecorder()
	srv.HandleDailyTotals(w, req)

	if mock.lastDays != 14 {
		t.Errorf("expected days=14, got %d", mock.lastDays)
	}
}

func TestHandleDailyTotals_GET_InvalidDaysParam(t *testing.T) {
	mock := &mockRepository{}
	srv := NewServer(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/feedings/daily?days=abc", nil)
	w := httptest.NewRecorder()
	srv.HandleDailyTotals(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if mock.lastDays != 7 {
		t.Errorf("expected default days=7 for invalid param, got %d", mock.lastDays)
	}
}

func TestHandleDailyTotals_GET_EmptyResult(t *testing.T) {
	mock := &mockRepository{}
	srv := NewServer(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/feedings/daily", nil)
	w := httptest.NewRecorder()
	srv.HandleDailyTotals(w, req)

	var result []DailyTotal
	json.Unmarshal(w.Body.Bytes(), &result)
	if len(result) != 0 {
		t.Errorf("expected empty array, got %d totals", len(result))
	}
}

func TestHandleDailyTotals_MethodNotAllowed(t *testing.T) {
	mock := &mockRepository{}
	srv := NewServer(mock)

	req := httptest.NewRequest(http.MethodPost, "/api/feedings/daily", nil)
	w := httptest.NewRecorder()
	srv.HandleDailyTotals(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestHandleDailyTotals_GET_RepoError(t *testing.T) {
	mock := &mockRepository{dailyErr: fmt.Errorf("query failed")}
	srv := NewServer(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/feedings/daily", nil)
	w := httptest.NewRecorder()
	srv.HandleDailyTotals(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// --- respondJSON / respondError tests ---

func TestRespondJSON(t *testing.T) {
	w := httptest.NewRecorder()
	respondJSON(w, http.StatusOK, map[string]string{"key": "value"})

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected application/json, got %s", ct)
	}

	var result map[string]string
	json.Unmarshal(w.Body.Bytes(), &result)
	if result["key"] != "value" {
		t.Errorf("expected key=value, got %v", result)
	}
}

func TestRespondError(t *testing.T) {
	w := httptest.NewRecorder()
	respondError(w, http.StatusBadRequest, "something went wrong")

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}

	var result map[string]string
	json.Unmarshal(w.Body.Bytes(), &result)
	if result["error"] != "something went wrong" {
		t.Errorf("expected error message, got %v", result)
	}
}
