package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"zhixuan/server/config"
	"zhixuan/server/database"
	"zhixuan/server/model"
)

const maxWindowSize = 10

// SessionState 是会话级运行时状态，单文件持久化。
// 文件：<SessionStateDir>/<sessionID>.json
//   - RecallIDs: 召回 LRU 窗口（队首=最近），含 id=0 空结果占位
//   - MemProcessedAt: 记忆 agent 已处理到的消息毫秒时间戳（0=从未处理）
//
// 文件丢失/损坏无所谓：召回窗口下一轮自然重建，checkpoint 退化为 0（下批从头取）。
type SessionState struct {
	sessionID      string `json:"-"`
	RecallIDs      []uint `json:"recall_ids"`
	MemProcessedAt int64  `json:"mem_processed_at"`
}

func statePath(sessionID string) string {
	return filepath.Join(config.SessionStateDir(), sessionID+".json")
}

// LoadState 读取会话状态；文件不存在或损坏（含旧的 []uint 格式）返回空状态。
func LoadState(sessionID string) *SessionState {
	st := &SessionState{sessionID: sessionID}
	if data, err := os.ReadFile(statePath(sessionID)); err == nil {
		_ = json.Unmarshal(data, st) // 解析失败/旧格式则保持空状态
	}
	return st
}

// Save 持久化整个状态文件（recall 窗口 + checkpoint 一起写，避免互相覆盖）。
func (s *SessionState) Save() error {
	if err := os.MkdirAll(config.SessionStateDir(), 0755); err != nil {
		return fmt.Errorf("mkdir session_state: %w", err)
	}
	data, err := json.Marshal(s)
	if err != nil {
		return fmt.Errorf("marshal session state: %w", err)
	}
	return os.WriteFile(statePath(s.sessionID), data, 0644)
}

// MemCheckpoint 返回记忆 agent 已处理到的消息毫秒时间戳（0=从未）。
func MemCheckpoint(sessionID string) int64 {
	return LoadState(sessionID).MemProcessedAt
}

// AdvanceCheckpoint 把记忆 agent 的处理游标推进到 ts，保留召回窗口。
// 供记忆 agent 任务提交时调用（best-effort：失败仅记日志，不回退）。
func AdvanceCheckpoint(sessionID string, ts int64) {
	st := LoadState(sessionID)
	st.MemProcessedAt = ts
	if err := st.Save(); err != nil {
		log.Printf("[memory] checkpoint save failed session=%s: %v", sessionID, err)
	}
}

// ResetRecallWindow 清空召回 LRU 窗口，但保留记忆 agent checkpoint。
// 开新话题时调用：旧话题的窗口记忆不再常驻，但写入游标跨话题保留，
// 让话题切换前未达阈值的尾巴能在下批被自然处理。
func ResetRecallWindow(sessionID string) {
	st := LoadState(sessionID)
	st.RecallIDs = nil
	if err := st.Save(); err != nil {
		log.Printf("[memory] reset recall window failed session=%s: %v", sessionID, err)
	}
}

// Touch 按 LRU 合并：传入的 IDs 提升到队首（保持传入顺序、真实 ID 去重），
// 已有但未命中的保持原顺序跟后，超过 maxWindowSize 踢队尾。
// id == 0 是"空结果占位"：不参与去重，逐个累计，用于在用户持续聊新话题时
// 把窗口尾部的旧记忆按 LRU 逐步淘汰。Snapshot 会过滤占位，不注入上下文。
func (s *SessionState) Touch(ids []uint) {
	newOrder := make([]uint, 0, maxWindowSize)
	seen := make(map[uint]bool, len(ids)+len(s.RecallIDs))
	appendID := func(id uint) {
		if id == 0 {
			newOrder = append(newOrder, id)
			return
		}
		if seen[id] {
			return
		}
		seen[id] = true
		newOrder = append(newOrder, id)
	}
	for _, id := range ids {
		appendID(id)
	}
	for _, id := range s.RecallIDs {
		appendID(id)
	}
	if len(newOrder) > maxWindowSize {
		newOrder = newOrder[:maxWindowSize]
	}
	s.RecallIDs = newOrder
}

// Snapshot 按窗口顺序查 DB 返回记忆详情，content 超 100 字截断。
// 查不到的（已删除）自动跳过；空结果占位（id=0）不查询、不输出。
func (s *SessionState) Snapshot() []model.Memory {
	realIDs := make([]uint, 0, len(s.RecallIDs))
	for _, id := range s.RecallIDs {
		if id != 0 {
			realIDs = append(realIDs, id)
		}
	}
	if len(realIDs) == 0 {
		return nil
	}
	var mems []model.Memory
	if err := database.DB.Where("id IN ?", realIDs).Find(&mems).Error; err != nil {
		log.Printf("[memory] snapshot query failed: %v", err)
		return nil
	}
	memMap := make(map[uint]model.Memory, len(mems))
	for _, m := range mems {
		memMap[m.ID] = m
	}
	out := make([]model.Memory, 0, len(realIDs))
	for _, id := range s.RecallIDs {
		if id == 0 {
			continue
		}
		m, ok := memMap[id]
		if !ok {
			continue
		}
		if runes := []rune(m.Content); len(runes) > 100 {
			m.Content = string(runes[:100]) + "..."
		}
		out = append(out, m)
	}
	return out
}

// Recall 用 query 向量召回 top K 记忆，按相关度阈值过滤后合并进会话 LRU 窗口并持久化，
// 返回窗口快照（≤maxWindowSize 条，content 截断 100 字）。
// 供主 agent 每轮调用，结果注入 get_context 伪 tool call。
// 过滤前的全部命中及 score 会打印日志，便于观测分布、调整阈值。
func Recall(ctx context.Context, userID uint, sessionID, query string) []model.Memory {
	const topK = 5
	hits, err := Get().SearchWithScore(ctx, userID, query, topK)
	if err != nil {
		log.Printf("[memory] recall search failed userID=%d: %v", userID, err)
	}
	threshold := config.MemoryRecallThreshold
	kept := make([]uint, 0, len(hits))
	for _, h := range hits {
		log.Printf("[memory] recall userID=%d query=%q id=%d score=%.4f type=%s content=%q",
			userID, query, h.Memory.ID, h.Score, h.Memory.Type, h.Memory.Content)
		if h.Score >= float32(threshold) {
			kept = append(kept, h.Memory.ID)
		}
	}
	log.Printf("[memory] recall userID=%d threshold=%.2f kept=%d/%d", userID, threshold, len(kept), len(hits))
	st := LoadState(sessionID)
	if len(kept) == 0 {
		// 本轮无命中：空结果也占一个位置，逐步把窗口尾部的旧记忆淘汰出去
		st.Touch([]uint{0})
	} else {
		st.Touch(kept)
	}
	if err := st.Save(); err != nil {
		log.Printf("[memory] state save failed session=%s: %v", sessionID, err)
	}
	return st.Snapshot()
}
