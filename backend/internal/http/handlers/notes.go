package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/google/uuid"

	"personal-knowledge-base/backend/internal/models"
	"personal-knowledge-base/backend/internal/store"
)

type noteStore interface {
	ListNotes(ctx context.Context) ([]models.Note, error)
	SearchNoteResults(ctx context.Context, query string) ([]models.SearchResult, error)
	SearchNoteResultsWithDiagnostics(ctx context.Context, query string) (store.SearchResultsWithDiagnostics, error)
	CreateNote(ctx context.Context, input store.CreateNoteInput) (models.Note, error)
	UpdateNote(ctx context.Context, noteID uuid.UUID, input store.UpdateNoteInput) (models.Note, error)
	DeleteNote(ctx context.Context, noteID uuid.UUID) error
}

type NoteHandler struct {
	store noteStore
}

type createNoteRequest struct {
	Title   string   `json:"title"`
	Content string   `json:"content"`
	Tags    []string `json:"tags"`
}

type updateNoteRequest struct {
	Title   string   `json:"title"`
	Content string   `json:"content"`
	Tags    []string `json:"tags"`
}

func NewNoteHandler(store noteStore) *NoteHandler {
	return &NoteHandler{store: store}
}

func (h *NoteHandler) List(w http.ResponseWriter, r *http.Request) {
	notes, err := h.store.ListNotes(r.Context())
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "unable to load notes"})
		return
	}

	respondJSON(w, http.StatusOK, notes)
}

func (h *NoteHandler) Search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	search, err := h.store.SearchNoteResultsWithDiagnostics(r.Context(), query)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "unable to search notes"})
		return
	}

	log.Printf(
		`search query=%q mode=%s fallback_reason=%s result_count=%d`,
		query,
		search.Diagnostics.Mode,
		search.Diagnostics.FallbackReason,
		len(search.Results),
	)

	respondJSON(w, http.StatusOK, search.Results)
}

func (h *NoteHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createNoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON payload"})
		return
	}

	if req.Title == "" || req.Content == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "title and content are required"})
		return
	}

	note, err := h.store.CreateNote(r.Context(), store.CreateNoteInput{
		Title:   req.Title,
		Content: req.Content,
		Tags:    req.Tags,
	})
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "unable to create note"})
		return
	}

	respondJSON(w, http.StatusCreated, note)
}

func (h *NoteHandler) Update(w http.ResponseWriter, r *http.Request) {
	noteID, err := parseNoteID(r)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid note ID"})
		return
	}

	var req updateNoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON payload"})
		return
	}

	if req.Title == "" || req.Content == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "title and content are required"})
		return
	}

	note, err := h.store.UpdateNote(r.Context(), noteID, store.UpdateNoteInput{
		Title:   req.Title,
		Content: req.Content,
		Tags:    req.Tags,
	})
	if err != nil {
		if errors.Is(err, store.ErrNoteNotFound) {
			respondJSON(w, http.StatusNotFound, map[string]string{"error": "note not found"})
			return
		}

		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "unable to update note"})
		return
	}

	respondJSON(w, http.StatusOK, note)
}

func (h *NoteHandler) Delete(w http.ResponseWriter, r *http.Request) {
	noteID, err := parseNoteID(r)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid note ID"})
		return
	}

	if err := h.store.DeleteNote(r.Context(), noteID); err != nil {
		if errors.Is(err, store.ErrNoteNotFound) {
			respondJSON(w, http.StatusNotFound, map[string]string{"error": "note not found"})
			return
		}

		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "unable to delete note"})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func parseNoteID(r *http.Request) (uuid.UUID, error) {
	return uuid.Parse(r.PathValue("id"))
}
