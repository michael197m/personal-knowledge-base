package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"personal-knowledge-base/backend/internal/config"
	"personal-knowledge-base/backend/internal/embeddings"
	internalhttp "personal-knowledge-base/backend/internal/http"
	"personal-knowledge-base/backend/internal/store"
)

func main() {
	cfg := config.Load()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	dbPool, err := store.NewPostgresPool(ctx, cfg.DatabaseURL())
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	defer dbPool.Close()

	var embedder *embeddings.OllamaClient
	if cfg.OllamaBaseURL != "" {
		embedder = embeddings.NewOllamaClient(
			cfg.OllamaBaseURL,
			cfg.OllamaEmbedModel,
			cfg.OllamaTimeout,
		)
	}

	app := internalhttp.NewServer(cfg, dbPool, store.NewNoteStore(dbPool, embedder), embedder)

	server := &http.Server{
		Addr:              ":" + cfg.ServerPort,
		Handler:           app.Router(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("shutdown error: %v", err)
		}
	}()

	log.Printf("api listening on :%s", cfg.ServerPort)

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}
