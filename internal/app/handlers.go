package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Server holds dependencies for HTTP handlers.
type Server struct {
	Repo   Repository
	Gemini *GeminiClient
}

// NewServer creates a new Server with the given repository and optional Gemini client.
func NewServer(repo Repository, gemini ...*GeminiClient) *Server {
	s := &Server{Repo: repo}
	if len(gemini) > 0 {
		s.Gemini = gemini[0]
	}
	return s
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, msg string) {
	if status >= 500 {
		log.Printf("internal error: %s", msg)
		msg = "internal server error"
	}
	respondJSON(w, status, map[string]string{"error": msg})
}

// validateTimezone returns true if tz is empty or a valid IANA timezone.
func validateTimezone(tz string) bool {
	if tz == "" {
		return true
	}
	_, err := time.LoadLocation(tz)
	return err == nil
}

func (s *Server) HandleFeedings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.listFeedings(w, r)
	case http.MethodPost:
		s.createFeeding(w, r)
	default:
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) HandleFeedingByID(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/feedings/"), "/")
	id, err := strconv.Atoi(parts[0])
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	switch r.Method {
	case http.MethodPut:
		s.updateFeeding(w, r, id)
	case http.MethodDelete:
		s.deleteFeeding(w, r, id)
	default:
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) listFeedings(w http.ResponseWriter, r *http.Request) {
	filter := r.URL.Query().Get("month")
	if filter == "" {
		filter = r.URL.Query().Get("date")
	}
	tz := r.URL.Query().Get("tz")
	if !validateTimezone(tz) {
		respondError(w, http.StatusBadRequest, "invalid timezone")
		return
	}
	feedings, err := s.Repo.GetFeedings(filter, tz)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if feedings == nil {
		feedings = []Feeding{}
	}
	respondJSON(w, http.StatusOK, feedings)
}

func (s *Server) createFeeding(w http.ResponseWriter, r *http.Request) {
	var input FeedingInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if err := input.Validate(); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	feeding, err := s.Repo.CreateFeeding(input)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	_ = s.Repo.InvalidateInsightCache()
	respondJSON(w, http.StatusCreated, feeding)
}

func (s *Server) updateFeeding(w http.ResponseWriter, r *http.Request, id int) {
	var input FeedingInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if err := input.Validate(); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	feeding, err := s.Repo.UpdateFeeding(id, input)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			respondError(w, http.StatusNotFound, "feeding not found")
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	_ = s.Repo.InvalidateInsightCache()
	respondJSON(w, http.StatusOK, feeding)
}

func (s *Server) deleteFeeding(w http.ResponseWriter, r *http.Request, id int) {
	if err := s.Repo.DeleteFeeding(id); err != nil {
		if errors.Is(err, ErrNotFound) {
			respondError(w, http.StatusNotFound, "feeding not found")
		} else {
			respondError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	_ = s.Repo.InvalidateInsightCache()
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) HandleDailyTotals(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var totals []DailyTotal
	var err error

	tz := r.URL.Query().Get("tz")
	if !validateTimezone(tz) {
		respondError(w, http.StatusBadRequest, "invalid timezone")
		return
	}

	if month := r.URL.Query().Get("month"); month != "" {
		totals, err = s.Repo.GetDailyTotalsByMonth(month, tz)
	} else {
		days := 7
		if d := r.URL.Query().Get("days"); d != "" {
			if parsed, err := strconv.Atoi(d); err == nil && parsed > 0 {
				days = parsed
			}
		}
		totals, err = s.Repo.GetDailyTotals(days, tz)
	}

	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if totals == nil {
		totals = []DailyTotal{}
	}
	respondJSON(w, http.StatusOK, totals)
}

func (s *Server) HandleLastFeeding(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	feeding, err := s.Repo.GetLastFeeding()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if feeding == nil {
		respondJSON(w, http.StatusOK, nil)
		return
	}
	respondJSON(w, http.StatusOK, feeding)
}

// ── Sleep handlers ──

func (s *Server) HandleSleeps(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.listSleeps(w, r)
	case http.MethodPost:
		s.createSleep(w, r)
	default:
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) HandleSleepByID(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/sleeps/"), "/")
	id, err := strconv.Atoi(parts[0])
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	switch r.Method {
	case http.MethodPut:
		s.updateSleep(w, r, id)
	case http.MethodDelete:
		s.deleteSleep(w, r, id)
	default:
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) listSleeps(w http.ResponseWriter, r *http.Request) {
	filter := r.URL.Query().Get("month")
	if filter == "" {
		filter = r.URL.Query().Get("date")
	}
	tz := r.URL.Query().Get("tz")
	if !validateTimezone(tz) {
		respondError(w, http.StatusBadRequest, "invalid timezone")
		return
	}
	sleeps, err := s.Repo.GetSleeps(filter, tz)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if sleeps == nil {
		sleeps = []Sleep{}
	}
	respondJSON(w, http.StatusOK, sleeps)
}

func (s *Server) createSleep(w http.ResponseWriter, r *http.Request) {
	var input SleepInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if err := input.Validate(); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	sleep, err := s.Repo.CreateSleep(input)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	_ = s.Repo.InvalidateInsightCache()
	respondJSON(w, http.StatusCreated, sleep)
}

func (s *Server) updateSleep(w http.ResponseWriter, r *http.Request, id int) {
	var input SleepInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if err := input.Validate(); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	sleep, err := s.Repo.UpdateSleep(id, input)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			respondError(w, http.StatusNotFound, "sleep not found")
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	_ = s.Repo.InvalidateInsightCache()
	respondJSON(w, http.StatusOK, sleep)
}

func (s *Server) deleteSleep(w http.ResponseWriter, r *http.Request, id int) {
	if err := s.Repo.DeleteSleep(id); err != nil {
		if errors.Is(err, ErrNotFound) {
			respondError(w, http.StatusNotFound, "sleep not found")
		} else {
			respondError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	_ = s.Repo.InvalidateInsightCache()
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) HandleSleepDailyTotals(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var totals []DailySleepTotal
	var err error

	tz := r.URL.Query().Get("tz")
	if !validateTimezone(tz) {
		respondError(w, http.StatusBadRequest, "invalid timezone")
		return
	}

	if month := r.URL.Query().Get("month"); month != "" {
		totals, err = s.Repo.GetSleepDailyTotalsByMonth(month, tz)
	} else {
		days := 7
		if d := r.URL.Query().Get("days"); d != "" {
			if parsed, err := strconv.Atoi(d); err == nil && parsed > 0 {
				days = parsed
			}
		}
		totals, err = s.Repo.GetSleepDailyTotals(days, tz)
	}

	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if totals == nil {
		totals = []DailySleepTotal{}
	}
	respondJSON(w, http.StatusOK, totals)
}

func (s *Server) HandleLastSleep(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	sleep, err := s.Repo.GetLastSleep()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if sleep == nil {
		respondJSON(w, http.StatusOK, nil)
		return
	}
	respondJSON(w, http.StatusOK, sleep)
}

// ── Baby profile & development handlers ──

func (s *Server) HandleBabyProfile(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		profile, err := s.Repo.GetBabyProfile()
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if profile == nil {
			respondJSON(w, http.StatusOK, nil)
			return
		}
		respondJSON(w, http.StatusOK, profile)
	case http.MethodPut:
		var input BabyProfileInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			respondError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		if err := input.Validate(); err != nil {
			respondError(w, http.StatusBadRequest, err.Error())
			return
		}
		profile, err := s.Repo.SaveBabyProfile(input)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		// Invalidate insight cache when profile changes
		_ = s.Repo.InvalidateInsightCache()
		respondJSON(w, http.StatusOK, profile)
	default:
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) HandleDevelopment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Get baby profile
	profile, err := s.Repo.GetBabyProfile()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if profile == nil {
		respondError(w, http.StatusBadRequest, "please set baby date of birth first")
		return
	}

	// Calculate current week
	now := time.Now()
	ageInDays := int(now.Sub(profile.DateOfBirth).Hours() / 24)
	currentWeek := ageInDays / 7
	if currentWeek < 0 {
		currentWeek = 0
	}

	// Optional week query param to get a specific week
	if w_param := r.URL.Query().Get("week"); w_param != "" {
		if parsed, err := strconv.Atoi(w_param); err == nil && parsed >= 0 {
			// Return single week
			content, err := s.getOrGenerateWeek(parsed, profile.DateOfBirth)
			if err != nil {
				respondError(w, http.StatusInternalServerError, err.Error())
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, `{"current_week":%d,"weeks":[%s]}`, currentWeek, content)
			return
		}
	}

	// Load current week + 3 weeks ahead
	var weeks []string
	for i := 0; i < 4; i++ {
		wk := currentWeek + i
		content, err := s.getOrGenerateWeek(wk, profile.DateOfBirth)
		if err != nil {
			log.Printf("Failed to generate content for week %d: %v", wk, err)
			// Return error placeholder
			content = fmt.Sprintf(`{"week_number":%d,"error":"Failed to generate content: %s"}`, wk, strings.ReplaceAll(err.Error(), `"`, `\"`))
		}
		weeks = append(weeks, content)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"current_week":%d,"weeks":[%s]}`, currentWeek, strings.Join(weeks, ","))
}

// getOrGenerateWeek returns cached content or generates it via Gemini.
func (s *Server) getOrGenerateWeek(weekNumber int, dob time.Time) (string, error) {
	// Check cache first
	cached, err := s.Repo.GetDevelopmentCache(weekNumber)
	if err != nil {
		return "", err
	}
	if cached != nil {
		return cached.Content, nil
	}

	// Generate via Gemini
	if s.Gemini == nil {
		return "", fmt.Errorf("AI service is not configured (missing GEMINI_API_KEY)")
	}

	content, err := s.Gemini.GenerateDevelopmentContent(weekNumber, dob)
	if err != nil {
		return "", err
	}

	// Cache for future use
	if saveErr := s.Repo.SaveDevelopmentCache(weekNumber, content); saveErr != nil {
		log.Printf("Warning: failed to cache week %d content: %v", weekNumber, saveErr)
	}

	return content, nil
}

// ── Growth measurement handlers ──

func (s *Server) HandleGrowthMeasurements(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.listGrowthMeasurements(w, r)
	case http.MethodPost:
		s.createGrowthMeasurement(w, r)
	default:
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) HandleGrowthMeasurementByID(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/growth/"), "/")
	id, err := strconv.Atoi(parts[0])
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	switch r.Method {
	case http.MethodPut:
		s.updateGrowthMeasurement(w, r, id)
	case http.MethodDelete:
		s.deleteGrowthMeasurement(w, r, id)
	default:
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) listGrowthMeasurements(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	measurements, err := s.Repo.GetGrowthMeasurements(limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if measurements == nil {
		measurements = []GrowthMeasurement{}
	}
	respondJSON(w, http.StatusOK, measurements)
}

func (s *Server) createGrowthMeasurement(w http.ResponseWriter, r *http.Request) {
	var input GrowthMeasurementInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if err := input.Validate(); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	measurement, err := s.Repo.CreateGrowthMeasurement(input)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	_ = s.Repo.InvalidateInsightCache()
	respondJSON(w, http.StatusCreated, measurement)
}

func (s *Server) updateGrowthMeasurement(w http.ResponseWriter, r *http.Request, id int) {
	var input GrowthMeasurementInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if err := input.Validate(); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	measurement, err := s.Repo.UpdateGrowthMeasurement(id, input)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			respondError(w, http.StatusNotFound, "growth measurement not found")
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	_ = s.Repo.InvalidateInsightCache()
	respondJSON(w, http.StatusOK, measurement)
}

func (s *Server) deleteGrowthMeasurement(w http.ResponseWriter, r *http.Request, id int) {
	if err := s.Repo.DeleteGrowthMeasurement(id); err != nil {
		if errors.Is(err, ErrNotFound) {
			respondError(w, http.StatusNotFound, "growth measurement not found")
		} else {
			respondError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	_ = s.Repo.InvalidateInsightCache()
	w.WriteHeader(http.StatusNoContent)
}

// ── Insights handler ──

func (s *Server) HandleInsights(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Get baby profile
	profile, err := s.Repo.GetBabyProfile()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if profile == nil {
		respondError(w, http.StatusBadRequest, "please set baby date of birth first")
		return
	}

	// Calculate cache key based on current week
	now := time.Now()
	ageInDays := int(now.Sub(profile.DateOfBirth).Hours() / 24)
	currentWeek := ageInDays / 7

	cacheKey := fmt.Sprintf("insight-week-%d", currentWeek)

	// Check cache
	cached, err := s.Repo.GetInsightCache(cacheKey)
	if err != nil {
		log.Printf("Warning: failed to check insight cache: %v", err)
	}
	if cached != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, cached.Content)
		return
	}

	// Gather data for insight generation
	latestGrowth, _ := s.Repo.GetLatestGrowthMeasurement()
	feedingAvg, _ := s.Repo.GetFeedingDailyAvg(7)
	sleepAvg, _ := s.Repo.GetSleepDailyAvg(7)

	// Generate via Gemini
	if s.Gemini == nil {
		respondError(w, http.StatusServiceUnavailable, "AI service is not configured (missing GEMINI_API_KEY)")
		return
	}

	content, err := s.Gemini.GeneratePersonalizedInsight(*profile, latestGrowth, feedingAvg, sleepAvg)
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to generate insights: %v", err))
		return
	}

	// Cache for 24 hours
	expiresAt := now.Add(24 * time.Hour)
	if saveErr := s.Repo.SaveInsightCache(cacheKey, content, expiresAt); saveErr != nil {
		log.Printf("Warning: failed to cache insight: %v", saveErr)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, content)
}

// ── WHO data handler ──

func (s *Server) HandleWHOData(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	gender := r.URL.Query().Get("gender")
	if gender == "" {
		gender = "male"
	}
	metric := r.URL.Query().Get("metric")
	if metric == "" {
		metric = "weight"
	}
	months := 24
	if m := r.URL.Query().Get("months"); m != "" {
		if parsed, err := strconv.Atoi(m); err == nil && parsed > 0 {
			months = parsed
		}
	}

	data := GetWHOCurveData(gender, metric, months)
	if data == nil {
		respondError(w, http.StatusBadRequest, "invalid metric, use 'weight' or 'length'")
		return
	}

	// Also include baby's actual measurements if available
	type WHOResponse struct {
		*WHOPercentileData
		BabyData []struct {
			Month  int     `json:"month"`
			Value  float64 `json:"value"`
		} `json:"baby_data"`
	}

	resp := WHOResponse{WHOPercentileData: data}

	// Get growth measurements to overlay
	measurements, err := s.Repo.GetGrowthMeasurements(100)
	if err == nil && len(measurements) > 0 {
		profile, _ := s.Repo.GetBabyProfile()
		if profile != nil {
			for _, m := range measurements {
				ageInDays := int(m.Date.Sub(profile.DateOfBirth).Hours() / 24)
				ageMonth := ageInDays / 30
				var value float64
				if metric == "weight" {
					value = m.WeightKg
				} else {
					value = m.LengthCm
				}
				resp.BabyData = append(resp.BabyData, struct {
					Month  int     `json:"month"`
					Value  float64 `json:"value"`
				}{Month: ageMonth, Value: value})
			}
		}
	}

	respondJSON(w, http.StatusOK, resp)
}

// ── Diaper handlers ──

func (s *Server) HandleDiapers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.listDiapers(w, r)
	case http.MethodPost:
		s.createDiaper(w, r)
	default:
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) HandleDiaperByID(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/diapers/"), "/")
	id, err := strconv.Atoi(parts[0])
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	switch r.Method {
	case http.MethodPut:
		s.updateDiaper(w, r, id)
	case http.MethodDelete:
		s.deleteDiaper(w, r, id)
	default:
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) listDiapers(w http.ResponseWriter, r *http.Request) {
	filter := r.URL.Query().Get("month")
	if filter == "" {
		filter = r.URL.Query().Get("date")
	}
	tz := r.URL.Query().Get("tz")
	if !validateTimezone(tz) {
		respondError(w, http.StatusBadRequest, "invalid timezone")
		return
	}
	diapers, err := s.Repo.GetDiapers(filter, tz)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if diapers == nil {
		diapers = []Diaper{}
	}
	respondJSON(w, http.StatusOK, diapers)
}

func (s *Server) createDiaper(w http.ResponseWriter, r *http.Request) {
	var input DiaperInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if err := input.Validate(); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	diaper, err := s.Repo.CreateDiaper(input)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusCreated, diaper)
}

func (s *Server) updateDiaper(w http.ResponseWriter, r *http.Request, id int) {
	var input DiaperInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if err := input.Validate(); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	diaper, err := s.Repo.UpdateDiaper(id, input)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			respondError(w, http.StatusNotFound, "diaper not found")
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, diaper)
}

func (s *Server) deleteDiaper(w http.ResponseWriter, r *http.Request, id int) {
	if err := s.Repo.DeleteDiaper(id); err != nil {
		if errors.Is(err, ErrNotFound) {
			respondError(w, http.StatusNotFound, "diaper not found")
		} else {
			respondError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── Bath handlers ──

func (s *Server) HandleBaths(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.listBaths(w, r)
	case http.MethodPost:
		s.createBath(w, r)
	default:
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) HandleBathByID(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/baths/"), "/")
	id, err := strconv.Atoi(parts[0])
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	switch r.Method {
	case http.MethodPut:
		s.updateBath(w, r, id)
	case http.MethodDelete:
		s.deleteBath(w, r, id)
	default:
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) listBaths(w http.ResponseWriter, r *http.Request) {
	filter := r.URL.Query().Get("month")
	if filter == "" {
		filter = r.URL.Query().Get("date")
	}
	tz := r.URL.Query().Get("tz")
	if !validateTimezone(tz) {
		respondError(w, http.StatusBadRequest, "invalid timezone")
		return
	}
	baths, err := s.Repo.GetBaths(filter, tz)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if baths == nil {
		baths = []Bath{}
	}
	respondJSON(w, http.StatusOK, baths)
}

func (s *Server) createBath(w http.ResponseWriter, r *http.Request) {
	var input BathInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if err := input.Validate(); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	bath, err := s.Repo.CreateBath(input)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusCreated, bath)
}

func (s *Server) updateBath(w http.ResponseWriter, r *http.Request, id int) {
	var input BathInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if err := input.Validate(); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	bath, err := s.Repo.UpdateBath(id, input)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			respondError(w, http.StatusNotFound, "bath not found")
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, bath)
}

func (s *Server) deleteBath(w http.ResponseWriter, r *http.Request, id int) {
	if err := s.Repo.DeleteBath(id); err != nil {
		if errors.Is(err, ErrNotFound) {
			respondError(w, http.StatusNotFound, "bath not found")
		} else {
			respondError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
