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

type mockEmbeddingHealthChecker struct {
	healthErr error
	model     string
}

func (m mockEmbeddingHealthChecker) Health(context.Context) error {
	return m.healthErr
}

func (m mockEmbeddingHealthChecker) Model() string {
	return m.model
}

func TestHealthHandlerReportsDatabaseOK(t *testing.T) {
	handler := NewHealthHandler(mockPinger{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rec := httptest.NewRecorder()

	handler.Get(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if got["database"] != "ok" {
		t.Fatalf("expected database status ok, got %q", got["database"])
	}

	embeddings, ok := got["embeddings"].(map[string]any)
	if !ok {
		t.Fatalf("expected embeddings object, got %#v", got["embeddings"])
	}
	if embeddings["status"] != "unconfigured" {
		t.Fatalf("expected embeddings status unconfigured, got %q", embeddings["status"])
	}
}

func TestHealthHandlerReportsDatabaseUnavailable(t *testing.T) {
	handler := NewHealthHandler(mockPinger{pingErr: errors.New("db down")})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rec := httptest.NewRecorder()

	handler.Get(rec, req)

	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if got["database"] != "unavailable" {
		t.Fatalf("expected database status unavailable, got %q", got["database"])
	}
}

func TestHealthHandlerReportsEmbeddingsOK(t *testing.T) {
	handler := NewHealthHandler(
		mockPinger{},
		mockEmbeddingHealthChecker{model: "nomic-embed-text"},
	)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rec := httptest.NewRecorder()

	handler.Get(rec, req)

	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	embeddings := got["embeddings"].(map[string]any)
	if embeddings["status"] != "ok" {
		t.Fatalf("expected embeddings status ok, got %q", embeddings["status"])
	}
	if embeddings["model"] != "nomic-embed-text" {
		t.Fatalf("expected embeddings model nomic-embed-text, got %q", embeddings["model"])
	}
}

func TestHealthHandlerReportsEmbeddingsUnavailable(t *testing.T) {
	handler := NewHealthHandler(
		mockPinger{},
		mockEmbeddingHealthChecker{
			model:     "nomic-embed-text",
			healthErr: errors.New("ollama down"),
		},
	)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rec := httptest.NewRecorder()

	handler.Get(rec, req)

	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	embeddings := got["embeddings"].(map[string]any)
	if embeddings["status"] != "unavailable" {
		t.Fatalf("expected embeddings status unavailable, got %q", embeddings["status"])
	}
}
