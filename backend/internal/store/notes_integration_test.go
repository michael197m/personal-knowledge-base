package store

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestNoteStoreIntegrationCreateAndListNotes(t *testing.T) {
	ctx := context.Background()
	pool := openIntegrationTestPool(t, ctx)
	resetIntegrationDatabase(t, ctx, pool)

	store := NewNoteStore(pool)

	created, err := store.CreateNote(ctx, CreateNoteInput{
		Title:   "  Integration note  ",
		Content: "  Persist this through Postgres  ",
		Tags:    []string{"pgx", " go ", "", "pgx"},
	})
	if err != nil {
		t.Fatalf("CreateNote returned error: %v", err)
	}

	if created.Title != "Integration note" || created.Content != "Persist this through Postgres" {
		t.Fatalf("expected trimmed fields, got %+v", created)
	}

	if !reflect.DeepEqual(created.Tags, []string{"pgx", "go"}) {
		t.Fatalf("expected normalized tags, got %v", created.Tags)
	}

	listed, err := store.ListNotes(ctx)
	if err != nil {
		t.Fatalf("ListNotes returned error: %v", err)
	}

	if len(listed) != 1 {
		t.Fatalf("expected 1 note, got %d", len(listed))
	}

	if listed[0].ID != created.ID {
		t.Fatalf("expected listed note ID %s, got %s", created.ID, listed[0].ID)
	}

	if !reflect.DeepEqual(listed[0].Tags, []string{"go", "pgx"}) {
		t.Fatalf("expected tags ordered from query, got %v", listed[0].Tags)
	}

	var userCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM users WHERE email = $1`, demoUserEmail).Scan(&userCount); err != nil {
		t.Fatalf("count demo user: %v", err)
	}

	if userCount != 1 {
		t.Fatalf("expected exactly 1 demo user, got %d", userCount)
	}
}

func TestNoteStoreIntegrationListNotesOrdersNewestFirst(t *testing.T) {
	ctx := context.Background()
	pool := openIntegrationTestPool(t, ctx)
	resetIntegrationDatabase(t, ctx, pool)

	store := NewNoteStore(pool)

	first, err := store.CreateNote(ctx, CreateNoteInput{
		Title:   "First note",
		Content: "Older note",
		Tags:    []string{"one"},
	})
	if err != nil {
		t.Fatalf("create first note: %v", err)
	}

	second, err := store.CreateNote(ctx, CreateNoteInput{
		Title:   "Second note",
		Content: "Newer note",
		Tags:    []string{"two"},
	})
	if err != nil {
		t.Fatalf("create second note: %v", err)
	}

	listed, err := store.ListNotes(ctx)
	if err != nil {
		t.Fatalf("ListNotes returned error: %v", err)
	}

	if len(listed) != 2 {
		t.Fatalf("expected 2 notes, got %d", len(listed))
	}

	if listed[0].ID != second.ID || listed[1].ID != first.ID {
		t.Fatalf("expected newest note first, got IDs %s then %s", listed[0].ID, listed[1].ID)
	}
}

func TestNoteStoreIntegrationUpdateNoteReplacesTags(t *testing.T) {
	ctx := context.Background()
	pool := openIntegrationTestPool(t, ctx)
	resetIntegrationDatabase(t, ctx, pool)

	store := NewNoteStore(pool)

	created, err := store.CreateNote(ctx, CreateNoteInput{
		Title:   "Original",
		Content: "Original content",
		Tags:    []string{"old", "keep"},
	})
	if err != nil {
		t.Fatalf("create note: %v", err)
	}

	updated, err := store.UpdateNote(ctx, created.ID, UpdateNoteInput{
		Title:   "Updated",
		Content: "Updated content",
		Tags:    []string{"new", "keep", "new"},
	})
	if err != nil {
		t.Fatalf("update note: %v", err)
	}

	if updated.Title != "Updated" || updated.Content != "Updated content" {
		t.Fatalf("unexpected updated note: %+v", updated)
	}

	if !reflect.DeepEqual(updated.Tags, []string{"new", "keep"}) {
		t.Fatalf("unexpected updated tags: %v", updated.Tags)
	}

	listed, err := store.ListNotes(ctx)
	if err != nil {
		t.Fatalf("list notes after update: %v", err)
	}

	if len(listed) != 1 {
		t.Fatalf("expected 1 note, got %d", len(listed))
	}

	if !reflect.DeepEqual(listed[0].Tags, []string{"keep", "new"}) {
		t.Fatalf("expected replaced tags, got %v", listed[0].Tags)
	}
}

func TestNoteStoreIntegrationDeleteNoteRemovesIt(t *testing.T) {
	ctx := context.Background()
	pool := openIntegrationTestPool(t, ctx)
	resetIntegrationDatabase(t, ctx, pool)

	store := NewNoteStore(pool)

	created, err := store.CreateNote(ctx, CreateNoteInput{
		Title:   "Delete me",
		Content: "Soon gone",
		Tags:    []string{"temp"},
	})
	if err != nil {
		t.Fatalf("create note: %v", err)
	}

	if err := store.DeleteNote(ctx, created.ID); err != nil {
		t.Fatalf("delete note: %v", err)
	}

	listed, err := store.ListNotes(ctx)
	if err != nil {
		t.Fatalf("list notes after delete: %v", err)
	}

	if len(listed) != 0 {
		t.Fatalf("expected 0 notes after delete, got %d", len(listed))
	}

	if err := store.DeleteNote(ctx, created.ID); !errors.Is(err, ErrNoteNotFound) {
		t.Fatalf("expected ErrNoteNotFound on second delete, got %v", err)
	}
}

func TestNoteStoreIntegrationSearchNotesMatchesTitleContentAndTags(t *testing.T) {
	ctx := context.Background()
	pool := openIntegrationTestPool(t, ctx)
	resetIntegrationDatabase(t, ctx, pool)

	store := NewNoteStore(pool)

	if _, err := store.CreateNote(ctx, CreateNoteInput{
		Title:   "Graph ideas",
		Content: "Link notes by concepts",
		Tags:    []string{"pkb"},
	}); err != nil {
		t.Fatalf("create title match note: %v", err)
	}

	if _, err := store.CreateNote(ctx, CreateNoteInput{
		Title:   "Daily note",
		Content: "Semantic search exploration",
		Tags:    []string{"journal"},
	}); err != nil {
		t.Fatalf("create content match note: %v", err)
	}

	if _, err := store.CreateNote(ctx, CreateNoteInput{
		Title:   "Reference",
		Content: "Something else",
		Tags:    []string{"search"},
	}); err != nil {
		t.Fatalf("create tag match note: %v", err)
	}

	results, err := store.SearchNotes(ctx, "search")
	if err != nil {
		t.Fatalf("search notes: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 search results, got %d", len(results))
	}

	if results[0].Title != "Daily note" {
		t.Fatalf("expected content match first by recency/order rules, got %q", results[0].Title)
	}

	results, err = store.SearchNotes(ctx, "graph")
	if err != nil {
		t.Fatalf("search notes by title: %v", err)
	}

	if len(results) != 1 || results[0].Title != "Graph ideas" {
		t.Fatalf("unexpected title search results: %+v", results)
	}
}

func TestNoteStoreIntegrationSemanticSearchUsesVectorSimilarity(t *testing.T) {
	ctx := context.Background()
	pool := openIntegrationTestPool(t, ctx)
	resetIntegrationDatabase(t, ctx, pool)

	store := NewNoteStore(pool)

	first, err := store.CreateNote(ctx, CreateNoteInput{
		Title:   "Vectors one",
		Content: "First semantic note",
		Tags:    []string{"alpha"},
	})
	if err != nil {
		t.Fatalf("create first note: %v", err)
	}

	second, err := store.CreateNote(ctx, CreateNoteInput{
		Title:   "Vectors two",
		Content: "Second semantic note",
		Tags:    []string{"beta"},
	})
	if err != nil {
		t.Fatalf("create second note: %v", err)
	}

	if _, err := pool.Exec(ctx, `UPDATE notes SET embedding = $2::vector WHERE id = $1`, first.ID, vectorLiteral(unitVectorAt(0))); err != nil {
		t.Fatalf("set first embedding: %v", err)
	}

	if _, err := pool.Exec(ctx, `UPDATE notes SET embedding = $2::vector WHERE id = $1`, second.ID, vectorLiteral(unitVectorAt(1))); err != nil {
		t.Fatalf("set second embedding: %v", err)
	}

	semanticStore := NewNoteStore(pool, mockEmbedder{
		embedFn: func(ctx context.Context, input string) ([]float32, error) {
			return unitVectorAt(0), nil
		},
	})

	results, err := semanticStore.SearchNotes(ctx, "vector-ish query")
	if err != nil {
		t.Fatalf("semantic search: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 semantic result after thresholding, got %d", len(results))
	}

	if results[0].ID != first.ID {
		t.Fatalf("expected first note to rank highest, got %s", results[0].ID)
	}
}

func unitVectorAt(index int) []float32 {
	vector := make([]float32, 768)
	vector[index] = 1
	return vector
}

func openIntegrationTestPool(t *testing.T, ctx context.Context) *pgxpool.Pool {
	t.Helper()

	databaseURL := strings.TrimSpace(os.Getenv("PKB_TEST_DATABASE_URL"))
	if databaseURL == "" {
		t.Skip("set PKB_TEST_DATABASE_URL to run store integration tests")
	}

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("create pgx pool: %v", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Fatalf("ping integration database: %v", err)
	}

	applyIntegrationSchema(t, ctx, pool)
	t.Cleanup(pool.Close)

	return pool
}

func applyIntegrationSchema(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()

	for _, name := range []string{"001_extensions.sql", "002_schema.sql"} {
		scriptPath := integrationScriptPath(t, name)
		script, err := os.ReadFile(scriptPath)
		if err != nil {
			t.Fatalf("read schema script %s: %v", scriptPath, err)
		}

		if _, err := pool.Exec(ctx, string(script)); err != nil {
			t.Fatalf("apply schema script %s: %v", scriptPath, err)
		}
	}
}

func resetIntegrationDatabase(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()

	if _, err := pool.Exec(ctx, `TRUNCATE TABLE note_tags, tags, notes, users RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("truncate integration tables: %v", err)
	}
}

func integrationScriptPath(t *testing.T, name string) string {
	t.Helper()

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("unable to resolve current test file path")
	}

	return filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", "..", "db", "init", name))
}
