package memory

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	"zhixuan/server/config"
	"zhixuan/server/database"
	"zhixuan/server/embedding"
	"zhixuan/server/model"
	"zhixuan/server/search"
)

// Store 管理用户记忆的 SQLite 记录与向量索引。
// 每个用户一个独立的 sqvect 向量库，位于 <MemoryDir>/<userID>/vectors.db。
type Store struct {
	mu     sync.Mutex
	stores map[uint]*search.SqvectStore
	emb    *embedding.Client
}

var (
	instance *Store
	once     sync.Once
)

// Get 返回单例，Init 之后才可用。
func Get() *Store {
	return instance
}

// Init 初始化全局 Store，进程启动时调用一次。
func Init() {
	once.Do(func() {
		instance = &Store{
			stores: make(map[uint]*search.SqvectStore),
			emb:    embedding.New(),
		}
	})
}

func (s *Store) getOrOpen(userID uint) (*search.SqvectStore, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if vs, ok := s.stores[userID]; ok {
		return vs, nil
	}
	dir := filepath.Join(config.MemoryDir(), fmt.Sprintf("%d", userID))
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create memory dir: %w", err)
	}
	vs, err := search.NewSqvectStore(filepath.Join(dir, "vectors.db"))
	if err != nil {
		return nil, fmt.Errorf("open vector store: %w", err)
	}
	s.stores[userID] = vs
	return vs, nil
}

func vecID(id uint) string { return fmt.Sprintf("mem_%d", id) }

// Save 写入 SQLite 与向量索引。向量失败不阻塞 SQLite 写入。
func (s *Store) Save(ctx context.Context, userID uint, memType, content, tags, sessionID string) (*model.Memory, error) {
	mem := &model.Memory{
		UserID:    userID,
		Type:      memType,
		Content:   content,
		Tags:      tags,
		SessionID: sessionID,
	}
	if err := database.DB.Create(mem).Error; err != nil {
		return nil, fmt.Errorf("save memory: %w", err)
	}

	vectors, err := s.emb.Embed(ctx, []string{content})
	if err != nil {
		log.Printf("[memory] 向量化失败 memID=%d: %v", mem.ID, err)
		return mem, nil
	}
	vs, err := s.getOrOpen(userID)
	if err != nil {
		log.Printf("[memory] 打开向量库失败 userID=%d: %v", userID, err)
		return mem, nil
	}
	vid := vecID(mem.ID)
	if err := vs.CreateDoc(ctx, vid, vid); err != nil {
		log.Printf("[memory] CreateDoc 失败 memID=%d: %v", mem.ID, err)
	}
	if err := vs.AddWithDocID(ctx, vid, vid, vectors[0], content); err != nil {
		log.Printf("[memory] 向量写入失败 memID=%d: %v", mem.ID, err)
	}
	return mem, nil
}

// Update 更新记忆内容与向量。content 为空则只改 tags。
func (s *Store) Update(ctx context.Context, userID, id uint, content, tags string) error {
	var mem model.Memory
	if err := database.DB.First(&mem, id).Error; err != nil {
		return err
	}
	updates := map[string]interface{}{}
	if content != "" {
		updates["content"] = content
	}
	if tags != "" {
		updates["tags"] = tags
	}
	if len(updates) == 0 {
		return nil
	}
	if err := database.DB.Model(&mem).Updates(updates).Error; err != nil {
		return err
	}
	if content != "" {
		if vectors, err := s.emb.Embed(ctx, []string{content}); err == nil {
			if vs, err := s.getOrOpen(userID); err == nil {
				vid := vecID(id)
				vs.AddWithDocID(ctx, vid, vid, vectors[0], content)
			}
		}
	}
	return nil
}

// Search 语义召回 topK 条记忆，按相关度排序。
func (s *Store) Search(ctx context.Context, userID uint, query string, topK int) ([]model.Memory, error) {
	if topK <= 0 {
		topK = 5
	}
	vs, err := s.getOrOpen(userID)
	if err != nil {
		return nil, err
	}
	vectors, err := s.emb.Embed(ctx, []string{query})
	if err != nil {
		return nil, fmt.Errorf("embed query: %w", err)
	}
	results, err := vs.Search(ctx, vectors[0], topK)
	if err != nil {
		return nil, fmt.Errorf("vector search: %w", err)
	}
	if len(results) == 0 {
		return nil, nil
	}

	// 解析 ID 并从 SQLite 取详情
	idOrder := make([]uint, 0, len(results))
	for _, r := range results {
		var id uint
		if _, err := fmt.Sscanf(r.ID, "mem_%d", &id); err == nil {
			idOrder = append(idOrder, id)
		}
	}
	if len(idOrder) == 0 {
		return nil, nil
	}
	var mems []model.Memory
	if err := database.DB.Where("id IN ?", idOrder).Find(&mems).Error; err != nil {
		return nil, err
	}
	memMap := make(map[uint]model.Memory, len(mems))
	for _, m := range mems {
		memMap[m.ID] = m
	}
	out := make([]model.Memory, 0, len(idOrder))
	for _, id := range idOrder {
		if m, ok := memMap[id]; ok {
			out = append(out, m)
		}
	}
	return out, nil
}

// MemoryHit 带相关度分数（向量余弦相似度）的记忆命中。
type MemoryHit struct {
	Memory model.Memory
	Score  float32
}

// SearchWithScore 同 Search，但保留向量相关度分数，按分数降序返回。
// 供主 agent 召回时按阈值过滤使用；记忆 agent 查重继续用 Search。
func (s *Store) SearchWithScore(ctx context.Context, userID uint, query string, topK int) ([]MemoryHit, error) {
	if topK <= 0 {
		topK = 5
	}
	vs, err := s.getOrOpen(userID)
	if err != nil {
		return nil, err
	}
	vectors, err := s.emb.Embed(ctx, []string{query})
	if err != nil {
		return nil, fmt.Errorf("embed query: %w", err)
	}
	results, err := vs.Search(ctx, vectors[0], topK)
	if err != nil {
		return nil, fmt.Errorf("vector search: %w", err)
	}
	if len(results) == 0 {
		return nil, nil
	}

	type scored struct {
		id    uint
		score float32
	}
	order := make([]scored, 0, len(results))
	for _, r := range results {
		var id uint
		if _, err := fmt.Sscanf(r.ID, "mem_%d", &id); err == nil {
			order = append(order, scored{id: id, score: r.Score})
		}
	}
	if len(order) == 0 {
		return nil, nil
	}
	ids := make([]uint, 0, len(order))
	for _, o := range order {
		ids = append(ids, o.id)
	}
	var mems []model.Memory
	if err := database.DB.Where("id IN ?", ids).Find(&mems).Error; err != nil {
		return nil, err
	}
	memMap := make(map[uint]model.Memory, len(mems))
	for _, m := range mems {
		memMap[m.ID] = m
	}
	out := make([]MemoryHit, 0, len(order))
	for _, o := range order {
		if m, ok := memMap[o.id]; ok {
			out = append(out, MemoryHit{Memory: m, Score: o.score})
		}
	}
	return out, nil
}

// List 按时间倒序列出用户记忆。
func (s *Store) List(userID uint, limit int) ([]model.Memory, error) {
	if limit <= 0 {
		limit = 100
	}
	var mems []model.Memory
	err := database.DB.Where("user_id = ?", userID).Order("created_at desc").Limit(limit).Find(&mems).Error
	return mems, err
}

// Delete 删除 SQLite 记录与向量。
func (s *Store) Delete(ctx context.Context, userID, id uint) error {
	if err := database.DB.Delete(&model.Memory{}, id).Error; err != nil {
		return err
	}
	if vs, err := s.getOrOpen(userID); err == nil {
		vs.DeleteDoc(ctx, vecID(id))
	}
	return nil
}
