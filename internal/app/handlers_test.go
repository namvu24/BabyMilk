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

	// Sleep-specific mock state
	sleeps           []Sleep
	sleepDailyTotals []DailySleepTotal
	sleepGetErr      error
	sleepCreateErr   error
	sleepUpdateErr   error
	sleepDeleteErr   error
	sleepDailyErr    error
	lastSleepInput   SleepInput

	// Baby profile & development mock state
	babyProfile      *BabyProfile
	babyProfileErr   error
	devCache         map[int]*DevelopmentContent
	devCacheErr      error
	lastDOB          string

	// Growth measurement mock state
	growthMeasurements []GrowthMeasurement
	growthGetErr       error
	growthCreateErr    error
	growthUpdateErr    error
	growthDeleteErr    error
	lastGrowthInput    GrowthMeasurementInput

	// Insight cache mock state
	insightCache    map[string]*InsightCache
	feedingDailyAvg int
	sleepDailyAvg   int
}

func (m *mockRepository) GetFeedings(date string, tz string) ([]Feeding, error) {
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

func (m *mockRepository) GetDailyTotals(days int, tz string) ([]DailyTotal, error) {
	m.lastDays = days
	return m.dailyTotals, m.dailyErr
}

func (m *mockRepository) GetDailyTotalsByMonth(month string, tz string) ([]DailyTotal, error) {
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

// ── Sleep mock methods ──

func (m *mockRepository) GetSleeps(date string, tz string) ([]Sleep, error) {
	m.lastDate = date
	return m.sleeps, m.sleepGetErr
}

func (m *mockRepository) GetLastSleep() (*Sleep, error) {
	if m.sleepGetErr != nil {
		return nil, m.sleepGetErr
	}
	if len(m.sleeps) == 0 {
		return nil, nil
	}
	return &m.sleeps[0], nil
}

func (m *mockRepository) CreateSleep(input SleepInput) (Sleep, error) {
	m.lastSleepInput = input
	if m.sleepCreateErr != nil {
		return Sleep{}, m.sleepCreateErr
	}
	s := Sleep{
		ID:        1,
		StartTime: time.Now(),
		EndTime:   time.Now().Add(time.Hour),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	return s, nil
}

func (m *mockRepository) UpdateSleep(id int, input SleepInput) (Sleep, error) {
	m.lastID = id
	m.lastSleepInput = input
	if m.sleepUpdateErr != nil {
		return Sleep{}, m.sleepUpdateErr
	}
	s := Sleep{
		ID:        id,
		StartTime: time.Now(),
		EndTime:   time.Now().Add(time.Hour),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	return s, nil
}

func (m *mockRepository) DeleteSleep(id int) error {
	m.lastID = id
	return m.sleepDeleteErr
}

func (m *mockRepository) GetSleepDailyTotals(days int, tz string) ([]DailySleepTotal, error) {
	m.lastDays = days
	return m.sleepDailyTotals, m.sleepDailyErr
}

func (m *mockRepository) GetSleepDailyTotalsByMonth(month string, tz string) ([]DailySleepTotal, error) {
	m.lastDate = month
	return m.sleepDailyTotals, m.sleepDailyErr
}

// ── Baby profile & development mock methods ──

func (m *mockRepository) GetBabyProfile() (*BabyProfile, error) {
	return m.babyProfile, m.babyProfileErr
}

func (m *mockRepository) SaveBabyProfile(input BabyProfileInput) (*BabyProfile, error) {
	m.lastDOB = input.DateOfBirth
	if m.babyProfileErr != nil {
		return nil, m.babyProfileErr
	}
	parsed, _ := time.Parse("2006-01-02", input.DateOfBirth)
	m.babyProfile = &BabyProfile{
		ID:          1,
		DateOfBirth: parsed,
		Name:        input.Name,
		Gender:      input.Gender,
		MilkType:    input.MilkType,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	return m.babyProfile, nil
}

func (m *mockRepository) GetDevelopmentCache(weekNumber int) (*DevelopmentContent, error) {
	if m.devCacheErr != nil {
		return nil, m.devCacheErr
	}
	if m.devCache != nil {
		if c, ok := m.devCache[weekNumber]; ok {
			return c, nil
		}
	}
	return nil, nil
}

func (m *mockRepository) SaveDevelopmentCache(weekNumber int, content string) error {
	if m.devCache == nil {
		m.devCache = make(map[int]*DevelopmentContent)
	}
	m.devCache[weekNumber] = &DevelopmentContent{
		ID:         weekNumber,
		WeekNumber: weekNumber,
		Content:    content,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	return m.devCacheErr
}

// ── Growth measurement mock methods ──

func (m *mockRepository) GetGrowthMeasurements(limit int) ([]GrowthMeasurement, error) {
	return m.growthMeasurements, m.growthGetErr
}

func (m *mockRepository) GetGrowthMeasurementsByRange(from, to time.Time) ([]GrowthMeasurement, error) {
	return m.growthMeasurements, m.growthGetErr
}

func (m *mockRepository) GetLatestGrowthMeasurement() (*GrowthMeasurement, error) {
	if m.growthGetErr != nil {
		return nil, m.growthGetErr
	}
	if len(m.growthMeasurements) == 0 {
		return nil, nil
	}
	return &m.growthMeasurements[0], nil
}

func (m *mockRepository) CreateGrowthMeasurement(input GrowthMeasurementInput) (GrowthMeasurement, error) {
	m.lastGrowthInput = input
	if m.growthCreateErr != nil {
		return GrowthMeasurement{}, m.growthCreateErr
	}
	return GrowthMeasurement{
		ID:       1,
		WeightKg: input.WeightKg,
		LengthCm: input.LengthCm,
		Date:     time.Now(),
	}, nil
}

func (m *mockRepository) UpdateGrowthMeasurement(id int, input GrowthMeasurementInput) (GrowthMeasurement, error) {
	m.lastID = id
	m.lastGrowthInput = input
	if m.growthUpdateErr != nil {
		return GrowthMeasurement{}, m.growthUpdateErr
	}
	return GrowthMeasurement{
		ID:       id,
		WeightKg: input.WeightKg,
		LengthCm: input.LengthCm,
		Date:     time.Now(),
	}, nil
}

func (m *mockRepository) DeleteGrowthMeasurement(id int) error {
	m.lastID = id
	return m.growthDeleteErr
}

// ── Insight cache mock methods ──

func (m *mockRepository) GetInsightCache(key string) (*InsightCache, error) {
	if m.insightCache != nil {
		if c, ok := m.insightCache[key]; ok {
			return c, nil
		}
	}
	return nil, nil
}

func (m *mockRepository) SaveInsightCache(key, content string, expiresAt time.Time) error {
	if m.insightCache == nil {
		m.insightCache = make(map[string]*InsightCache)
	}
	m.insightCache[key] = &InsightCache{CacheKey: key, Content: content, ExpiresAt: expiresAt}
	return nil
}

func (m *mockRepository) InvalidateInsightCache() error {
	m.insightCache = nil
	return nil
}

func (m *mockRepository) GetFeedingDailyAvg(days int) (int, error) {
	return m.feedingDailyAvg, nil
}

func (m *mockRepository) GetSleepDailyAvg(days int) (int, error) {
	return m.sleepDailyAvg, nil
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

// ═══════════════════════════════════════════════════════════════════════════
// ── Sleep handler tests ──
// ═══════════════════════════════════════════════════════════════════════════

func validSleepJSON() string {
	return `{"start_time":"2025-01-15T22:00:00Z","end_time":"2025-01-16T06:00:00Z"}`
}

// --- HandleSleeps tests ---

func TestHandleSleeps_GET_Success(t *testing.T) {
	mock := &mockRepository{
		sleeps: []Sleep{
			{ID: 1, StartTime: time.Now(), EndTime: time.Now().Add(time.Hour)},
			{ID: 2, StartTime: time.Now(), EndTime: time.Now().Add(2 * time.Hour)},
		},
	}
	srv := NewServer(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/sleeps", nil)
	w := httptest.NewRecorder()
	srv.HandleSleeps(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var result []Sleep
	json.Unmarshal(w.Body.Bytes(), &result)
	if len(result) != 2 {
		t.Errorf("expected 2 sleeps, got %d", len(result))
	}
}

func TestHandleSleeps_GET_EmptyList(t *testing.T) {
	mock := &mockRepository{}
	srv := NewServer(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/sleeps", nil)
	w := httptest.NewRecorder()
	srv.HandleSleeps(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var result []Sleep
	json.Unmarshal(w.Body.Bytes(), &result)
	if len(result) != 0 {
		t.Errorf("expected empty array, got %d sleeps", len(result))
	}
}

func TestHandleSleeps_GET_RepoError(t *testing.T) {
	mock := &mockRepository{sleepGetErr: fmt.Errorf("db connection lost")}
	srv := NewServer(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/sleeps", nil)
	w := httptest.NewRecorder()
	srv.HandleSleeps(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestHandleSleeps_POST_Success(t *testing.T) {
	mock := &mockRepository{}
	srv := NewServer(mock)

	body := strings.NewReader(validSleepJSON())
	req := httptest.NewRequest(http.MethodPost, "/api/sleeps", body)
	w := httptest.NewRecorder()
	srv.HandleSleeps(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", w.Code)
	}
}

func TestHandleSleeps_POST_InvalidJSON(t *testing.T) {
	mock := &mockRepository{}
	srv := NewServer(mock)

	body := strings.NewReader("{invalid")
	req := httptest.NewRequest(http.MethodPost, "/api/sleeps", body)
	w := httptest.NewRecorder()
	srv.HandleSleeps(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleSleeps_POST_ValidationError(t *testing.T) {
	mock := &mockRepository{}
	srv := NewServer(mock)

	body := strings.NewReader(`{"start_time":"2025-01-16T06:00:00Z","end_time":"2025-01-15T22:00:00Z"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/sleeps", body)
	w := httptest.NewRecorder()
	srv.HandleSleeps(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleSleeps_POST_RepoError(t *testing.T) {
	mock := &mockRepository{sleepCreateErr: fmt.Errorf("insert failed")}
	srv := NewServer(mock)

	body := strings.NewReader(validSleepJSON())
	req := httptest.NewRequest(http.MethodPost, "/api/sleeps", body)
	w := httptest.NewRecorder()
	srv.HandleSleeps(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestHandleSleeps_MethodNotAllowed(t *testing.T) {
	mock := &mockRepository{}
	srv := NewServer(mock)

	req := httptest.NewRequest(http.MethodDelete, "/api/sleeps", nil)
	w := httptest.NewRecorder()
	srv.HandleSleeps(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

// --- HandleSleepByID tests ---

func TestHandleSleepByID_PUT_Success(t *testing.T) {
	mock := &mockRepository{}
	srv := NewServer(mock)

	body := strings.NewReader(validSleepJSON())
	req := httptest.NewRequest(http.MethodPut, "/api/sleeps/42", body)
	w := httptest.NewRecorder()
	srv.HandleSleepByID(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if mock.lastID != 42 {
		t.Errorf("expected id=42, got %d", mock.lastID)
	}
}

func TestHandleSleepByID_PUT_InvalidJSON(t *testing.T) {
	mock := &mockRepository{}
	srv := NewServer(mock)

	body := strings.NewReader("{bad")
	req := httptest.NewRequest(http.MethodPut, "/api/sleeps/1", body)
	w := httptest.NewRecorder()
	srv.HandleSleepByID(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleSleepByID_PUT_ValidationError(t *testing.T) {
	mock := &mockRepository{}
	srv := NewServer(mock)

	body := strings.NewReader(`{"start_time":"2025-01-16T06:00:00Z","end_time":"2025-01-15T22:00:00Z"}`)
	req := httptest.NewRequest(http.MethodPut, "/api/sleeps/1", body)
	w := httptest.NewRecorder()
	srv.HandleSleepByID(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleSleepByID_DELETE_Success(t *testing.T) {
	mock := &mockRepository{}
	srv := NewServer(mock)

	req := httptest.NewRequest(http.MethodDelete, "/api/sleeps/5", nil)
	w := httptest.NewRecorder()
	srv.HandleSleepByID(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
	if mock.lastID != 5 {
		t.Errorf("expected id=5, got %d", mock.lastID)
	}
}

func TestHandleSleepByID_DELETE_NotFound(t *testing.T) {
	mock := &mockRepository{sleepDeleteErr: fmt.Errorf("sleep not found")}
	srv := NewServer(mock)

	req := httptest.NewRequest(http.MethodDelete, "/api/sleeps/999", nil)
	w := httptest.NewRecorder()
	srv.HandleSleepByID(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestHandleSleepByID_InvalidID(t *testing.T) {
	mock := &mockRepository{}
	srv := NewServer(mock)

	req := httptest.NewRequest(http.MethodPut, "/api/sleeps/abc", nil)
	w := httptest.NewRecorder()
	srv.HandleSleepByID(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleSleepByID_MethodNotAllowed(t *testing.T) {
	mock := &mockRepository{}
	srv := NewServer(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/sleeps/1", nil)
	w := httptest.NewRecorder()
	srv.HandleSleepByID(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

// --- HandleSleepDailyTotals tests ---

func TestHandleSleepDailyTotals_GET_Success(t *testing.T) {
	mock := &mockRepository{
		sleepDailyTotals: []DailySleepTotal{
			{Date: "2025-01-15", TotalMinutes: 480},
			{Date: "2025-01-14", TotalMinutes: 420},
		},
	}
	srv := NewServer(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/sleeps/daily", nil)
	w := httptest.NewRecorder()
	srv.HandleSleepDailyTotals(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if mock.lastDays != 7 {
		t.Errorf("expected default days=7, got %d", mock.lastDays)
	}
}

func TestHandleSleepDailyTotals_GET_WithDaysParam(t *testing.T) {
	mock := &mockRepository{}
	srv := NewServer(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/sleeps/daily?days=14", nil)
	w := httptest.NewRecorder()
	srv.HandleSleepDailyTotals(w, req)

	if mock.lastDays != 14 {
		t.Errorf("expected days=14, got %d", mock.lastDays)
	}
}

func TestHandleSleepDailyTotals_GET_EmptyResult(t *testing.T) {
	mock := &mockRepository{}
	srv := NewServer(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/sleeps/daily", nil)
	w := httptest.NewRecorder()
	srv.HandleSleepDailyTotals(w, req)

	var result []DailySleepTotal
	json.Unmarshal(w.Body.Bytes(), &result)
	if len(result) != 0 {
		t.Errorf("expected empty array, got %d totals", len(result))
	}
}

func TestHandleSleepDailyTotals_GET_RepoError(t *testing.T) {
	mock := &mockRepository{sleepDailyErr: fmt.Errorf("query failed")}
	srv := NewServer(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/sleeps/daily", nil)
	w := httptest.NewRecorder()
	srv.HandleSleepDailyTotals(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestHandleSleepDailyTotals_MethodNotAllowed(t *testing.T) {
	mock := &mockRepository{}
	srv := NewServer(mock)

	req := httptest.NewRequest(http.MethodPost, "/api/sleeps/daily", nil)
	w := httptest.NewRecorder()
	srv.HandleSleepDailyTotals(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

// --- HandleLastSleep tests ---

func TestHandleLastSleep_GET_Success(t *testing.T) {
	mock := &mockRepository{
		sleeps: []Sleep{
			{ID: 1, StartTime: time.Now(), EndTime: time.Now().Add(time.Hour)},
		},
	}
	srv := NewServer(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/sleeps/last", nil)
	w := httptest.NewRecorder()
	srv.HandleLastSleep(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestHandleLastSleep_GET_NoSleeps(t *testing.T) {
	mock := &mockRepository{}
	srv := NewServer(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/sleeps/last", nil)
	w := httptest.NewRecorder()
	srv.HandleLastSleep(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if strings.TrimSpace(w.Body.String()) != "null" {
		t.Errorf("expected null body, got %s", w.Body.String())
	}
}

func TestHandleLastSleep_MethodNotAllowed(t *testing.T) {
	mock := &mockRepository{}
	srv := NewServer(mock)

	req := httptest.NewRequest(http.MethodPost, "/api/sleeps/last", nil)
	w := httptest.NewRecorder()
	srv.HandleLastSleep(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

// --- HandleGrowthMeasurements tests ---

func TestHandleGrowthMeasurements_GET_Success(t *testing.T) {
	mock := &mockRepository{
		growthMeasurements: []GrowthMeasurement{
			{ID: 1, WeightKg: 5.5, LengthCm: 55, Date: time.Now()},
		},
	}
	srv := NewServer(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/growth", nil)
	w := httptest.NewRecorder()
	srv.HandleGrowthMeasurements(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var result []GrowthMeasurement
	json.Unmarshal(w.Body.Bytes(), &result)
	if len(result) != 1 {
		t.Errorf("expected 1 measurement, got %d", len(result))
	}
}

func TestHandleGrowthMeasurements_POST_Success(t *testing.T) {
	mock := &mockRepository{}
	srv := NewServer(mock)

	body := strings.NewReader(`{"date":"2025-06-15","weight_kg":5.5,"length_cm":55.0}`)
	req := httptest.NewRequest(http.MethodPost, "/api/growth", body)
	w := httptest.NewRecorder()
	srv.HandleGrowthMeasurements(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", w.Code)
	}
	if mock.lastGrowthInput.WeightKg != 5.5 {
		t.Errorf("expected weight 5.5, got %f", mock.lastGrowthInput.WeightKg)
	}
}

func TestHandleGrowthMeasurements_POST_InvalidJSON(t *testing.T) {
	mock := &mockRepository{}
	srv := NewServer(mock)

	body := strings.NewReader(`{bad json}`)
	req := httptest.NewRequest(http.MethodPost, "/api/growth", body)
	w := httptest.NewRecorder()
	srv.HandleGrowthMeasurements(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleGrowthMeasurements_POST_ValidationError(t *testing.T) {
	mock := &mockRepository{}
	srv := NewServer(mock)

	body := strings.NewReader(`{"date":"2025-06-15","weight_kg":0.1,"length_cm":55}`)
	req := httptest.NewRequest(http.MethodPost, "/api/growth", body)
	w := httptest.NewRecorder()
	srv.HandleGrowthMeasurements(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleGrowthMeasurements_MethodNotAllowed(t *testing.T) {
	mock := &mockRepository{}
	srv := NewServer(mock)

	req := httptest.NewRequest(http.MethodDelete, "/api/growth", nil)
	w := httptest.NewRecorder()
	srv.HandleGrowthMeasurements(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

// --- HandleGrowthMeasurementByID tests ---

func TestHandleGrowthMeasurementByID_DELETE_Success(t *testing.T) {
	mock := &mockRepository{}
	srv := NewServer(mock)

	req := httptest.NewRequest(http.MethodDelete, "/api/growth/1", nil)
	w := httptest.NewRecorder()
	srv.HandleGrowthMeasurementByID(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
	if mock.lastID != 1 {
		t.Errorf("expected deleted ID 1, got %d", mock.lastID)
	}
}

func TestHandleGrowthMeasurementByID_DELETE_InvalidID(t *testing.T) {
	mock := &mockRepository{}
	srv := NewServer(mock)

	req := httptest.NewRequest(http.MethodDelete, "/api/growth/abc", nil)
	w := httptest.NewRecorder()
	srv.HandleGrowthMeasurementByID(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// --- HandleWHOData tests ---

func TestHandleWHOData_GET_Success(t *testing.T) {
	mock := &mockRepository{
		babyProfile: &BabyProfile{
			ID:          1,
			DateOfBirth: time.Now().AddDate(0, -3, 0),
			Gender:      "male",
		},
	}
	srv := NewServer(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/who-data?metric=weight", nil)
	w := httptest.NewRecorder()
	srv.HandleWHOData(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestHandleWHOData_MethodNotAllowed(t *testing.T) {
	mock := &mockRepository{}
	srv := NewServer(mock)

	req := httptest.NewRequest(http.MethodPost, "/api/who-data", nil)
	w := httptest.NewRecorder()
	srv.HandleWHOData(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

// --- HandleBabyProfile tests with extended fields ---

func TestHandleBabyProfile_PUT_WithGenderAndMilkType(t *testing.T) {
	mock := &mockRepository{}
	srv := NewServer(mock)

	body := strings.NewReader(`{"date_of_birth":"2025-01-15","name":"Mia","gender":"female","milk_type":"breast"}`)
	req := httptest.NewRequest(http.MethodPut, "/api/baby", body)
	w := httptest.NewRecorder()
	srv.HandleBabyProfile(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if mock.babyProfile.Gender != "female" {
		t.Errorf("expected gender female, got %s", mock.babyProfile.Gender)
	}
	if mock.babyProfile.Name != "Mia" {
		t.Errorf("expected name Mia, got %s", mock.babyProfile.Name)
	}
}

func TestHandleBabyProfile_PUT_InvalidGender(t *testing.T) {
	mock := &mockRepository{}
	srv := NewServer(mock)

	body := strings.NewReader(`{"date_of_birth":"2025-01-15","gender":"other"}`)
	req := httptest.NewRequest(http.MethodPut, "/api/baby", body)
	w := httptest.NewRecorder()
	srv.HandleBabyProfile(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}
