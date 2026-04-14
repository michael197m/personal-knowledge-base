package http

import (
	"context"
	"net/http"

	"github.com/google/uuid"
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
	userStore *store.UserStore
	auth      authService
	embedder  embeddingHealthChecker
}

type authService interface {
	GenerateToken(userID uuid.UUID) (string, error)
	ParseToken(token string) (uuid.UUID, error)
}

func NewServer(
	cfg config.Config,
	db *pgxpool.Pool,
	noteStore *store.NoteStore,
	userStore *store.UserStore,
	auth authService,
	embedders ...embeddingHealthChecker,
) *Server {
	server := &Server{
		config:    cfg,
		db:        db,
		noteStore: noteStore,
		userStore: userStore,
		auth:      auth,
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
	authHandler := handlers.NewAuthHandler(s.userStore, s.auth)

	mux.HandleFunc("GET /api/v1/health", healthHandler.Get)
	mux.HandleFunc("POST /api/v1/auth/register", authHandler.Register)
	mux.HandleFunc("POST /api/v1/auth/login", authHandler.Login)
	mux.Handle("GET /api/v1/notes", withAuth(s.auth, http.HandlerFunc(noteHandler.List)))
	mux.Handle("GET /api/v1/search", withAuth(s.auth, http.HandlerFunc(noteHandler.Search)))
	mux.Handle("POST /api/v1/notes", withAuth(s.auth, http.HandlerFunc(noteHandler.Create)))
	mux.Handle("PUT /api/v1/notes/{id}", withAuth(s.auth, http.HandlerFunc(noteHandler.Update)))
	mux.Handle("DELETE /api/v1/notes/{id}", withAuth(s.auth, http.HandlerFunc(noteHandler.Delete)))

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
