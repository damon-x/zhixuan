package gateway

import (
	stdctx "context"
	"fmt"
	"log"
	"sync"
	"time"

	"zhixuan/server/config"
	"zhixuan/server/context"
	"zhixuan/server/database"
	"zhixuan/server/llm"
	"zhixuan/server/memory"
	"zhixuan/server/model"

	"github.com/openai/openai-go"
)

// Agent manages a per-user message queue and processes messages sequentially.
type Agent struct {
	userID  uint
	mu      sync.Mutex
	queue   []*ChatRequest
	running bool
	stopCh  chan struct{} // non-nil while processMessage is running; close to cancel
}

func newAgent(userID uint) *Agent {
	return &Agent{userID: userID}
}

// Stop signals the current processMessage to cancel. Safe to call anytime.
func (a *Agent) Stop() {
	a.mu.Lock()
	ch := a.stopCh
	a.mu.Unlock()
	if ch != nil {
		select {
		case <-ch:
		default:
			close(ch)
		}
	}
}

// enqueue adds a request to the queue and starts the process loop if needed.
func (a *Agent) enqueue(req *ChatRequest) {
	a.mu.Lock()
	a.queue = append(a.queue, req)
	if !a.running {
		a.running = true
		go a.processLoop()
	}
	a.mu.Unlock()
}

// processLoop processes messages from the queue one by one.
// Exits when the queue is empty.
func (a *Agent) processLoop() {
	for {
		a.mu.Lock()
		if len(a.queue) == 0 {
			a.running = false
			a.mu.Unlock()
			return
		}
		req := a.queue[0]
		a.queue = a.queue[1:]

		// Create stop channel for this message
		stopCh := make(chan struct{})
		a.stopCh = stopCh
		a.mu.Unlock()

		ctx, cancel := stdctx.WithCancel(stdctx.Background())
		go func() {
			select {
			case <-stopCh:
				cancel()
			case <-ctx.Done():
			}
		}()

		resp := a.processMessage(ctx, req)

		cancel()
		a.mu.Lock()
		a.stopCh = nil
		a.mu.Unlock()

		// Write back result
		if req.ResultChan != nil {
			req.ResultChan <- resp
		}
		if req.QQReplyFn != nil && resp.Error == nil {
			req.QQReplyFn(resp.Content)
		}
		if req.WeChatReplyFn != nil && resp.Error == nil {
			req.WeChatReplyFn(resp.Content)
		}
	}
}

// processMessage handles a single chat request end-to-end.
func (a *Agent) processMessage(ctx stdctx.Context, req *ChatRequest) *ChatResponse {
	// 1. Load user
	var user model.User
	if err := database.DB.First(&user, req.UserID).Error; err != nil {
		return &ChatResponse{Error: fmt.Errorf("用户不存在")}
	}

	// 2. Determine session ID
	sessionID := req.SessionID
	var topicSince uint
	if (req.Source == SourceQQ || req.Source == SourceSchedule || req.Source == SourceWeChat) && sessionID == "" {
		session, err := ensureMainSession(user.ID)
		if err != nil {
			return &ChatResponse{Error: fmt.Errorf("获取主会话失败")}
		}
		sessionID = session.SessionID
		topicSince = session.TopicSince
	} else {
		// Load topic_since for web sessions
		var session model.Session
		if err := database.DB.Where("user_id = ? AND session_id = ?", user.ID, sessionID).First(&session).Error; err == nil {
			topicSince = session.TopicSince
		}
	}

	// 3. Update session: touch updated_at, fill title if empty
	var session model.Session
	if err := database.DB.Where("user_id = ? AND session_id = ?", user.ID, sessionID).First(&session).Error; err == nil {
		updates := map[string]interface{}{"updated_at": time.Now()}
		if session.Title == "" {
			runes := []rune(req.Content)
			if len(runes) > 30 {
				updates["title"] = string(runes[:30])
			} else {
				updates["title"] = req.Content
			}
		}
		database.DB.Model(&session).Updates(updates)
	}

	// 4. Load context from JSONL cache (must be before saving user msg to DB,
	//    otherwise GetOrLoad will load the user message from DB and cause duplication)
	msgs, err := context.GetOrLoad(user.ID, sessionID, topicSince)
	if err != nil {
		return &ChatResponse{Error: fmt.Errorf("加载上下文失败")}
	}

	// 5. Append user message to history + cache (after GetOrLoad to avoid duplication)
	userLLMMsg := llm.Message{Role: "user", Content: req.Content, CreatedAt: time.Now()}
	context.AppendHistory(sessionID, []llm.Message{userLLMMsg})
	context.AppendMessage(sessionID, userLLMMsg)
	msgs = append(msgs, userLLMMsg)

	// 7. Build system prompt
	systemPrompt := buildSystemPrompt(user.ID, req.KnowledgeBases)

	var messages []llm.Message
	messages = append(messages, llm.Message{
		Role:    "system",
		Content: systemPrompt,
	})
	messages = append(messages, msgs...)

	// 7.4 Recall memories via vector search, update session LRU window, snapshot for injection
	memSnapshot := memory.Recall(ctx, user.ID, sessionID, req.Content)

	// 7.5 Inject dynamic context as a fake tool call (not saved to cache)
	now := time.Now().Format("2006-01-02 15:04:05")
	messages = append(messages,
		llm.Message{
			Role: "assistant",
			ToolCalls: []llm.ToolCall{
				{ID: "ctx_inject", Name: "get_context", Arguments: "{}"},
			},
		},
		llm.Message{
			Role:       "tool",
			ToolCallID: "ctx_inject",
			Content:    buildContextPayload(user.ID, now, memSnapshot),
		},
	)

	// 8. Tool executor closure
	executeTool := func(name, argsJSON string) (string, error) {
		switch name {
		case "save_note":
			return executeSaveNote(&user, argsJSON)
		case "get_note_content":
			return executeGetNoteContent(&user, argsJSON)
		case "search_knowledge_base":
			return executeSearchKB(&user, argsJSON)
		case "web_search":
			return executeWebSearch(&user, argsJSON)
		case "list_knowledge_bases":
			return executeListKB(&user, argsJSON)
		case "list_notes":
			return executeListNotes(&user, argsJSON)
		case "create_todo":
			return executeCreateTodo(&user, argsJSON)
		case "list_todos":
			return executeListTodos(&user, argsJSON)
		case "update_todo":
			return executeUpdateTodo(&user, argsJSON)
		case "delete_todo":
			return executeDeleteTodo(&user, argsJSON)
		case "update_note":
			return executeUpdateNote(&user, argsJSON)
		case "delete_note":
			return executeDeleteNote(&user, argsJSON)
		case "list_plans":
			return executeListPlans(&user, argsJSON)
		case "get_plan":
			return executeGetPlan(&user, argsJSON)
		case "create_schedule":
			return executeCreateSchedule(&user, argsJSON)
		case "list_schedules":
			return executeListSchedules(&user, argsJSON)
		case "qq_notify":
			return executeQQNotify(&user, argsJSON)
		case "wechat_notify":
			return executeWeChatNotify(&user, argsJSON)
		case "describe_image":
			return executeDescribeImage(&user, argsJSON)
		case "read_file":
			return executeReadFile(&user, argsJSON)
		case "write_file":
			return executeWriteFile(&user, argsJSON)
		case "edit_file":
			return executeEditFile(&user, argsJSON)
		case "list_files":
			return executeListFiles(&user, argsJSON)
		case "list_tables":
			return executeListTables(&user, argsJSON)
		case "describe_table":
			return executeDescribeTable(&user, argsJSON)
		case "dump_schema":
			return executeDumpSchema(&user, argsJSON)
		case "query":
			return executeQuerySQL(&user, argsJSON)
		case "execute":
			return executeExecSQL(&user, argsJSON)
		case "load_skill":
			return executeLoadSkill(&user, argsJSON)
		case "search_memory":
			return executeSearchMemory(user.ID, argsJSON)
		default:
			return "", fmt.Errorf("unknown tool: %s", name)
		}
	}

	// 9. Assemble tool list
	tools := []openai.ChatCompletionToolParam{
		saveNoteTool, getNoteContentTool, searchKBTool,
		webSearchTool, listKBTool, listNotesTool, createTodoTool,
		qqNotifyTool, wechatNotifyTool, describeImageTool,
		readFileTool, writeFileTool, editFileTool, listFilesTool,
		updateNoteTool, deleteNoteTool,
		listTodosTool, updateTodoTool, deleteTodoTool,
		listPlansTool, getPlanTool,
		createScheduleTool, listSchedulesTool,
		dumpSchemaTool, queryTool, executeSQLTool, loadSkillTool,
		searchMemoryTool,
	}

	// 10. Call LLM with ReAct agent
	// 给 user/assistant 消息加时间戳前缀（仅发送给 LLM，不改原 Content）
	messages = applyTimestamps(messages)
	reply, intermediates, totalTokens, err := llm.ChatWithTools(ctx, messages, tools, executeTool)
	if err != nil {
		log.Printf("[agent] LLM 调用失败: %v", err)
		return &ChatResponse{Error: fmt.Errorf("AI 回复失败: %s", err.Error())}
	}

	// 11. Append intermediate messages + reply to history + cache
	replyAt := time.Now()
	replyMsg := llm.Message{Role: "assistant", Content: reply, CreatedAt: replyAt}
	for i := range intermediates {
		if intermediates[i].CreatedAt.IsZero() {
			intermediates[i].CreatedAt = replyAt
		}
	}
	cacheMsgs := append(intermediates, replyMsg)
	context.AppendHistory(sessionID, cacheMsgs)
	// 写路径：按 totalTokens 判断追加还是压缩
	context.AppendOrCompress(ctx, sessionID, cacheMsgs, totalTokens)

	// 记忆 agent 批处理触发：
	//   - checkpoint==0（首次/部署后首遇）：只处理当前这一轮，避免回放全部历史
	//   - 否则：自 checkpoint 以来累计的用户消息达阈值才拉起，整段新对话（已剔 tool call）喂给它
	// 提交即推进 checkpoint（best-effort，失败不回退）。
	checkpoint := memory.MemCheckpoint(sessionID)
	var batch []llm.Message
	if checkpoint == 0 {
		batch = []llm.Message{userLLMMsg, replyMsg}
	} else {
		batch = context.LoadMessagesSince(sessionID, checkpoint)
	}
	userMsgs := 0
	for _, m := range batch {
		if m.Role == "user" {
			userMsgs++
		}
	}
	if checkpoint == 0 || userMsgs >= config.MemoryBatchRounds {
		memory.AdvanceCheckpoint(sessionID, replyAt.UnixMilli())
		startMemoryAgent(user.ID, sessionID, batch)
	}

	return &ChatResponse{
		MessageID: uint(replyAt.UnixMilli()),
		Content:   reply,
		CreatedAt: replyAt,
	}
}

// applyTimestamps 返回一份新的消息切片：对 role 为 user/assistant 且 CreatedAt 非零的消息，
// 在 Content 前加上 "[YYYY-MM-DD HH:MM:SS] " 前缀（本地时区），让 LLM 感知每条消息的发生时刻。
// 原切片的 Content 不被修改，保证 DB / 快照 / 向量库等持久化数据不受影响。
// system/tool 消息不动（system 保 prompt cache，tool 的 Content 是 JSON 不能加前缀）；
// CreatedAt 为零值的消息（如压缩生成的摘要）跳过。
func applyTimestamps(msgs []llm.Message) []llm.Message {
	out := make([]llm.Message, len(msgs))
	copy(out, msgs)
	for i := range out {
		m := &out[i]
		if m.CreatedAt.IsZero() {
			continue
		}
		if m.Role != "user" && m.Role != "assistant" {
			continue
		}
		m.Content = "[" + m.CreatedAt.Format("2006-01-02 15:04:05") + "] " + m.Content
	}
	return out
}
