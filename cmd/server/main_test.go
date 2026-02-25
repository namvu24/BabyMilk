package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCorsMiddleware_APIPath_OptionsRequest(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := corsMiddleware(inner)

	req := httptest.NewRequest(http.MethodOptions, "/api/feedings", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204 for OPTIONS, got %d", w.Code)
	}
	if v := w.Header().Get("Access-Control-Allow-Origin"); v != "*" {
		t.Errorf("expected Allow-Origin *, got %s", v)
	}
	if v := w.Header().Get("Access-Control-Allow-Methods"); v == "" {
		t.Error("expected Allow-Methods header to be set")
	}
	if v := w.Header().Get("Access-Control-Allow-Headers"); v == "" {
		t.Error("expected Allow-Headers header to be set")
	}
}

func TestCorsMiddleware_APIPath_GETRequest(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := corsMiddleware(inner)

	req := httptest.NewRequest(http.MethodGet, "/api/feedings", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if v := w.Header().Get("Access-Control-Allow-Origin"); v != "*" {
		t.Errorf("expected Allow-Origin *, got %s", v)
	}
}

func TestCorsMiddleware_NonAPIPath(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := corsMiddleware(inner)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if v := w.Header().Get("Access-Control-Allow-Origin"); v != "" {
		t.Errorf("expected no CORS headers on non-API path, got Allow-Origin=%s", v)
	}
}
