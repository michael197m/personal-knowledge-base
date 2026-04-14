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

type HealthHandler struct {
	db dbPinger
}

type healthResponse struct {
	Status    string `json:"status"`
	Service   string `json:"service"`
	Timestamp string `json:"timestamp"`
	Database  string `json:"database"`
}

func NewHealthHandler(db dbPinger) *HealthHandler {
	return &HealthHandler{db: db}
}

func (h *HealthHandler) Get(w http.ResponseWriter, r *http.Request) {
	dbStatus := "ok"
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := h.db.Ping(ctx); err != nil && err != pgx.ErrTxClosed {
		dbStatus = "unavailable"
	}

	respondJSON(w, http.StatusOK, healthResponse{
		Status:    "ok",
		Service:   "personal-knowledge-base-api",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Database:  dbStatus,
	})
}

func respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
