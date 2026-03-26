package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"babymilk/internal/app"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	db, err := app.InitDB()
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}
	app.RunMigrations(db)

	repo := app.NewPostgresRepository(db)

	// Initialize Gemini client (optional — development tab degrades gracefully without it)
	var gemini *app.GeminiClient
	geminiKey := os.Getenv("GEMINI_API_KEY")
	if geminiKey != "" {
		geminiModel := os.Getenv("GEMINI_MODEL")
		gemini = app.NewGeminiClient(geminiKey, geminiModel)
		log.Printf("Gemini AI configured (model: %s)", gemini.Model)
	} else {
		log.Println("Warning: GEMINI_API_KEY not set — Development tab AI features disabled")
	}

	// Start DB connection pool metrics collector (every 5s)
	app.StartDBStatsCollector(db, 5*time.Second)

	srv := app.NewServer(repo, gemini)

	mux := http.NewServeMux()

	// Observability endpoints
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	mux.HandleFunc("/readyz", readyzHandler(db))

	// API routes — Feedings
	mux.HandleFunc("/api/feedings/daily", srv.HandleDailyTotals)
	mux.HandleFunc("/api/feedings/last", srv.HandleLastFeeding)
	mux.HandleFunc("/api/feedings/", srv.HandleFeedingByID)
	mux.HandleFunc("/api/feedings", srv.HandleFeedings)

	// API routes — Sleeps
	mux.HandleFunc("/api/sleeps/active", srv.HandleSleepActive)
	mux.HandleFunc("/api/sleeps/start", srv.HandleSleepStart)
	mux.HandleFunc("/api/sleeps/daily", srv.HandleSleepDailyTotals)
	mux.HandleFunc("/api/sleeps/last", srv.HandleLastSleep)
	mux.HandleFunc("/api/sleeps/", srv.HandleSleepByID)
	mux.HandleFunc("/api/sleeps", srv.HandleSleeps)

	// API routes — Diapers
	mux.HandleFunc("/api/diapers/", srv.HandleDiaperByID)
	mux.HandleFunc("/api/diapers", srv.HandleDiapers)

	// API routes — Baths
	mux.HandleFunc("/api/baths/", srv.HandleBathByID)
	mux.HandleFunc("/api/baths", srv.HandleBaths)

	// API routes — Baby profile & Development
	mux.HandleFunc("/api/baby", srv.HandleBabyProfile)
	mux.HandleFunc("/api/development", srv.HandleDevelopment)

	// API routes — Growth measurements
	mux.HandleFunc("/api/growth/", srv.HandleGrowthMeasurementByID)
	mux.HandleFunc("/api/growth", srv.HandleGrowthMeasurements)

	// API routes — Insights & WHO data
	mux.HandleFunc("/api/insights", srv.HandleInsights)
	mux.HandleFunc("/api/who-data", srv.HandleWHOData)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	handler := app.PrometheusMiddleware(corsMiddleware(maxBodyMiddleware(mux)))

	log.Printf("BabyMilk server starting on http://localhost:%s", port)
	log.Fatal(http.ListenAndServe(":"+port, handler))
}

// maxBodyMiddleware limits request body size for POST/PUT requests to prevent
// memory exhaustion from oversized payloads.
func maxBodyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
			r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MB
		}
		next.ServeHTTP(w, r)
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// readyzHandler returns 200 if the DB is reachable, 503 otherwise.
func readyzHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := db.Ping(); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("db not ready"))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}
}
