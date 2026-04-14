package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"personal-knowledge-base/backend/internal/models"
	"personal-knowledge-base/backend/internal/store"
)

type mockNoteStore struct {
	listNotesFn     func(ctx context.Context) ([]models.Note, error)
	searchResultsFn func(ctx context.Context, query string) ([]models.SearchResult, error)
	createNoteFn    func(ctx context.Context, input store.CreateNoteInput) (models.Note, error)
	updateNoteFn    func(ctx context.Context, noteID uuid.UUID, input store.UpdateNoteInput) (models.Note, error)
	deleteNoteFn    func(ctx context.Context, noteID uuid.UUID) error
	receivedInput   store.CreateNoteInput
	receivedID      uuid.UUID
	updateInput     store.UpdateNoteInput
	searchQuery     string
}

func (m *mockNoteStore) ListNotes(ctx context.Context) ([]models.Note, error) {
	if m.listNotesFn == nil {
		return nil, nil
	}

	return m.listNotesFn(ctx)
}

func (m *mockNoteStore) SearchNoteResults(ctx context.Context, query string) ([]models.SearchResult, error) {
	m.searchQuery = query
	if m.searchResultsFn == nil {
		return nil, nil
	}

	return m.searchResultsFn(ctx, query)
}

func (m *mockNoteStore) SearchNoteResultsWithDiagnostics(ctx context.Context, query string) (store.SearchResultsWithDiagnostics, error) {
	results, err := m.SearchNoteResults(ctx, query)
	if err != nil {
		return store.SearchResultsWithDiagnostics{}, err
	}

	return store.SearchResultsWithDiagnostics{
		Results: results,
		Diagnostics: store.SearchDiagnostics{
			Mode: "text",
		},
	}, nil
}

func (m *mockNoteStore) CreateNote(ctx context.Context, input store.CreateNoteInput) (models.Note, error) {
	m.receivedInput = input
	if m.createNoteFn == nil {
		return models.Note{}, nil
	}

	return m.createNoteFn(ctx, input)
}

func (m *mockNoteStore) UpdateNote(ctx context.Context, noteID uuid.UUID, input store.UpdateNoteInput) (models.Note, error) {
	m.receivedID = noteID
	m.updateInput = input
	if m.updateNoteFn == nil {
		return models.Note{}, nil
	}

	return m.updateNoteFn(ctx, noteID, input)
}

func (m *mockNoteStore) DeleteNote(ctx context.Context, noteID uuid.UUID) error {
	m.receivedID = noteID
	if m.deleteNoteFn == nil {
		return nil
	}

	return m.deleteNoteFn(ctx, noteID)
}

func TestNoteHandlerListReturnsNotes(t *testing.T) {
	expected := []models.Note{
		{
			ID:        uuid.MustParse("c1ca93a1-4dc1-45df-aa3b-a63f233c6711"),
			Title:     "First",
			Content:   "Body",
			Tags:      []string{"go"},
			CreatedAt: time.Unix(1, 0).UTC(),
			UpdatedAt: time.Unix(2, 0).UTC(),
		},
	}
	handler := NewNoteHandler(&mockNoteStore{
		listNotesFn: func(context.Context) ([]models.Note, error) {
			return expected, nil
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notes", nil)
	rec := httptest.NewRecorder()

	handler.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var got []models.Note
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(got) != 1 || got[0].Title != expected[0].Title {
		t.Fatalf("unexpected response payload: %+v", got)
	}
}

func TestNoteHandlerListStoreError(t *testing.T) {
	handler := NewNoteHandler(&mockNoteStore{
		listNotesFn: func(context.Context) ([]models.Note, error) {
			return nil, errors.New("boom")
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notes", nil)
	rec := httptest.NewRecorder()

	handler.List(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestNoteHandlerSearchReturnsNotes(t *testing.T) {
	similarity := 0.82
	expected := []models.SearchResult{
		{
			Note: models.Note{
				ID:        uuid.MustParse("1dd611e1-f3e6-46d1-9aa3-c5162a9ec08d"),
				Title:     "Search result",
				Content:   "Matched note",
				Tags:      []string{"search"},
				CreatedAt: time.Unix(1, 0).UTC(),
				UpdatedAt: time.Unix(2, 0).UTC(),
			},
			MatchType:  "semantic",
			Similarity: &similarity,
		},
	}
	store := &mockNoteStore{
		searchResultsFn: func(_ context.Context, query string) ([]models.SearchResult, error) {
			return expected, nil
		},
	}
	handler := NewNoteHandler(store)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/search?q=match", nil)
	rec := httptest.NewRecorder()

	handler.Search(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if store.searchQuery != "match" {
		t.Fatalf("expected query match, got %q", store.searchQuery)
	}
}

func TestNoteHandlerSearchStoreError(t *testing.T) {
	handler := NewNoteHandler(&mockNoteStore{
		searchResultsFn: func(_ context.Context, query string) ([]models.SearchResult, error) {
			return nil, errors.New("boom")
		},
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/search?q=match", nil)
	rec := httptest.NewRecorder()

	handler.Search(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestNoteHandlerCreateRejectsInvalidJSON(t *testing.T) {
	handler := NewNoteHandler(&mockNoteStore{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notes", bytes.NewBufferString("{"))
	rec := httptest.NewRecorder()

	handler.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestNoteHandlerCreateRejectsMissingFields(t *testing.T) {
	handler := NewNoteHandler(&mockNoteStore{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notes", bytes.NewBufferString(`{"title":"","content":""}`))
	rec := httptest.NewRecorder()

	handler.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestNoteHandlerCreateReturnsCreatedNote(t *testing.T) {
	expected := models.Note{
		ID:        uuid.MustParse("5964c5e7-f2ef-42df-aa57-ffb4fa5c12a9"),
		Title:     "API tests",
		Content:   "Cover HTTP behavior",
		Tags:      []string{"go", "test"},
		CreatedAt: time.Unix(10, 0).UTC(),
		UpdatedAt: time.Unix(10, 0).UTC(),
	}
	store := &mockNoteStore{
		createNoteFn: func(_ context.Context, input store.CreateNoteInput) (models.Note, error) {
			return expected, nil
		},
	}
	handler := NewNoteHandler(store)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notes", bytes.NewBufferString(`{
		"title":"API tests",
		"content":"Cover HTTP behavior",
		"tags":["go","test"]
	}`))
	rec := httptest.NewRecorder()

	handler.Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rec.Code)
	}

	if store.receivedInput.Title != "API tests" || len(store.receivedInput.Tags) != 2 {
		t.Fatalf("unexpected create input: %+v", store.receivedInput)
	}

	var got models.Note
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if got.ID != expected.ID || got.Title != expected.Title {
		t.Fatalf("unexpected response payload: %+v", got)
	}
}

func TestNoteHandlerCreateStoreError(t *testing.T) {
	handler := NewNoteHandler(&mockNoteStore{
		createNoteFn: func(_ context.Context, input store.CreateNoteInput) (models.Note, error) {
			return models.Note{}, errors.New("boom")
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notes", bytes.NewBufferString(`{"title":"T","content":"C"}`))
	rec := httptest.NewRecorder()

	handler.Create(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestNoteHandlerUpdateRejectsInvalidID(t *testing.T) {
	handler := NewNoteHandler(&mockNoteStore{})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/notes/not-a-uuid", bytes.NewBufferString(`{"title":"T","content":"C"}`))
	req.SetPathValue("id", "not-a-uuid")
	rec := httptest.NewRecorder()

	handler.Update(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestNoteHandlerUpdateReturnsUpdatedNote(t *testing.T) {
	noteID := uuid.MustParse("671f33d6-db85-4c0f-a6f8-3f8cdd83f7ab")
	expected := models.Note{
		ID:        noteID,
		Title:     "Updated",
		Content:   "Changed",
		Tags:      []string{"go"},
		CreatedAt: time.Unix(10, 0).UTC(),
		UpdatedAt: time.Unix(20, 0).UTC(),
	}
	store := &mockNoteStore{
		updateNoteFn: func(_ context.Context, gotID uuid.UUID, input store.UpdateNoteInput) (models.Note, error) {
			return expected, nil
		},
	}
	handler := NewNoteHandler(store)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/notes/"+noteID.String(), bytes.NewBufferString(`{"title":"Updated","content":"Changed","tags":["go"]}`))
	req.SetPathValue("id", noteID.String())
	rec := httptest.NewRecorder()

	handler.Update(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if store.receivedID != noteID || store.updateInput.Title != "Updated" {
		t.Fatalf("unexpected update call: id=%s input=%+v", store.receivedID, store.updateInput)
	}
}

func TestNoteHandlerUpdateNotFound(t *testing.T) {
	noteID := uuid.MustParse("52f92331-99fb-4059-a44a-780cdba1efaa")
	handler := NewNoteHandler(&mockNoteStore{
		updateNoteFn: func(_ context.Context, gotID uuid.UUID, input store.UpdateNoteInput) (models.Note, error) {
			return models.Note{}, store.ErrNoteNotFound
		},
	})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/notes/"+noteID.String(), bytes.NewBufferString(`{"title":"Updated","content":"Changed"}`))
	req.SetPathValue("id", noteID.String())
	rec := httptest.NewRecorder()

	handler.Update(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestNoteHandlerDeleteReturnsNoContent(t *testing.T) {
	noteID := uuid.MustParse("dc434947-f9c2-4c14-aa4c-a3afb33f5c91")
	store := &mockNoteStore{}
	handler := NewNoteHandler(store)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/notes/"+noteID.String(), nil)
	req.SetPathValue("id", noteID.String())
	rec := httptest.NewRecorder()

	handler.Delete(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, rec.Code)
	}

	if store.receivedID != noteID {
		t.Fatalf("expected delete note ID %s, got %s", noteID, store.receivedID)
	}
}

func TestNoteHandlerDeleteNotFound(t *testing.T) {
	noteID := uuid.MustParse("ed269d1f-92de-4b44-9ab0-aeffd504e5c2")
	handler := NewNoteHandler(&mockNoteStore{
		deleteNoteFn: func(_ context.Context, gotID uuid.UUID) error {
			return store.ErrNoteNotFound
		},
	})
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/notes/"+noteID.String(), nil)
	req.SetPathValue("id", noteID.String())
	rec := httptest.NewRecorder()

	handler.Delete(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}
