package search

import "context"

// VectorStore is the interface for vector storage operations.
type VectorStore interface {
	Add(ctx context.Context, id string, vector []float32, content string) error
	Search(ctx context.Context, vector []float32, topK int) ([]SearchResult, error)
	DeleteByDocID(ctx context.Context, docID string) error
	Close()
}

// SearchResult represents a single search result.
type SearchResult struct {
	ID      string
	Content string
	Score   float32
	DocID   string
}
