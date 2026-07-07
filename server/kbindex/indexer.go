package kbindex

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"zhixuan/server/config"
	"zhixuan/server/chunker"
	"zhixuan/server/embedding"
	"zhixuan/server/rerank"
	"zhixuan/server/search"
)

// SearchResult from a knowledge base search.
type SearchResult struct {
	Content string  `json:"content"`
	Score   float32 `json:"score"`
	DocID   string  `json:"doc_id"`
}

type kbStores struct {
	vector   *search.SqvectStore
	fulltext *search.BleveStore
}

// Indexer manages per-KB index instances. Singleton.
type Indexer struct {
	mu      sync.Mutex
	stores  map[string]*kbStores // key: "userID/kbName"
	emb     *embedding.Client
	rerank  *rerank.Client
}

var (
	instance *Indexer
	once     sync.Once
)

// Get returns the singleton Indexer.
func Get() *Indexer {
	return instance
}

// Init initializes the global Indexer. Call once after database init.
func Init() {
	once.Do(func() {
		instance = &Indexer{
			stores: make(map[string]*kbStores),
			emb:    embedding.New(),
			rerank: rerank.New(),
		}
	})
}

func kbKey(userID uint, kbName string) string {
	return fmt.Sprintf("%d/%s", userID, kbName)
}

func indexPath(userID uint, kbName string) string {
	return filepath.Join(config.KBDir(), fmt.Sprintf("%d", userID), kbName, ".index")
}

func (idx *Indexer) getOrOpen(userID uint, kbName string) (*kbStores, error) {
	key := kbKey(userID, kbName)
	idx.mu.Lock()
	defer idx.mu.Unlock()

	if s, ok := idx.stores[key]; ok {
		return s, nil
	}

	ip := indexPath(userID, kbName)
	vectPath := filepath.Join(ip, "vectors.db")
	blevePath := filepath.Join(ip, "fulltext.bleve")

	os.MkdirAll(ip, 0755)

	vs, err := search.NewSqvectStore(vectPath)
	if err != nil {
		return nil, fmt.Errorf("open vector store: %w", err)
	}
	fs, err := search.NewBleveStore(blevePath)
	if err != nil {
		vs.Close()
		return nil, fmt.Errorf("open fulltext store: %w", err)
	}

	s := &kbStores{vector: vs, fulltext: fs}
	idx.stores[key] = s
	return s, nil
}

// IndexDocument reads a file, chunks it, embeds, and stores in vector + fulltext indexes.
func (idx *Indexer) IndexDocument(ctx context.Context, userID uint, kbName, filename string) error {
	stores, err := idx.getOrOpen(userID, kbName)
	if err != nil {
		return err
	}

	filePath := filepath.Join(config.KBDir(), fmt.Sprintf("%d", userID), kbName, filename)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}
	text := string(data)

	chunks := chunker.Split(text)
	if len(chunks) == 0 {
		return nil
	}

	log.Printf("[kbindex] 切片完成 %s/%s: %d 个切片", kbName, filename, len(chunks))

	// Embed all chunks
	texts := make([]string, len(chunks))
	for i, c := range chunks {
		texts[i] = c.Text
	}
	vectors, err := idx.emb.Embed(ctx, texts)
	if err != nil {
		return fmt.Errorf("embed: %w", err)
	}

	log.Printf("[kbindex] 向量化完成 %s/%s: %d 个向量", kbName, filename, len(vectors))

	// Store in vector and fulltext indexes
	docID := filename
	if err := stores.vector.CreateDoc(ctx, docID, filename); err != nil {
		return fmt.Errorf("create doc: %w", err)
	}
	for i, chunk := range chunks {
		id := fmt.Sprintf("%s_%d", filename, chunk.Index)
		if err := stores.vector.AddWithDocID(ctx, id, docID, vectors[i], chunk.Text); err != nil {
			return fmt.Errorf("add vector %s: %w", id, err)
		}
		fulltextID := fmt.Sprintf("ft_%s_%d", filename, chunk.Index)
		if err := stores.fulltext.Index(ctx, fulltextID, chunk.Text); err != nil {
			return fmt.Errorf("index fulltext %s: %w", fulltextID, err)
		}
	}

	log.Printf("[kbindex] 向量索引+全文索引写入完成 %s/%s: %d 条", kbName, filename, len(chunks))
	return nil
}

// IndexImage indexes OCR text from an image as a single chunk with source marker.
func (idx *Indexer) IndexImage(ctx context.Context, userID uint, kbName, filename, ocrText string) error {
	stores, err := idx.getOrOpen(userID, kbName)
	if err != nil {
		return err
	}

	content := fmt.Sprintf("[source:img:knowledge@%s/%s]\n%s", kbName, filename, ocrText)

	log.Printf("[kbindex] 图片索引开始 %s/%s", kbName, filename)

	vectors, err := idx.emb.Embed(ctx, []string{content})
	if err != nil {
		return fmt.Errorf("embed image: %w", err)
	}

	docID := filename
	if err := stores.vector.CreateDoc(ctx, docID, filename); err != nil {
		return fmt.Errorf("create doc: %w", err)
	}

	id := fmt.Sprintf("%s_img", filename)
	if err := stores.vector.AddWithDocID(ctx, id, docID, vectors[0], content); err != nil {
		return fmt.Errorf("add vector %s: %w", id, err)
	}
	fulltextID := fmt.Sprintf("ft_%s_img", filename)
	if err := stores.fulltext.Index(ctx, fulltextID, content); err != nil {
		return fmt.Errorf("index fulltext %s: %w", fulltextID, err)
	}

	log.Printf("[kbindex] 图片索引完成 %s/%s", kbName, filename)
	return nil
}

// DeleteDocument removes a document from vector and fulltext indexes.
func (idx *Indexer) DeleteDocument(ctx context.Context, userID uint, kbName, filename string) error {
	stores, err := idx.getOrOpen(userID, kbName)
	if err != nil {
		return err
	}

	log.Printf("[kbindex] 删除索引开始 %s/%s", kbName, filename)

	if err := stores.vector.DeleteDoc(ctx, filename); err != nil {
		return fmt.Errorf("delete doc: %w", err)
	}
	log.Printf("[kbindex] 向量索引删除完成 %s/%s", kbName, filename)

	prefix := fmt.Sprintf("ft_%s_", filename)
	if err := stores.fulltext.DeleteByPrefix(ctx, prefix); err != nil {
		return fmt.Errorf("delete fulltext: %w", err)
	}
	log.Printf("[kbindex] 全文索引删除完成 %s/%s", kbName, filename)
	return nil
}

// DeleteKB closes stores and removes the .index directory.
func (idx *Indexer) DeleteKB(userID uint, kbName string) {
	key := kbKey(userID, kbName)
	idx.mu.Lock()
	s, ok := idx.stores[key]
	if ok {
		delete(idx.stores, key)
	}
	idx.mu.Unlock()

	if s != nil {
		s.vector.Close()
		s.fulltext.Close()
	}

	ip := indexPath(userID, kbName)
	os.RemoveAll(ip)
	log.Printf("[kbindex] 知识库索引已删除 user=%d kb=%s", userID, kbName)
}

// Search performs vector top5 + fulltext top5, then reranks.
func (idx *Indexer) Search(ctx context.Context, userID uint, kbName, query string) ([]SearchResult, error) {
	log.Printf("[kbindex] 搜索开始 kb=%s query=%q", kbName, query)

	stores, err := idx.getOrOpen(userID, kbName)
	if err != nil {
		return nil, err
	}

	// Embed query
	vectors, err := idx.emb.Embed(ctx, []string{query})
	if err != nil {
		return nil, fmt.Errorf("embed query: %w", err)
	}
	queryVec := vectors[0]

	// Vector search top5
	vectResults, err := stores.vector.Search(ctx, queryVec, 5)
	if err != nil {
		return nil, fmt.Errorf("vector search: %w", err)
	}
	log.Printf("[kbindex] 向量搜索完成: %d 条结果", len(vectResults))

	// Fulltext search top5
	ftResults, err := stores.fulltext.Search(ctx, query, 5)
	if err != nil {
		return nil, fmt.Errorf("fulltext search: %w", err)
	}
	log.Printf("[kbindex] 全文搜索完成: %d 条结果", len(ftResults))

	// Merge and deduplicate by content
	seen := make(map[string]bool)
	var docs []string
	var allResults []search.SearchResult
	for _, r := range vectResults {
		if !seen[r.Content] {
			seen[r.Content] = true
			docs = append(docs, r.Content)
			allResults = append(allResults, r)
		}
	}
	for _, r := range ftResults {
		if !seen[r.Content] {
			seen[r.Content] = true
			docs = append(docs, r.Content)
			allResults = append(allResults, r)
		}
	}

	if len(docs) == 0 {
		log.Printf("[kbindex] 搜索无结果")
		return nil, nil
	}

	// Rerank
	reranked, err := idx.rerank.Rerank(ctx, query, docs, 5)
	if err != nil {
		// Fallback: return vector results
		log.Printf("[kbindex] rerank 失败，回退向量搜索结果: %v", err)
		out := make([]SearchResult, 0, len(vectResults))
		for _, r := range vectResults {
			out = append(out, SearchResult{Content: r.Content, Score: r.Score, DocID: r.DocID})
		}
		return out, nil
	}

	log.Printf("[kbindex] rerank 完成: %d 条结果", len(reranked))

	out := make([]SearchResult, 0, len(reranked))
	for _, r := range reranked {
		out = append(out, SearchResult{
			Content: r.Text,
			Score:   float32(r.RelevanceScore),
		})
	}
	return out, nil
}

// ListKBNames lists knowledge base names for a user (used by chat to build system prompt).
func (idx *Indexer) ListKBNames(userID uint) []string {
	userDir := filepath.Join(config.KBDir(), fmt.Sprintf("%d", userID))
	entries, err := os.ReadDir(userDir)
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
			names = append(names, e.Name())
		}
	}
	return names
}
