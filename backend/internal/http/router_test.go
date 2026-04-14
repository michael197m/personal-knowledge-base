package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWithCORSHandlesPreflight(t *testing.T) {
	handlerCalled := false
	handler := withCORS(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	}), "http://localhost:5173")

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/notes", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, rec.Code)
	}

	if handlerCalled {
		t.Fatal("expected preflight request to short-circuit before handler")
	}

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Fatalf("expected allow origin http://localhost:5173, got %q", got)
	}
}

func TestWithCORSPassesThroughNonOptions(t *testing.T) {
	handler := withCORS(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}), "http://localhost:5173")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notes", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d", http.StatusAccepted, rec.Code)
	}

	if got := rec.Header().Get("Access-Control-Allow-Methods"); got == "" {
		t.Fatal("expected CORS methods header to be set")
	}
	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("expected allow credentials true, got %q", got)
	}
}
