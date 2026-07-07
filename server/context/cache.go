package context

import (
	"bufio"
	stdctx "context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"zhixuan/server/config"
	"zhixuan/server/llm"
)

var (
	dirOnce sync.Once
	dirMu   sync.Mutex
	dirPath string
)

func ensureDir() string {
	dirOnce.Do(func() {
		dirPath = config.ContextCacheDir()
		os.MkdirAll(dirPath, 0755)
	})
	return dirPath
}

func filePath(sessionID string) string {
	return filepath.Join(ensureDir(), sessionID+".jsonl")
}

// jsonlEntry is the JSONL line format — reuses llm.Message directly.

func readJSONL(sessionID string) ([]llm.Message, error) {
	f, err := os.Open(filePath(sessionID))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var msgs []llm.Message
	scanner := bufio.NewScanner(f)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 10*1024*1024)
	for scanner.Scan() {
		var m llm.Message
		if err := json.Unmarshal(scanner.Bytes(), &m); err != nil {
			continue
		}
		msgs = append(msgs, m)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return msgs, nil
}

func writeJSONL(sessionID string, msgs []llm.Message) error {
	f, err := os.Create(filePath(sessionID))
	if err != nil {
		return err
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	for _, m := range msgs {
		line, _ := json.Marshal(m)
		w.Write(line)
		w.WriteByte('\n')
	}
	return w.Flush()
}

// GetOrLoad tries to read the JSONL cache; rebuilds from chat_history if missing/corrupt.
// 读路径：读到就直接返回，不剪裁、不压缩；读不到就从历史记录重建最近 defaultRounds 轮。
// userID is retained for API compatibility but not used (history is keyed by sessionID).
func GetOrLoad(userID uint, sessionID string, topicSince uint) ([]llm.Message, error) {
	dirMu.Lock()
	defer dirMu.Unlock()

	msgs, err := readJSONL(sessionID)
	if err == nil && len(msgs) > 0 {
		return msgs, nil
	}

	// context_cache 不存在或为空，从 chat_history 重建最近 defaultRounds 轮
	msgs = LoadRecentForContext(sessionID, topicSince, defaultRounds)
	if len(msgs) == 0 {
		return msgs, nil
	}
	_ = writeJSONL(sessionID, msgs)
	return msgs, nil
}

// AppendMessage appends a single message to the JSONL (pure append, no trimming).
func AppendMessage(sessionID string, msg llm.Message) {
	AppendMessages(sessionID, []llm.Message{msg})
}

// AppendMessages appends multiple messages to the JSONL (pure append, no trimming).
func AppendMessages(sessionID string, newMsgs []llm.Message) {
	if len(newMsgs) == 0 {
		return
	}
	dirMu.Lock()
	defer dirMu.Unlock()

	msgs, _ := readJSONL(sessionID)
	msgs = append(msgs, newMsgs...)
	_ = writeJSONL(sessionID, msgs)
}

// AppendOrCompress 写路径：追加新消息后按 totalTokens 判断是否压缩。
//   - 未超阈值：直接追加写
//   - 超阈值且 user 消息 > keepRecentUser：调用 LLM 压缩后覆盖写
//   - 超阈值但 user 消息 ≤ keepRecentUser（无可压缩内容）或压缩失败：退回追加写
func AppendOrCompress(ctx stdctx.Context, sessionID string, newMsgs []llm.Message, totalTokens int) {
	dirMu.Lock()
	defer dirMu.Unlock()

	msgs, _ := readJSONL(sessionID)
	msgs = append(msgs, newMsgs...)

	if totalTokens <= config.ContextCompressThreshold {
		_ = writeJSONL(sessionID, msgs)
		return
	}

	compressed, err := tryCompress(ctx, msgs)
	if err != nil {
		log.Printf("[context] 会话 %s 压缩失败，退回追加写: %v", sessionID, err)
		_ = writeJSONL(sessionID, msgs)
		return
	}
	if compressed == nil {
		// 无可压缩内容（user 消息不足），跳过压缩
		_ = writeJSONL(sessionID, msgs)
		return
	}
	log.Printf("[context] 会话 %s 上下文压缩：%d 条 → %d 条（total_tokens=%d）",
		sessionID, len(msgs), len(compressed), totalTokens)
	_ = writeJSONL(sessionID, compressed)
}

// keepRecentUser 是压缩时末尾原样保留的 user 消息条数。
const keepRecentUser = 5

// tryCompress 把 msgs 按最后 keepRecentUser 条 user 消息切分：
//   - 后段（最后 keepRecentUser 条 user 及其间全部消息）原样保留，不允许任何处理
//   - 前段压成 ≤ summaryMaxChars 的摘要，作为 role=user 置于头部
//
// user 消息总数 ≤ keepRecentUser 时返回 (nil, nil)，表示无可压缩内容。
func tryCompress(ctx stdctx.Context, msgs []llm.Message) ([]llm.Message, error) {
	if countUserMessages(msgs) <= keepRecentUser {
		return nil, nil
	}
	splitIdx := findLastNUserBoundary(msgs, keepRecentUser)
	if splitIdx <= 0 {
		return nil, nil
	}
	front := msgs[:splitIdx]
	tail := msgs[splitIdx:]

	summary, err := summarize(ctx, front)
	if err != nil {
		return nil, err
	}
	result := make([]llm.Message, 0, len(tail)+1)
	result = append(result, llm.Message{Role: "user", Content: summary})
	result = append(result, tail...)
	return result, nil
}

func countUserMessages(msgs []llm.Message) int {
	n := 0
	for _, m := range msgs {
		if m.Role == "user" {
			n++
		}
	}
	return n
}

// findLastNUserBoundary 返回最后 n 条 role=user 消息中最早那条的下标。
// 即 msgs[idx:] 恰好包含最后 n 条 user 消息（及其间的全部消息）。
// user 总数 < n 时返回 0。
func findLastNUserBoundary(msgs []llm.Message, n int) int {
	count := 0
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == "user" {
			count++
			if count == n {
				return i
			}
		}
	}
	return 0
}

const summarizeSystemPrompt = `你是对话摘要助手。请把下面这段较早的对话压缩成一段简洁的摘要，供后续对话参考。

要求：
- 用第三人称陈述句
- 保留：用户的意图与需求、已确认的重要事实、关键决策、未完成的待办、对后续对话有用的上下文
- 丢弃：寒暄、已过时的细节、冗长的工具原始输出（只保留结论）
- 不要添加对话中不存在的信息
- 不超过 %d 字`

func summarize(ctx stdctx.Context, msgs []llm.Message) (string, error) {
	maxChars := config.ContextSummaryMaxChars
	req := []llm.Message{
		{Role: "system", Content: fmt.Sprintf(summarizeSystemPrompt, maxChars)},
		{Role: "user", Content: renderTranscript(msgs)},
	}
	summary, err := llm.Chat(ctx, req)
	if err != nil {
		return "", err
	}
	if maxChars > 0 {
		if r := []rune(summary); len(r) > maxChars {
			summary = string(r[:maxChars])
		}
	}
	return summary, nil
}

// renderTranscript 把消息序列渲染成可读文本，喂给摘要 LLM。
func renderTranscript(msgs []llm.Message) string {
	var b strings.Builder
	for _, m := range msgs {
		switch m.Role {
		case "user":
			b.WriteString("用户：")
			b.WriteString(m.Content)
		case "assistant":
			if len(m.ToolCalls) > 0 {
				b.WriteString("助手（调用工具）：")
				for _, tc := range m.ToolCalls {
					b.WriteString(tc.Name)
					b.WriteString("(")
					b.WriteString(tc.Arguments)
					b.WriteString(") ")
				}
				if m.Content != "" {
					b.WriteString(" -> ")
					b.WriteString(m.Content)
				}
			} else {
				b.WriteString("助手：")
				b.WriteString(m.Content)
			}
		case "tool":
			b.WriteString("工具结果：")
			b.WriteString(m.Content)
		case "system":
			b.WriteString("（前情摘要）")
			b.WriteString(m.Content)
		}
		b.WriteString("\n")
	}
	return b.String()
}

// Invalidate deletes the JSONL cache file for the session.
func Invalidate(sessionID string) {
	dirMu.Lock()
	defer dirMu.Unlock()

	os.Remove(filePath(sessionID))
}
