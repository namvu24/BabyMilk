package app

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

// Server holds dependencies for HTTP handlers.
type Server struct {
	Repo Repository
}

// NewServer creates a new Server with the given repository.
func NewServer(repo Repository) *Server {
	return &Server{Repo: repo}
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
