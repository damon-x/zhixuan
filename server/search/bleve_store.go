package search

import (
	"context"
	"fmt"
	"os"
	"strings"

	"zhixuan/server/config"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/analysis"
	"github.com/blevesearch/bleve/v2/registry"
	"github.com/wangbin/jiebago"
)

func init() {
	registry.RegisterTokenizer("jieba", newJiebaTokenizer)
	registry.RegisterAnalyzer("jieba", func(cfg map[string]interface{}, cache *registry.Cache) (analysis.Analyzer, error) {
		tok, err := cache.TokenizerNamed("jieba")
		if err != nil {
			return nil, err
		}
		lower, err := cache.TokenFilterNamed("to_lower")
		if err != nil {
			return nil, err
		}
		return &jiebaAnalyzer{Tokenizer: tok, Filters: []analysis.TokenFilter{lower}}, nil
	})
}

type jiebaTokenizer struct {
	seg *jiebago.Segmenter
}

func newJiebaTokenizer(map[string]interface{}, *registry.Cache) (analysis.Tokenizer, error) {
	var seg jiebago.Segmenter
	if err := seg.LoadDictionary(config.DictPath()); err != nil {
		return nil, fmt.Errorf("load jieba dict: %w", err)
	}
	return &jiebaTokenizer{seg: &seg}, nil
}

func (t *jiebaTokenizer) Tokenize(input []byte) analysis.TokenStream {
	var tokens analysis.TokenStream
	pos := 1
	for word := range t.seg.Cut(string(input), true) {
		word = strings.TrimSpace(word)
		if word == "" {
			continue
		}
		token := analysis.Token{Term: []byte(word), Position: pos, Type: analysis.AlphaNumeric}
		pos++
		tokens = append(tokens, &token)
	}
	return tokens
}

type jiebaAnalyzer struct {
	Tokenizer analysis.Tokenizer
	Filters   []analysis.TokenFilter
}

func (a *jiebaAnalyzer) Analyze(input []byte) analysis.TokenStream {
	tokens := a.Tokenizer.Tokenize(input)
	for _, f := range a.Filters {
		tokens = f.Filter(tokens)
	}
	return tokens
}

// bleveDoc is the document structure stored in bleve.
type bleveDoc struct {
	DocID   string `json:"doc_id"`
	Content string `json:"content"`
}

// BleveStore implements FullTextStore using bleve with jieba.
type BleveStore struct {
	index bleve.Index
}

// NewBleveStore opens or creates a bleve index at the given path.
func NewBleveStore(path string) (*BleveStore, error) {
	mapping := bleve.NewIndexMapping()
	mapping.DefaultAnalyzer = "jieba"
	mapping.StoreDynamic = true

	var idx bleve.Index
	var err error
	if _, statErr := os.Stat(path); os.IsNotExist(statErr) {
		idx, err = bleve.New(path, mapping)
	} else {
		idx, err = bleve.Open(path)
	}
	if err != nil {
		return nil, fmt.Errorf("open bleve: %w", err)
	}
	return &BleveStore{index: idx}, nil
}

func (b *BleveStore) Index(ctx context.Context, docID string, text string) error {
	return b.index.Index(docID, bleveDoc{DocID: docID, Content: text})
}

func (b *BleveStore) Search(ctx context.Context, query string, topK int) ([]SearchResult, error) {
	q := bleve.NewMatchQuery(query)
	req := bleve.NewSearchRequest(q)
	req.Size = topK
	req.Fields = []string{"doc_id", "content"}

	resp, err := b.index.Search(req)
	if err != nil {
		return nil, err
	}
	out := make([]SearchResult, 0, len(resp.Hits))
	for _, hit := range resp.Hits {
		content := ""
		if hit.Fields != nil {
			if c, ok := hit.Fields["content"]; ok && c != nil {
				content = fmt.Sprintf("%v", c)
			}
		}
		out = append(out, SearchResult{
			ID:      hit.ID,
			Content: content,
			Score:   float32(hit.Score),
		})
	}
	return out, nil
}

func (b *BleveStore) DeleteByPrefix(ctx context.Context, prefix string) error {
	q := bleve.NewPrefixQuery(prefix)
	req := bleve.NewSearchRequest(q)
	req.Size = 1000
	req.Fields = []string{"doc_id"}

	resp, err := b.index.Search(req)
	if err != nil {
		return err
	}
	for _, hit := range resp.Hits {
		if err := b.index.Delete(hit.ID); err != nil {
			return err
		}
	}
	return nil
}

func (b *BleveStore) Close() {
	b.index.Close()
}
