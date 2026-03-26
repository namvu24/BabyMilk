package app

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Version is overridden at build time via -ldflags.
var Version = "dev"

// ── HTTP metrics ────────────────────────────────────────────────────────────

var (
	httpRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total number of HTTP requests.",
	}, []string{"method", "path", "status_code"})

	httpRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "Duration of HTTP requests in seconds.",
		Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
	}, []string{"method", "path"})

	httpRequestsInFlight = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "http_requests_in_flight",
		Help: "Number of HTTP requests currently being processed.",
	})
)

// ── Database metrics ────────────────────────────────────────────────────────

var (
	dbQueryDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "db_query_duration_seconds",
		Help:    "Duration of database queries in seconds.",
		Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
	}, []string{"query"})

	dbConnectionsOpen = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "db_connections_open",
		Help: "Number of open database connections.",
	})

	dbConnectionsIdle = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "db_connections_idle",
		Help: "Number of idle database connections.",
	})

	dbConnectionsMaxOpen = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "db_connections_max_open",
		Help: "Maximum number of open database connections.",
	})
)

// ── Entity operation metrics ────────────────────────────────────────────────

var entityOperationsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "entity_operations_total",
	Help: "Total number of entity create/update/delete operations.",
}, []string{"entity", "operation"})

// RecordEntityOp increments the entity operations counter.
func RecordEntityOp(entity, operation string) {
	entityOperationsTotal.WithLabelValues(entity, operation).Inc()
}

// ── Gemini API metrics ──────────────────────────────────────────────────────

var (
	geminiRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "gemini_requests_total",
		Help: "Total number of Gemini API requests.",
	}, []string{"status"})

	geminiRequestDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "gemini_request_duration_seconds",
		Help:    "Duration of Gemini API requests in seconds.",
		Buckets: []float64{0.5, 1, 2.5, 5, 10, 30, 60},
	})

	geminiRetriesTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "gemini_retries_total",
		Help: "Total number of Gemini API retry attempts.",
	})
)

// RecordGeminiRequest records a Gemini API call outcome.
func RecordGeminiRequest(status string, duration time.Duration) {
	geminiRequestsTotal.WithLabelValues(status).Inc()
	geminiRequestDuration.Observe(duration.Seconds())
}

// RecordGeminiRetry increments the Gemini retry counter.
func RecordGeminiRetry() {
	geminiRetriesTotal.Inc()
}

// ── App info metric ─────────────────────────────────────────────────────────

var appInfo = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Name: "app_info",
	Help: "Application build information.",
}, []string{"version"})

func init() {
	appInfo.WithLabelValues(Version).Set(1)
}

// ── Prometheus HTTP middleware ───────────────────────────────────────────────

// normalizePath converts request paths to low-cardinality labels.
// e.g. "/api/feedings/42" → "/api/feedings/:id"
func normalizePath(path string) string {
	// Only normalize /api/ paths
	if !strings.HasPrefix(path, "/api/") {
		return path
	}

	parts := strings.Split(strings.TrimRight(path, "/"), "/")
	// paths like /api/feedings/123  → ["", "api", "feedings", "123"]
	// paths like /api/growth/45     → ["", "api", "growth", "45"]
	if len(parts) >= 4 {
		// Check if the last segment is a numeric ID
		if _, err := strconv.Atoi(parts[len(parts)-1]); err == nil {
			parts[len(parts)-1] = ":id"
		}
	}
	return strings.Join(parts, "/")
}

// PrometheusMiddleware records HTTP request metrics.
func PrometheusMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip metrics endpoint itself to avoid self-scraping noise
		if r.URL.Path == "/metrics" || r.URL.Path == "/healthz" || r.URL.Path == "/readyz" {
			next.ServeHTTP(w, r)
			return
		}

		httpRequestsInFlight.Inc()
		defer httpRequestsInFlight.Dec()

		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)
		duration := time.Since(start)

		path := normalizePath(r.URL.Path)
		httpRequestsTotal.WithLabelValues(r.Method, path, strconv.Itoa(sw.status)).Inc()
		httpRequestDuration.WithLabelValues(r.Method, path).Observe(duration.Seconds())
	})
}

// statusWriter wraps http.ResponseWriter to capture the status code.
type statusWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (sw *statusWriter) WriteHeader(code int) {
	if !sw.wroteHeader {
		sw.status = code
		sw.wroteHeader = true
	}
	sw.ResponseWriter.WriteHeader(code)
}

func (sw *statusWriter) Write(b []byte) (int, error) {
	if !sw.wroteHeader {
		sw.wroteHeader = true
	}
	return sw.ResponseWriter.Write(b)
}

// ── DB stats collector ──────────────────────────────────────────────────────

// StartDBStatsCollector periodically updates the DB connection pool gauges.
func StartDBStatsCollector(db *sql.DB, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			stats := db.Stats()
			dbConnectionsOpen.Set(float64(stats.OpenConnections))
			dbConnectionsIdle.Set(float64(stats.Idle))
			dbConnectionsMaxOpen.Set(float64(stats.MaxOpenConnections))
		}
	}()
}

// ObserveDBQuery records the duration of a database query.
func ObserveDBQuery(queryName string, start time.Time) {
	dbQueryDuration.WithLabelValues(queryName).Observe(time.Since(start).Seconds())
}
