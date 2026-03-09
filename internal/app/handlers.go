package app

import (
	"encoding/json"
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
	respondJSON(w, status, map[string]string{"error": msg})
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
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, feeding)
}

func (s *Server) deleteFeeding(w http.ResponseWriter, r *http.Request, id int) {
	if err := s.Repo.DeleteFeeding(id); err != nil {
		if err.Error() == "feeding not found" {
			respondError(w, http.StatusNotFound, err.Error())
		} else {
			respondError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
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
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, sleep)
}

func (s *Server) deleteSleep(w http.ResponseWriter, r *http.Request, id int) {
	if err := s.Repo.DeleteSleep(id); err != nil {
		if err.Error() == "sleep not found" {
			respondError(w, http.StatusNotFound, err.Error())
		} else {
			respondError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
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
		profile, err := s.Repo.SaveBabyProfile(input.DateOfBirth)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
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
