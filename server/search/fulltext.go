package search

import "context"

// FullTextStore is the interface for full-text search operations.
type FullTextStore interface {
	Index(ctx context.Context, docID string, text string) error
	Search(ctx context.Context, query string, topK int) ([]SearchResult, error)
	DeleteByPrefix(ctx context.Context, prefix string) error
	Close()
}
