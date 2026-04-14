package http

import (
	"context"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"personal-knowledge-base/backend/internal/config"
	"personal-knowledge-base/backend/internal/http/handlers"
	"personal-knowledge-base/backend/internal/store"
)

type embeddingHealthChecker interface {
	Health(ctx context.Context) error
	Model() string
}

type Server struct {
	config    config.Config
	db        *pgxpool.Pool
	noteStore *store.NoteStore
	embedder  embeddingHealthChecker
}

func NewServer(cfg config.Config, db *pgxpool.Pool, noteStore *store.NoteStore, embedders ...embeddingHealthChecker) *Server {
	server := &Server{
		config:    cfg,
		db:        db,
		noteStore: noteStore,
	}
	if len(embedders) > 0 {
		server.embedder = embedders[0]
	}

	return server
}

func (s *Server) Router() http.Handler {
	mux := http.NewServeMux()
	healthHandler := handlers.NewHealthHandler(s.db, s.embedder)
	noteHandler := handlers.NewNoteHandler(s.noteStore)

	mux.HandleFunc("GET /api/v1/health", healthHandler.Get)
	mux.HandleFunc("GET /api/v1/notes", noteHandler.List)
	mux.HandleFunc("GET /api/v1/search", noteHandler.Search)
	mux.HandleFunc("POST /api/v1/notes", noteHandler.Create)
	mux.HandleFunc("PUT /api/v1/notes/{id}", noteHandler.Update)
	mux.HandleFunc("DELETE /api/v1/notes/{id}", noteHandler.Delete)

	return withCORS(mux)
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
