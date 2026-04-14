package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type mockPinger struct {
	pingErr error
}

func (m mockPinger) Ping(context.Context) error {
	return m.pingErr
}

func TestHealthHandlerReportsDatabaseOK(t *testing.T) {
	handler := NewHealthHandler(mockPinger{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rec := httptest.NewRecorder()

	handler.Get(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var got map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if got["database"] != "ok" {
		t.Fatalf("expected database status ok, got %q", got["database"])
	}
}

func TestHealthHandlerReportsDatabaseUnavailable(t *testing.T) {
	handler := NewHealthHandler(mockPinger{pingErr: errors.New("db down")})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rec := httptest.NewRecorder()

	handler.Get(rec, req)

	var got map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if got["database"] != "unavailable" {
		t.Fatalf("expected database status unavailable, got %q", got["database"])
	}
}
