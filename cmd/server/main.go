package main

import (
	"log"
	"net/http"
	"os"
	"strings"

	"babymilk/internal/app"
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

	srv := app.NewServer(repo, gemini)

	mux := http.NewServeMux()

	// API routes — Feedings
	mux.HandleFunc("/api/feedings/daily", srv.HandleDailyTotals)
	mux.HandleFunc("/api/feedings/last", srv.HandleLastFeeding)
	mux.HandleFunc("/api/feedings/", srv.HandleFeedingByID)
	mux.HandleFunc("/api/feedings", srv.HandleFeedings)

	// API routes — Sleeps
	mux.HandleFunc("/api/sleeps/daily", srv.HandleSleepDailyTotals)
	mux.HandleFunc("/api/sleeps/last", srv.HandleLastSleep)
	mux.HandleFunc("/api/sleeps/", srv.HandleSleepByID)
	mux.HandleFunc("/api/sleeps", srv.HandleSleeps)

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

	handler := corsMiddleware(mux)

	log.Printf("BabyMilk server starting on http://localhost:%s", port)
	log.Fatal(http.ListenAndServe(":"+port, handler))
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
