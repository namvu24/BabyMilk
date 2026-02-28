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
	srv := app.NewServer(repo)

	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("/api/feedings/daily", srv.HandleDailyTotals)
	mux.HandleFunc("/api/feedings/", srv.HandleFeedingByID)
	mux.HandleFunc("/api/feedings", srv.HandleFeedings)

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
