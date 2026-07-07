package context

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"zhixuan/server/config"
	"zhixuan/server/llm"
)

// 历史记录按天分片，文件名格式：2006-01-02.jsonl
const dateLayout = "2006-01-02"

// 默认加载的对话轮数（1 轮 = 1 个 user + 1 个 assistant）
const defaultRounds = 20

// sessionHistoryDir 返回 <ChatHistoryDir>/<sessionID>/
func sessionHistoryDir(sessionID string) string {
	return filepath.Join(config.ChatHistoryDir(), sessionID)
}

// AppendHistory 将消息追加写入当天对应的历史文件（纯追加，不读取不裁剪）。
// 调用方未设置 CreatedAt 时自动补当前时间。
func AppendHistory(sessionID string, msgs []llm.Message) error {
	if len(msgs) == 0 {
		return nil
	}
	dir := sessionHistoryDir(sessionID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建历史目录失败: %w", err)
	}
	now := time.Now()
	date := now.Format(dateLayout)
	path := filepath.Join(dir, date+".jsonl")

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("打开历史文件失败: %w", err)
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	for i := range msgs {
		if msgs[i].CreatedAt.IsZero() {
			msgs[i].CreatedAt = now
		}
		line, err := json.Marshal(msgs[i])
		if err != nil {
			continue
		}
		w.Write(line)
		w.WriteByte('\n')
	}
	return w.Flush()
}

// readHistoryBackward 按日期倒序遍历历史文件，逐文件正向读取后从末尾向前访问。
// visit 返回 false 时停止遍历。文件不存在视为空。
func readHistoryBackward(sessionID string, visit func(msg llm.Message) bool) error {
	dir := sessionHistoryDir(sessionID)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".jsonl" {
			names = append(names, e.Name())
		}
	}
	// 文件名是日期，字典序倒序即时间倒序
	sort.Sort(sort.Reverse(sort.StringSlice(names)))

	for _, name := range names {
		f, err := os.Open(filepath.Join(dir, name))
		if err != nil {
			continue
		}
		var lines []llm.Message
		scanner := bufio.NewScanner(f)
		buf := make([]byte, 0, 64*1024)
		scanner.Buffer(buf, 10*1024*1024)
		for scanner.Scan() {
			var m llm.Message
			if err := json.Unmarshal(scanner.Bytes(), &m); err == nil {
				lines = append(lines, m)
			}
		}
		f.Close()
		for i := len(lines) - 1; i >= 0; i-- {
			if !visit(lines[i]) {
				return nil
			}
		}
	}
	return nil
}

// LoadRecentForContext 从 chat_history 加载最近 rounds 轮对话（含 tool call/result），
// 过滤 CreatedAt <= topicSince 的消息（topicSince 为毫秒时间戳，0 表示不过滤）。
// 结果按时间正序返回。
func LoadRecentForContext(sessionID string, topicSince uint, rounds int) []llm.Message {
	if rounds <= 0 {
		rounds = defaultRounds
	}
	var topicBoundary time.Time
	if topicSince > 0 {
		topicBoundary = time.Unix(0, int64(topicSince)*int64(time.Millisecond))
	}

	var collected []llm.Message
	userCount := 0
	readHistoryBackward(sessionID, func(msg llm.Message) bool {
		if !topicBoundary.IsZero() && !msg.CreatedAt.IsZero() && !msg.CreatedAt.After(topicBoundary) {
			return false
		}
		collected = append(collected, msg)
		if msg.Role == "user" {
			userCount++
			if userCount >= rounds {
				return false
			}
		}
		return true
	})

	for i, j := 0, len(collected)-1; i < j; i, j = i+1, j-1 {
		collected[i], collected[j] = collected[j], collected[i]
	}
	return collected
}

// HistoryItem 是给前端展示用的消息结构，JSON 与 model.Chat 兼容。
type HistoryItem struct {
	ID        uint      `json:"id"`
	UserID    uint      `json:"user_id"`
	SessionID string    `json:"session_id"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

// LoadRecentForDisplay 从 chat_history 加载最近 rounds 轮对话（仅 user/assistant 文本），
// 不含 tool call/result，用于 web 页面展示。结果按时间正序返回。
// beforeTs > 0 时只加载严格早于该毫秒时间戳的消息（游标分页）。
// hasMore 表示是否还有更早的可展示消息。
func LoadRecentForDisplay(sessionID string, rounds int, beforeTs int64) (items []HistoryItem, hasMore bool) {
	if rounds <= 0 {
		rounds = defaultRounds
	}
	var collected []HistoryItem
	userCount := 0
	enough := false
	readHistoryBackward(sessionID, func(msg llm.Message) bool {
		// 游标过滤：跳过 >= beforeTs 的消息（不含等于，避免边界重复）
		if beforeTs > 0 && !msg.CreatedAt.IsZero() && msg.CreatedAt.UnixMilli() >= beforeTs {
			return true
		}
		// 过滤 tool call
		if msg.Role == "user" {
			// user 消息
		} else if msg.Role == "assistant" && len(msg.ToolCalls) == 0 {
			// 纯文本 assistant 回复
		} else {
			return true
		}
		if enough {
			// 已收集够一轮分页，再看一条即可判定 hasMore
			hasMore = true
			return false
		}
		collected = append(collected, HistoryItem{
			ID:        uint(msg.CreatedAt.UnixMilli()),
			Role:      msg.Role,
			Content:   msg.Content,
			CreatedAt: msg.CreatedAt,
		})
		if msg.Role == "user" {
			userCount++
			if userCount >= rounds {
				enough = true
			}
		}
		return true
	})

	for i, j := 0, len(collected)-1; i < j; i, j = i+1, j-1 {
		collected[i], collected[j] = collected[j], collected[i]
	}
	return collected, hasMore
}

// LoadMessagesSince 返回 CreatedAt 严格晚于 sinceMs 的消息（时间正序），
// 已剔除 tool call / tool result，只保留 user 与纯文本 assistant 消息。
// 供记忆 agent 批处理：从上次 checkpoint 取整段新对话喂给它。
// readHistoryBackward 按时间倒序遍历，遇到 <= sinceMs 即可提前终止。
func LoadMessagesSince(sessionID string, sinceMs int64) []llm.Message {
	var collected []llm.Message
	readHistoryBackward(sessionID, func(msg llm.Message) bool {
		// 倒序遍历中遇到 <= sinceMs 的，更早的都不必再看
		if !msg.CreatedAt.IsZero() && msg.CreatedAt.UnixMilli() <= sinceMs {
			return false
		}
		// 剔除 tool call / tool result
		if msg.Role == "tool" || (msg.Role == "assistant" && len(msg.ToolCalls) > 0) {
			return true
		}
		collected = append(collected, msg)
		return true
	})
	for i, j := 0, len(collected)-1; i < j; i, j = i+1, j-1 {
		collected[i], collected[j] = collected[j], collected[i]
	}
	return collected
}

// DeleteSessionHistory 删除指定 session 的全部历史文件。
func DeleteSessionHistory(sessionID string) error {
	dir := sessionHistoryDir(sessionID)
	if err := os.RemoveAll(dir); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
