package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
)

type dbPinger interface {
	Ping(ctx context.Context) error
}

type embeddingHealthChecker interface {
	Health(ctx context.Context) error
	Model() string
}

type HealthHandler struct {
	db       dbPinger
	embedder embeddingHealthChecker
}

type healthResponse struct {
	Status     string `json:"status"`
	Service    string `json:"service"`
	Timestamp  string `json:"timestamp"`
	Database   string `json:"database"`
	Embeddings struct {
		Status string `json:"status"`
		Model  string `json:"model,omitempty"`
	} `json:"embeddings"`
}

func NewHealthHandler(db dbPinger, embedders ...embeddingHealthChecker) *HealthHandler {
	handler := &HealthHandler{db: db}
	if len(embedders) > 0 {
		handler.embedder = embedders[0]
	}

	return handler
}

func (h *HealthHandler) Get(w http.ResponseWriter, r *http.Request) {
	dbStatus := "ok"
	embeddingStatus := "unconfigured"
	var embeddingModel string
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := h.db.Ping(ctx); err != nil && err != pgx.ErrTxClosed {
		dbStatus = "unavailable"
	}
	if h.embedder != nil {
		embeddingModel = h.embedder.Model()
		embeddingStatus = "ok"
		if err := h.embedder.Health(ctx); err != nil {
			embeddingStatus = "unavailable"
		}
	}

	payload := healthResponse{
		Status:    "ok",
		Service:   "personal-knowledge-base-api",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Database:  dbStatus,
	}
	payload.Embeddings.Status = embeddingStatus
	payload.Embeddings.Model = embeddingModel

	respondJSON(w, http.StatusOK, payload)
}

func respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
