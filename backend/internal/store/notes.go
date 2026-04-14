package store

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"personal-knowledge-base/backend/internal/models"
)

const demoUserEmail = "demo@personal-knowledge-base.local"

var ErrNoteNotFound = errors.New("note not found")

const semanticSearchMaxDistance = 0.35

type NoteStore struct {
	db       noteDB
	embedder noteEmbedder
}

type SearchDiagnostics struct {
	Mode           string
	FallbackReason string
}

type SearchResultsWithDiagnostics struct {
	Results     []models.SearchResult
	Diagnostics SearchDiagnostics
}

type BackfillEmbeddingsReport struct {
	Scanned int
	Updated int
	Failed  int
}

type noteDB interface {
	queryRower
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

type CreateNoteInput struct {
	Title   string
	Content string
	Tags    []string
}

type UpdateNoteInput struct {
	Title   string
	Content string
	Tags    []string
}

type noteEmbedder interface {
	Embed(ctx context.Context, input string) ([]float32, error)
}

func NewNoteStore(db *pgxpool.Pool, embedders ...noteEmbedder) *NoteStore {
	store := &NoteStore{db: db}
	if len(embedders) > 0 {
		store.embedder = embedders[0]
	}

	return store
}

func (s *NoteStore) ListNotes(ctx context.Context) ([]models.Note, error) {
	return s.queryNotes(ctx, "")
}

func (s *NoteStore) SearchNotes(ctx context.Context, query string) ([]models.Note, error) {
	search, err := s.searchResultsWithDiagnostics(ctx, query)
	if err != nil {
		return nil, err
	}

	notes := make([]models.Note, 0, len(search.Results))
	for _, result := range search.Results {
		notes = append(notes, result.Note)
	}

	return notes, nil
}

func (s *NoteStore) SearchNoteResults(ctx context.Context, query string) ([]models.SearchResult, error) {
	search, err := s.searchResultsWithDiagnostics(ctx, query)
	if err != nil {
		return nil, err
	}

	return search.Results, nil
}

func (s *NoteStore) SearchNoteResultsWithDiagnostics(ctx context.Context, query string) (SearchResultsWithDiagnostics, error) {
	return s.searchResultsWithDiagnostics(ctx, query)
}

func (s *NoteStore) searchResultsWithDiagnostics(ctx context.Context, query string) (SearchResultsWithDiagnostics, error) {
	trimmedQuery := strings.TrimSpace(query)
	if trimmedQuery == "" {
		notes, err := s.ListNotes(ctx)
		if err != nil {
			return SearchResultsWithDiagnostics{}, err
		}

		results := make([]models.SearchResult, 0, len(notes))
		for _, note := range notes {
			results = append(results, models.SearchResult{
				Note:      note,
				MatchType: "list",
			})
		}

		return SearchResultsWithDiagnostics{
			Results: results,
			Diagnostics: SearchDiagnostics{
				Mode: "list",
			},
		}, nil
	}

	diagnostics := SearchDiagnostics{Mode: "text"}
	if s.embedder != nil {
		vector, err := s.embedder.Embed(ctx, trimmedQuery)
		if err != nil {
			diagnostics.FallbackReason = "embedding_failed"
		} else if len(vector) == 0 {
			diagnostics.FallbackReason = "embedding_empty"
		} else {
			results, semanticErr := s.querySemanticResults(ctx, vectorLiteral(vector), semanticSearchMaxDistance)
			if semanticErr == nil && len(results) > 0 {
				return SearchResultsWithDiagnostics{
					Results: results,
					Diagnostics: SearchDiagnostics{
						Mode: "semantic",
					},
				}, nil
			}

			if semanticErr != nil {
				diagnostics.FallbackReason = "semantic_query_failed"
			} else {
				diagnostics.FallbackReason = "semantic_no_match"
			}
		}
	} else {
		diagnostics.FallbackReason = "embedder_unconfigured"
	}

	notes, err := s.queryNotes(ctx, trimmedQuery)
	if err != nil {
		return SearchResultsWithDiagnostics{}, err
	}

	results := make([]models.SearchResult, 0, len(notes))
	for _, note := range notes {
		results = append(results, models.SearchResult{
			Note:      note,
			MatchType: "text",
		})
	}

	return SearchResultsWithDiagnostics{
		Results:     results,
		Diagnostics: diagnostics,
	}, nil
}

func (s *NoteStore) queryNotes(ctx context.Context, query string) ([]models.Note, error) {
	userID, err := s.ensureDemoUser(ctx)
	if err != nil {
		return nil, err
	}

	rows, err := s.db.Query(ctx, noteQuerySQL, userID, query, likePattern(query))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	notes := make([]models.Note, 0)
	for rows.Next() {
		var note models.Note
		if err := rows.Scan(
			&note.ID,
			&note.Title,
			&note.Content,
			&note.CreatedAt,
			&note.UpdatedAt,
			&note.Tags,
		); err != nil {
			return nil, err
		}
		notes = append(notes, note)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return notes, nil
}

func (s *NoteStore) querySemanticNotes(ctx context.Context, queryVector string, maxDistance float64) ([]models.Note, error) {
	results, err := s.querySemanticResults(ctx, queryVector, maxDistance)
	if err != nil {
		return nil, err
	}

	notes := make([]models.Note, 0, len(results))
	for _, result := range results {
		notes = append(notes, result.Note)
	}

	return notes, nil
}

func (s *NoteStore) querySemanticResults(ctx context.Context, queryVector string, maxDistance float64) ([]models.SearchResult, error) {
	userID, err := s.ensureDemoUser(ctx)
	if err != nil {
		return nil, err
	}

	rows, err := s.db.Query(ctx, semanticSearchSQL, userID, queryVector, maxDistance)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]models.SearchResult, 0)
	for rows.Next() {
		var note models.Note
		var distance float64
		if err := rows.Scan(
			&note.ID,
			&note.Title,
			&note.Content,
			&note.CreatedAt,
			&note.UpdatedAt,
			&note.Tags,
			&distance,
		); err != nil {
			return nil, err
		}
		similarity := 1 - distance
		results = append(results, models.SearchResult{
			Note:       note,
			MatchType:  "semantic",
			Similarity: &similarity,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

func (s *NoteStore) CreateNote(ctx context.Context, input CreateNoteInput) (models.Note, error) {
	note := models.Note{
		Title:   strings.TrimSpace(input.Title),
		Content: strings.TrimSpace(input.Content),
		Tags:    normalizeTags(input.Tags),
	}
	embeddingLiteral := s.embeddingLiteral(ctx, note)

	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return models.Note{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	userID, err := ensureDemoUserTx(ctx, tx)
	if err != nil {
		return models.Note{}, err
	}

	if err := tx.QueryRow(ctx, `
		INSERT INTO notes (user_id, title, content, embedding)
		VALUES ($1, $2, $3, $4::vector)
		RETURNING id, created_at, updated_at
	`, userID, note.Title, note.Content, embeddingLiteral).Scan(&note.ID, &note.CreatedAt, &note.UpdatedAt); err != nil {
		return models.Note{}, err
	}

	if err := syncNoteTags(ctx, tx, userID, note.ID, note.Tags); err != nil {
		return models.Note{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return models.Note{}, err
	}

	return note, nil
}

func (s *NoteStore) UpdateNote(ctx context.Context, noteID uuid.UUID, input UpdateNoteInput) (models.Note, error) {
	note := models.Note{
		ID:      noteID,
		Title:   strings.TrimSpace(input.Title),
		Content: strings.TrimSpace(input.Content),
		Tags:    normalizeTags(input.Tags),
	}
	embeddingLiteral := s.embeddingLiteral(ctx, note)

	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return models.Note{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	userID, err := ensureDemoUserTx(ctx, tx)
	if err != nil {
		return models.Note{}, err
	}

	err = tx.QueryRow(ctx, `
		UPDATE notes
		SET title = $3, content = $4, embedding = $5::vector, updated_at = NOW()
		WHERE id = $1 AND user_id = $2
		RETURNING created_at, updated_at
	`, note.ID, userID, note.Title, note.Content, embeddingLiteral).Scan(&note.CreatedAt, &note.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Note{}, ErrNoteNotFound
		}
		return models.Note{}, err
	}

	if err := syncNoteTags(ctx, tx, userID, note.ID, note.Tags); err != nil {
		return models.Note{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return models.Note{}, err
	}

	return note, nil
}

func (s *NoteStore) DeleteNote(ctx context.Context, noteID uuid.UUID) error {
	userID, err := s.ensureDemoUser(ctx)
	if err != nil {
		return err
	}

	commandTag, err := s.db.Exec(ctx, `
		DELETE FROM notes
		WHERE id = $1 AND user_id = $2
	`, noteID, userID)
	if err != nil {
		return err
	}

	if commandTag.RowsAffected() == 0 {
		return ErrNoteNotFound
	}

	return nil
}

func (s *NoteStore) BackfillMissingEmbeddings(ctx context.Context) (BackfillEmbeddingsReport, error) {
	if s.embedder == nil {
		return BackfillEmbeddingsReport{}, errors.New("embedder is not configured")
	}

	rows, err := s.db.Query(ctx, missingEmbeddingsSQL)
	if err != nil {
		return BackfillEmbeddingsReport{}, err
	}
	defer rows.Close()

	report := BackfillEmbeddingsReport{}
	for rows.Next() {
		var note models.Note
		if err := rows.Scan(&note.ID, &note.Title, &note.Content, &note.Tags); err != nil {
			return BackfillEmbeddingsReport{}, err
		}

		report.Scanned++
		vector, err := s.embedder.Embed(ctx, embeddingText(note))
		if err != nil || len(vector) == 0 {
			report.Failed++
			continue
		}

		if _, err := s.db.Exec(ctx, updateNoteEmbeddingSQL, note.ID, vectorLiteral(vector)); err != nil {
			report.Failed++
			continue
		}

		report.Updated++
	}

	if err := rows.Err(); err != nil {
		return BackfillEmbeddingsReport{}, err
	}

	return report, nil
}

func (s *NoteStore) ensureDemoUser(ctx context.Context) (uuid.UUID, error) {
	return ensureDemoUserTx(ctx, s.db)
}

type queryRower interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func ensureDemoUserTx(ctx context.Context, db queryRower) (uuid.UUID, error) {
	var userID uuid.UUID
	err := db.QueryRow(ctx, `
		INSERT INTO users (email, password_hash)
		VALUES ($1, $2)
		ON CONFLICT (email)
		DO UPDATE SET email = EXCLUDED.email
		RETURNING id
	`, demoUserEmail, "auth-not-enabled").Scan(&userID)

	return userID, err
}

func normalizeTags(tags []string) []string {
	if len(tags) == 0 {
		return []string{}
	}

	seen := make(map[string]struct{}, len(tags))
	normalized := make([]string, 0, len(tags))

	for _, tag := range tags {
		trimmed := strings.TrimSpace(tag)
		if trimmed == "" {
			continue
		}

		if _, ok := seen[trimmed]; ok {
			continue
		}

		seen[trimmed] = struct{}{}
		normalized = append(normalized, trimmed)
	}

	return normalized
}

func embeddingText(note models.Note) string {
	var builder strings.Builder
	builder.WriteString("Title: ")
	builder.WriteString(note.Title)
	builder.WriteString("\n")

	if len(note.Tags) > 0 {
		builder.WriteString("Tags: ")
		builder.WriteString(strings.Join(note.Tags, ", "))
		builder.WriteString("\n")
	}

	builder.WriteString("Content: ")
	builder.WriteString(note.Content)

	return builder.String()
}

func (s *NoteStore) embeddingLiteral(ctx context.Context, note models.Note) any {
	if s.embedder == nil {
		return nil
	}

	vector, err := s.embedder.Embed(ctx, embeddingText(note))
	if err != nil || len(vector) == 0 {
		return nil
	}

	return vectorLiteral(vector)
}

func vectorLiteral(vector []float32) string {
	parts := make([]string, 0, len(vector))
	for _, value := range vector {
		parts = append(parts, strconv.FormatFloat(float64(value), 'f', -1, 32))
	}

	return "[" + strings.Join(parts, ",") + "]"
}

func likePattern(query string) string {
	if query == "" {
		return "%"
	}

	return "%" + query + "%"
}

func syncNoteTags(ctx context.Context, tx pgx.Tx, userID uuid.UUID, noteID uuid.UUID, tags []string) error {
	if _, err := tx.Exec(ctx, `DELETE FROM note_tags WHERE note_id = $1`, noteID); err != nil {
		return err
	}

	for _, tagName := range tags {
		var tagID uuid.UUID
		if err := tx.QueryRow(ctx, `
			INSERT INTO tags (user_id, name)
			VALUES ($1, $2)
			ON CONFLICT (user_id, name)
			DO UPDATE SET name = EXCLUDED.name
			RETURNING id
		`, userID, tagName).Scan(&tagID); err != nil {
			return err
		}

		if _, err := tx.Exec(ctx, `
			INSERT INTO note_tags (note_id, tag_id)
			VALUES ($1, $2)
			ON CONFLICT (note_id, tag_id) DO NOTHING
		`, noteID, tagID); err != nil {
			return err
		}
	}

	return nil
}

const noteQuerySQL = `
	SELECT
		n.id,
		n.title,
		n.content,
		n.created_at,
		n.updated_at,
		COALESCE(
			array_agg(t.name ORDER BY t.name) FILTER (WHERE t.name IS NOT NULL),
			ARRAY[]::text[]
		) AS tags
	FROM notes n
	LEFT JOIN note_tags nt ON nt.note_id = n.id
	LEFT JOIN tags t ON t.id = nt.tag_id
	WHERE n.user_id = $1
		AND (
			$2 = ''
			OR n.title ILIKE $3
			OR n.content ILIKE $3
			OR t.name ILIKE $3
		)
	GROUP BY n.id, n.title, n.content, n.created_at, n.updated_at
	ORDER BY
		CASE
			WHEN $2 = '' THEN 0
			WHEN n.title ILIKE $3 THEN 0
			WHEN n.content ILIKE $3 THEN 1
			ELSE 2
		END,
		n.updated_at DESC,
		n.created_at DESC
`

const semanticSearchSQL = `
	WITH ranked_notes AS (
		SELECT
			n.id,
			n.title,
			n.content,
			n.created_at,
			n.updated_at,
			n.embedding <=> $2::vector AS distance
		FROM notes n
		WHERE n.user_id = $1
			AND n.embedding IS NOT NULL
			AND n.embedding <=> $2::vector <= $3
		ORDER BY distance ASC, n.updated_at DESC, n.created_at DESC
		LIMIT 20
	)
	SELECT
		rn.id,
		rn.title,
		rn.content,
		rn.created_at,
		rn.updated_at,
		COALESCE(
			array_agg(t.name ORDER BY t.name) FILTER (WHERE t.name IS NOT NULL),
			ARRAY[]::text[]
		) AS tags,
		rn.distance
	FROM ranked_notes rn
	LEFT JOIN note_tags nt ON nt.note_id = rn.id
	LEFT JOIN tags t ON t.id = nt.tag_id
	GROUP BY rn.id, rn.title, rn.content, rn.created_at, rn.updated_at, rn.distance
	ORDER BY rn.distance ASC, rn.updated_at DESC, rn.created_at DESC
`

const missingEmbeddingsSQL = `
	SELECT
		n.id,
		n.title,
		n.content,
		COALESCE(
			array_agg(t.name ORDER BY t.name) FILTER (WHERE t.name IS NOT NULL),
			ARRAY[]::text[]
		) AS tags
	FROM notes n
	LEFT JOIN note_tags nt ON nt.note_id = n.id
	LEFT JOIN tags t ON t.id = nt.tag_id
	WHERE n.embedding IS NULL
	GROUP BY n.id, n.title, n.content
	ORDER BY n.updated_at DESC, n.created_at DESC
`

const updateNoteEmbeddingSQL = `
	UPDATE notes
	SET embedding = $2::vector, updated_at = NOW()
	WHERE id = $1
`
