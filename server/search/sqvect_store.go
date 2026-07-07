package search

import (
	"context"
	"fmt"

	"zhixuan/server/config"

	"github.com/liliang-cn/sqvect/pkg/core"
	"github.com/liliang-cn/sqvect/pkg/sqvect"
)

// SqvectStore implements VectorStore using sqvect.
type SqvectStore struct {
	db    *sqvect.DB
	store core.Store
}

// NewSqvectStore opens or creates a sqvect vector store at the given path.
func NewSqvectStore(path string) (*SqvectStore, error) {
	db, err := sqvect.Open(sqvect.Config{
		Path:       path,
		Dimensions: config.EmbeddingDimensions,
	})
	if err != nil {
		return nil, fmt.Errorf("open sqvect: %w", err)
	}
	return &SqvectStore{db: db, store: db.Vector()}, nil
}

func (s *SqvectStore) Add(ctx context.Context, id string, vector []float32, content string) error {
	emb := &core.Embedding{
		ID:      id,
		Vector:  vector,
		Content: content,
	}
	return s.store.Upsert(ctx, emb)
}

func (s *SqvectStore) AddWithDocID(ctx context.Context, id, docID string, vector []float32, content string) error {
	emb := &core.Embedding{
		ID:      id,
		DocID:   docID,
		Vector:  vector,
		Content: content,
	}
	return s.store.Upsert(ctx, emb)
}

// CreateDoc creates a document record (must be called before AddWithDocID).
func (s *SqvectStore) CreateDoc(ctx context.Context, docID, title string) error {
	return s.store.CreateDocument(ctx, &core.Document{
		ID:     docID,
		Title:  title,
	})
}

func (s *SqvectStore) Search(ctx context.Context, vector []float32, topK int) ([]SearchResult, error) {
	opts := core.SearchOptions{TopK: topK}
	results, err := s.store.Search(ctx, vector, opts)
	if err != nil {
		return nil, err
	}
	out := make([]SearchResult, len(results))
	for i, r := range results {
		out[i] = SearchResult{
			ID:      r.ID,
			Content: r.Content,
			Score:   float32(r.Score),
			DocID:   r.DocID,
		}
	}
	return out, nil
}

// DeleteDoc deletes a document and all its embeddings (cascade).
func (s *SqvectStore) DeleteDoc(ctx context.Context, docID string) error {
	return s.store.DeleteDocument(ctx, docID)
}

func (s *SqvectStore) Close() {
	s.db.Close()
}
