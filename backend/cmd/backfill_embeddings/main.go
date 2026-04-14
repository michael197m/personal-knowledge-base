package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"personal-knowledge-base/backend/internal/config"
	"personal-knowledge-base/backend/internal/embeddings"
	"personal-knowledge-base/backend/internal/store"
)

func main() {
	cfg := config.Load()
	if cfg.OllamaBaseURL == "" {
		log.Fatal("OLLAMA_BASE_URL must be set to run embedding backfill")
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	dbPool, err := store.NewPostgresPool(ctx, cfg.DatabaseURL())
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	defer dbPool.Close()

	embedder := embeddings.NewOllamaClient(
		cfg.OllamaBaseURL,
		cfg.OllamaEmbedModel,
		cfg.OllamaTimeout,
	)
	noteStore := store.NewNoteStore(dbPool, embedder)

	report, err := noteStore.BackfillMissingEmbeddings(ctx)
	if err != nil {
		log.Fatalf("embedding backfill failed: %v", err)
	}

	log.Printf(
		"embedding backfill complete: scanned=%d updated=%d failed=%d",
		report.Scanned,
		report.Updated,
		report.Failed,
	)
}
