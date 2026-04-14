package store

import (
	"context"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type mockDB struct {
	queryFn   func(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	beginTxFn func(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error)
	execFn    func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	queryRow  *mockRow
}

func (m *mockDB) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	if m.queryFn == nil {
		return nil, nil
	}

	return m.queryFn(ctx, sql, args...)
}

func (m *mockDB) BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error) {
	if m.beginTxFn == nil {
		return nil, nil
	}

	return m.beginTxFn(ctx, txOptions)
}

func (m *mockDB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return m.queryRow
}

func (m *mockDB) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	if m.execFn == nil {
		return pgconn.CommandTag{}, nil
	}

	return m.execFn(ctx, sql, args...)
}

type mockTx struct {
	queryRows []*mockRow
	execFn    func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	commitErr error
}

func (m *mockTx) Begin(ctx context.Context) (pgx.Tx, error) {
	return nil, errors.New("not implemented")
}
func (m *mockTx) Commit(ctx context.Context) error   { return m.commitErr }
func (m *mockTx) Rollback(ctx context.Context) error { return nil }
func (m *mockTx) CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error) {
	return 0, errors.New("not implemented")
}
func (m *mockTx) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults {
	return nil
}
func (m *mockTx) LargeObjects() pgx.LargeObjects { return pgx.LargeObjects{} }
func (m *mockTx) Prepare(ctx context.Context, name, sql string) (*pgconn.StatementDescription, error) {
	return nil, errors.New("not implemented")
}
func (m *mockTx) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	if m.execFn == nil {
		return pgconn.CommandTag{}, nil
	}

	return m.execFn(ctx, sql, args...)
}
func (m *mockTx) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return nil, errors.New("not implemented")
}
func (m *mockTx) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	if len(m.queryRows) == 0 {
		return &mockRow{err: errors.New("unexpected query row call")}
	}

	row := m.queryRows[0]
	m.queryRows = m.queryRows[1:]
	return row
}
func (m *mockTx) Conn() *pgx.Conn { return nil }

type mockRows struct {
	rows [][]any
	idx  int
	err  error
}

func (m *mockRows) Close() {}
func (m *mockRows) Err() error {
	return m.err
}
func (m *mockRows) CommandTag() pgconn.CommandTag {
	return pgconn.CommandTag{}
}
func (m *mockRows) FieldDescriptions() []pgconn.FieldDescription {
	return nil
}
func (m *mockRows) Next() bool {
	if m.idx >= len(m.rows) {
		return false
	}

	m.idx++
	return true
}
func (m *mockRows) Scan(dest ...any) error {
	row := m.rows[m.idx-1]
	for i := range dest {
		switch d := dest[i].(type) {
		case *uuid.UUID:
			*d = row[i].(uuid.UUID)
		case *string:
			*d = row[i].(string)
		case *time.Time:
			*d = row[i].(time.Time)
		case *[]string:
			*d = append((*d)[:0], row[i].([]string)...)
		case *float64:
			*d = row[i].(float64)
		default:
			return errors.New("unsupported scan destination")
		}
	}

	return nil
}
func (m *mockRows) Values() ([]any, error) {
	return nil, errors.New("not implemented")
}
func (m *mockRows) RawValues() [][]byte {
	return nil
}
func (m *mockRows) Conn() *pgx.Conn { return nil }

type mockRow struct {
	values []any
	err    error
}

type mockEmbedder struct {
	embedFn func(ctx context.Context, input string) ([]float32, error)
}

func (m mockEmbedder) Embed(ctx context.Context, input string) ([]float32, error) {
	if m.embedFn == nil {
		return nil, nil
	}

	return m.embedFn(ctx, input)
}

func (m *mockRow) Scan(dest ...any) error {
	if m.err != nil {
		return m.err
	}

	for i := range dest {
		switch d := dest[i].(type) {
		case *uuid.UUID:
			*d = m.values[i].(uuid.UUID)
		case *time.Time:
			*d = m.values[i].(time.Time)
		case *string:
			*d = m.values[i].(string)
		default:
			return errors.New("unsupported scan destination")
		}
	}

	return nil
}

func TestNormalizeTags(t *testing.T) {
	got := normalizeTags([]string{" go ", "", "test", "go", "  test  ", "pgx"})
	want := []string{"go", "test", "pgx"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestEnsureDemoUserTxReturnsID(t *testing.T) {
	expectedID := uuid.MustParse("4451e179-d89f-462e-b5ce-123ad2b996c2")
	rower := &mockDB{
		queryRow: &mockRow{values: []any{expectedID}},
	}

	got, err := ensureDemoUserTx(context.Background(), rower)
	if err != nil {
		t.Fatalf("ensureDemoUserTx returned error: %v", err)
	}

	if got != expectedID {
		t.Fatalf("expected %s, got %s", expectedID, got)
	}
}

func TestListNotesReturnsAggregatedRows(t *testing.T) {
	noteID := uuid.MustParse("f723fbea-f557-4d3f-96db-e838ad613fec")
	userID := uuid.MustParse("5087d5b0-0ce0-46fd-b971-9e947f8db8c7")
	createdAt := time.Unix(100, 0).UTC()
	updatedAt := time.Unix(200, 0).UTC()

	store := &NoteStore{
		db: &mockDB{
			queryRow: &mockRow{values: []any{userID}},
			queryFn: func(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
				return &mockRows{
					rows: [][]any{
						{noteID, "Store tests", "Verify list behavior", createdAt, updatedAt, []string{"go", "pgx"}},
					},
				}, nil
			},
		},
	}

	got, err := store.ListNotes(context.Background())
	if err != nil {
		t.Fatalf("ListNotes returned error: %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("expected 1 note, got %d", len(got))
	}

	if got[0].ID != noteID || got[0].Title != "Store tests" {
		t.Fatalf("unexpected note: %+v", got[0])
	}

	if !reflect.DeepEqual(got[0].Tags, []string{"go", "pgx"}) {
		t.Fatalf("unexpected tags: %v", got[0].Tags)
	}
}

func TestCreateNoteNormalizesTagsAndReturnsCreatedNote(t *testing.T) {
	userID := uuid.MustParse("f3a0ae66-b405-4783-b6dc-c93e1a540f8c")
	noteID := uuid.MustParse("c3756dad-f71c-485b-8a9d-fafeb55f3ea7")
	tagOneID := uuid.MustParse("f5f03759-d821-4ed7-a485-3699762ab8b6")
	tagTwoID := uuid.MustParse("78b8c59c-9838-46a2-8aa5-dde4df9b0e95")
	createdAt := time.Unix(300, 0).UTC()
	execCalls := 0

	tx := &mockTx{
		queryRows: []*mockRow{
			{values: []any{userID}},
			{values: []any{noteID, createdAt, createdAt}},
			{values: []any{tagOneID}},
			{values: []any{tagTwoID}},
		},
		execFn: func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
			execCalls++
			return pgconn.CommandTag{}, nil
		},
	}

	store := &NoteStore{
		db: &mockDB{
			beginTxFn: func(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error) {
				return tx, nil
			},
		},
	}

	got, err := store.CreateNote(context.Background(), CreateNoteInput{
		Title:   "  Test title  ",
		Content: "  Test content  ",
		Tags:    []string{" go ", "", "go", "pgx"},
	})
	if err != nil {
		t.Fatalf("CreateNote returned error: %v", err)
	}

	if got.ID != noteID {
		t.Fatalf("expected note ID %s, got %s", noteID, got.ID)
	}

	if got.Title != "Test title" || got.Content != "Test content" {
		t.Fatalf("expected trimmed title/content, got %+v", got)
	}

	if !reflect.DeepEqual(got.Tags, []string{"go", "pgx"}) {
		t.Fatalf("expected normalized tags, got %v", got.Tags)
	}

	if execCalls != 3 {
		t.Fatalf("expected delete plus 2 note_tags exec calls, got %d", execCalls)
	}
}

func TestCreateNoteBeginTxError(t *testing.T) {
	store := &NoteStore{
		db: &mockDB{
			beginTxFn: func(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error) {
				return nil, errors.New("begin failed")
			},
		},
	}

	_, err := store.CreateNote(context.Background(), CreateNoteInput{Title: "t", Content: "c"})
	if err == nil {
		t.Fatal("expected CreateNote to return an error")
	}
}

func TestUpdateNoteReturnsNotFound(t *testing.T) {
	userID := uuid.MustParse("95ac94be-b7af-42e6-ad57-52059db632c3")
	noteID := uuid.MustParse("db7f880a-f0f1-4827-a326-a6827fb26b70")
	tx := &mockTx{
		queryRows: []*mockRow{
			{values: []any{userID}},
			{err: pgx.ErrNoRows},
		},
	}

	store := &NoteStore{
		db: &mockDB{
			beginTxFn: func(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error) {
				return tx, nil
			},
		},
	}

	_, err := store.UpdateNote(context.Background(), noteID, UpdateNoteInput{Title: "t", Content: "c"})
	if !errors.Is(err, ErrNoteNotFound) {
		t.Fatalf("expected ErrNoteNotFound, got %v", err)
	}
}

func TestUpdateNoteReplacesTags(t *testing.T) {
	userID := uuid.MustParse("bff2b12c-1777-43b1-aa7e-eb4db4bb6059")
	noteID := uuid.MustParse("66a6e595-c9ce-4dcb-b497-ab612a707857")
	tagID := uuid.MustParse("2899e216-6bd7-4630-9bd0-4d1cba321f92")
	createdAt := time.Unix(100, 0).UTC()
	updatedAt := time.Unix(200, 0).UTC()
	execCalls := 0

	tx := &mockTx{
		queryRows: []*mockRow{
			{values: []any{userID}},
			{values: []any{createdAt, updatedAt}},
			{values: []any{tagID}},
		},
		execFn: func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
			execCalls++
			return pgconn.CommandTag{}, nil
		},
	}

	store := &NoteStore{
		db: &mockDB{
			beginTxFn: func(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error) {
				return tx, nil
			},
		},
	}

	got, err := store.UpdateNote(context.Background(), noteID, UpdateNoteInput{
		Title:   "  Updated  ",
		Content: "  Changed  ",
		Tags:    []string{" go ", "go"},
	})
	if err != nil {
		t.Fatalf("UpdateNote returned error: %v", err)
	}

	if got.ID != noteID || got.Title != "Updated" || got.Content != "Changed" {
		t.Fatalf("unexpected updated note: %+v", got)
	}

	if !reflect.DeepEqual(got.Tags, []string{"go"}) {
		t.Fatalf("expected normalized tags, got %v", got.Tags)
	}

	if execCalls != 2 {
		t.Fatalf("expected delete+insert tag exec calls, got %d", execCalls)
	}
}

func TestDeleteNoteReturnsNotFound(t *testing.T) {
	userID := uuid.MustParse("4d99e4cc-9c9f-4f0e-9d0f-a63c2d42ebdb")
	store := &NoteStore{
		db: &mockDB{
			queryRow: &mockRow{values: []any{userID}},
			execFn: func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
				return pgconn.NewCommandTag("DELETE 0"), nil
			},
		},
	}

	err := store.DeleteNote(context.Background(), uuid.MustParse("9579e453-5701-4f71-ac98-ae9ec8bff46e"))
	if !errors.Is(err, ErrNoteNotFound) {
		t.Fatalf("expected ErrNoteNotFound, got %v", err)
	}
}

func TestDeleteNoteDeletesExistingNote(t *testing.T) {
	userID := uuid.MustParse("316902f5-bc15-4cc2-bfe2-aae6f40eb1cd")
	store := &NoteStore{
		db: &mockDB{
			queryRow: &mockRow{values: []any{userID}},
			execFn: func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
				return pgconn.NewCommandTag("DELETE 1"), nil
			},
		},
	}

	if err := store.DeleteNote(context.Background(), uuid.MustParse("761ec314-5a39-4cf9-91c4-c87e35cbc926")); err != nil {
		t.Fatalf("DeleteNote returned error: %v", err)
	}
}

func TestListNotesQueryError(t *testing.T) {
	store := &NoteStore{
		db: &mockDB{
			queryRow: &mockRow{values: []any{uuid.MustParse("2bc4e2d5-3662-45a9-a57f-6bf514034621")}},
			queryFn: func(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
				return nil, io.EOF
			},
		},
	}

	_, err := store.ListNotes(context.Background())
	if !errors.Is(err, io.EOF) {
		t.Fatalf("expected io.EOF, got %v", err)
	}
}

func TestSearchNotesUsesQueryFilter(t *testing.T) {
	userID := uuid.MustParse("a0927082-aa7b-49f6-a92b-d55e78a2b0e8")
	noteID := uuid.MustParse("9be2c9d9-5daa-4a07-b8e5-c8dc596e3407")
	createdAt := time.Unix(100, 0).UTC()
	updatedAt := time.Unix(200, 0).UTC()
	var gotArgs []any

	store := &NoteStore{
		db: &mockDB{
			queryRow: &mockRow{values: []any{userID}},
			queryFn: func(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
				gotArgs = args
				return &mockRows{
					rows: [][]any{
						{noteID, "Search title", "Search content", createdAt, updatedAt, []string{"search"}},
					},
				}, nil
			},
		},
	}

	got, err := store.SearchNotes(context.Background(), "search")
	if err != nil {
		t.Fatalf("SearchNotes returned error: %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("expected 1 note, got %d", len(got))
	}

	if len(gotArgs) != 3 || gotArgs[1] != "search" || gotArgs[2] != "%search%" {
		t.Fatalf("unexpected query args: %#v", gotArgs)
	}
}

func TestSearchNotesUsesSemanticQueryWhenEmbedderAvailable(t *testing.T) {
	userID := uuid.MustParse("f25d962f-c529-4764-b99c-221ebf6128c4")
	noteID := uuid.MustParse("fa20a651-c423-4fdb-a6d9-8d1b0db26479")
	createdAt := time.Unix(100, 0).UTC()
	updatedAt := time.Unix(200, 0).UTC()
	var gotSQL string
	var gotArgs []any

	store := &NoteStore{
		db: &mockDB{
			queryRow: &mockRow{values: []any{userID}},
			queryFn: func(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
				gotSQL = sql
				gotArgs = args
				return &mockRows{
					rows: [][]any{
						{noteID, "Semantic result", "Matched by embedding", createdAt, updatedAt, []string{"vector"}, 0.12},
					},
				}, nil
			},
		},
		embedder: mockEmbedder{
			embedFn: func(ctx context.Context, input string) ([]float32, error) {
				return []float32{0.1, 0.2, 0.3}, nil
			},
		},
	}

	got, err := store.SearchNotes(context.Background(), "knowledge query")
	if err != nil {
		t.Fatalf("SearchNotes returned error: %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("expected 1 note, got %d", len(got))
	}

	if gotSQL != semanticSearchSQL {
		t.Fatal("expected semantic search SQL to be used")
	}

	if len(gotArgs) != 3 || gotArgs[1] != "[0.1,0.2,0.3]" || gotArgs[2] != semanticSearchMaxDistance {
		t.Fatalf("unexpected semantic search args: %#v", gotArgs)
	}
}

func TestSearchNotesFallsBackToTextQueryWhenEmbeddingFails(t *testing.T) {
	userID := uuid.MustParse("9ef20c06-7eca-40a0-a6e0-87d17d80af4e")
	var gotSQL string

	store := &NoteStore{
		db: &mockDB{
			queryRow: &mockRow{values: []any{userID}},
			queryFn: func(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
				gotSQL = sql
				return &mockRows{}, nil
			},
		},
		embedder: mockEmbedder{
			embedFn: func(ctx context.Context, input string) ([]float32, error) {
				return nil, errors.New("ollama unavailable")
			},
		},
	}

	if _, err := store.SearchNotes(context.Background(), "fallback"); err != nil {
		t.Fatalf("SearchNotes returned error: %v", err)
	}

	if gotSQL != noteQuerySQL {
		t.Fatal("expected text fallback SQL to be used")
	}
}

func TestSearchNotesFallsBackToTextQueryWhenSemanticSearchHasNoRelevantMatches(t *testing.T) {
	userID := uuid.MustParse("d3ff8dd3-cb09-4faf-a2d5-17d12dce5b20")
	callCount := 0

	store := &NoteStore{
		db: &mockDB{
			queryRow: &mockRow{values: []any{userID}},
			queryFn: func(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
				callCount++
				if callCount == 1 {
					if sql != semanticSearchSQL {
						t.Fatalf("expected semantic SQL on first call, got %q", sql)
					}
					return &mockRows{}, nil
				}

				if sql != noteQuerySQL {
					t.Fatalf("expected text SQL on fallback call, got %q", sql)
				}

				return &mockRows{}, nil
			},
		},
		embedder: mockEmbedder{
			embedFn: func(ctx context.Context, input string) ([]float32, error) {
				return []float32{0.1, 0.2, 0.3}, nil
			},
		},
	}

	if _, err := store.SearchNotes(context.Background(), "fallback"); err != nil {
		t.Fatalf("SearchNotes returned error: %v", err)
	}

	if callCount != 2 {
		t.Fatalf("expected semantic search then text fallback, got %d calls", callCount)
	}
}

func TestSearchNoteResultsWithDiagnosticsReportsSemanticNoMatchFallback(t *testing.T) {
	userID := uuid.MustParse("6ff3c952-3733-4ce0-9f7f-020694671d03")
	callCount := 0

	store := &NoteStore{
		db: &mockDB{
			queryRow: &mockRow{values: []any{userID}},
			queryFn: func(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
				callCount++
				if callCount == 1 {
					return &mockRows{}, nil
				}

				return &mockRows{
					rows: [][]any{
						{
							uuid.MustParse("dfbb6b80-8fd6-47dc-97b3-d42c01a5d8c5"),
							"Text hit",
							"fallback note",
							time.Unix(1, 0).UTC(),
							time.Unix(2, 0).UTC(),
							[]string{"text"},
						},
					},
				}, nil
			},
		},
		embedder: mockEmbedder{
			embedFn: func(ctx context.Context, input string) ([]float32, error) {
				return []float32{0.1, 0.2, 0.3}, nil
			},
		},
	}

	search, err := store.SearchNoteResultsWithDiagnostics(context.Background(), "fallback")
	if err != nil {
		t.Fatalf("SearchNoteResultsWithDiagnostics returned error: %v", err)
	}

	if search.Diagnostics.Mode != "text" {
		t.Fatalf("expected mode text, got %q", search.Diagnostics.Mode)
	}
	if search.Diagnostics.FallbackReason != "semantic_no_match" {
		t.Fatalf("expected fallback reason semantic_no_match, got %q", search.Diagnostics.FallbackReason)
	}
	if len(search.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(search.Results))
	}
}

func TestSearchNoteResultsWithDiagnosticsReportsSemanticMode(t *testing.T) {
	userID := uuid.MustParse("6898498f-c8d7-4ebf-bfff-143558710d42")

	store := &NoteStore{
		db: &mockDB{
			queryRow: &mockRow{values: []any{userID}},
			queryFn: func(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
				return &mockRows{
					rows: [][]any{
						{
							uuid.MustParse("31b5f652-f2ef-44f3-adf8-c3686ab8887b"),
							"Semantic hit",
							"vector match",
							time.Unix(1, 0).UTC(),
							time.Unix(2, 0).UTC(),
							[]string{"semantic"},
							0.11,
						},
					},
				}, nil
			},
		},
		embedder: mockEmbedder{
			embedFn: func(ctx context.Context, input string) ([]float32, error) {
				return []float32{0.1, 0.2, 0.3}, nil
			},
		},
	}

	search, err := store.SearchNoteResultsWithDiagnostics(context.Background(), "semantic")
	if err != nil {
		t.Fatalf("SearchNoteResultsWithDiagnostics returned error: %v", err)
	}

	if search.Diagnostics.Mode != "semantic" {
		t.Fatalf("expected mode semantic, got %q", search.Diagnostics.Mode)
	}
	if search.Diagnostics.FallbackReason != "" {
		t.Fatalf("expected no fallback reason, got %q", search.Diagnostics.FallbackReason)
	}
}

func TestBackfillMissingEmbeddingsUpdatesEmbeddableNotes(t *testing.T) {
	var updatedIDs []uuid.UUID

	store := &NoteStore{
		db: &mockDB{
			queryFn: func(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
				if sql != missingEmbeddingsSQL {
					t.Fatalf("unexpected SQL for backfill query: %q", sql)
				}

				return &mockRows{
					rows: [][]any{
						{
							uuid.MustParse("9507f97d-8719-44f4-ba99-eb6f3759b9ce"),
							"Backfill A",
							"content A",
							[]string{"one"},
						},
						{
							uuid.MustParse("9b3d8e5f-981a-488f-aa5d-285719d06711"),
							"Backfill B",
							"content B",
							[]string{"two"},
						},
					},
				}, nil
			},
			execFn: func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
				if sql != updateNoteEmbeddingSQL {
					t.Fatalf("unexpected SQL for embedding update: %q", sql)
				}

				updatedIDs = append(updatedIDs, args[0].(uuid.UUID))
				return pgconn.CommandTag{}, nil
			},
		},
		embedder: mockEmbedder{
			embedFn: func(ctx context.Context, input string) ([]float32, error) {
				if strings.Contains(input, "Backfill B") {
					return nil, errors.New("embedder unavailable")
				}

				return []float32{0.1, 0.2, 0.3}, nil
			},
		},
	}

	report, err := store.BackfillMissingEmbeddings(context.Background())
	if err != nil {
		t.Fatalf("BackfillMissingEmbeddings returned error: %v", err)
	}

	if report.Scanned != 2 || report.Updated != 1 || report.Failed != 1 {
		t.Fatalf("unexpected report: %+v", report)
	}
	if len(updatedIDs) != 1 {
		t.Fatalf("expected 1 updated ID, got %d", len(updatedIDs))
	}
}

func TestBackfillMissingEmbeddingsRequiresEmbedder(t *testing.T) {
	store := &NoteStore{
		db: &mockDB{},
	}

	_, err := store.BackfillMissingEmbeddings(context.Background())
	if err == nil {
		t.Fatal("expected error when embedder is not configured")
	}
}
