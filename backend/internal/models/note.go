package models

import (
	"time"

	"github.com/google/uuid"
)

type Note struct {
	ID        uuid.UUID `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	Tags      []string  `json:"tags"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type SearchResult struct {
	Note       Note     `json:"note"`
	MatchType  string   `json:"matchType"`
	Similarity *float64 `json:"similarity,omitempty"`
}
