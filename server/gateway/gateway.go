package gateway

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"zhixuan/server/config"
	"zhixuan/server/database"
	"zhixuan/server/model"
	"zhixuan/server/qqbot"
	"zhixuan/server/wechat"
)

// Source indicates where the chat request originated from.
type Source int

const (
	SourceWeb Source = iota
	SourceQQ
	SourceSchedule
	SourceWeChat
)

// ChatRequest is the unified chat request.
type ChatRequest struct {
	UserID         uint
	SessionID      string               // web: from frontend; qq: uses main session (empty)
	Content        string
	WebSearch      bool
	KnowledgeBases []string
	Source         Source
	ResultChan     chan *ChatResponse   // web: blocking channel; qq: nil
	QQReplyFn      func(reply string)   // qq: callback to send message; web: nil
	WeChatReplyFn  func(reply string)   // wechat: callback to send message; web: nil
}

// ChatResponse is the result of processing a chat request.
type ChatResponse struct {
	MessageID uint
	Content   string
	CreatedAt time.Time
	Error     error
}

// QQManager tracks which users have active QQ chat listeners.
type QQManager struct {
	mu      sync.Mutex
	running map[uint]bool // key: userID
}

// WeChatManager tracks which users have active WeChat chat listeners.
type WeChatManager struct {
	mu      sync.Mutex
	running map[uint]bool   // key: userID
	cancel  map[uint]context.CancelFunc // key: userID
}

// Gateway is the global singleton.
type Gateway struct {
	mu      sync.Mutex
	agents  map[uint]*Agent // key: userID
	qqMgr   *QQManager
	wxMgr   *WeChatManager
}

var (
	gatewayOnce sync.Once
	instance    *Gateway
)

// Init initializes the global Gateway singleton.
func Init() {
	gatewayOnce.Do(func() {
		instance = &Gateway{
			agents: make(map[uint]*Agent),
			qqMgr:  &QQManager{running: make(map[uint]bool)},
			wxMgr:  &WeChatManager{running: make(map[uint]bool), cancel: make(map[uint]context.CancelFunc)},
		}
	})
}

// Get returns the global Gateway instance.
func Get() *Gateway {
	if instance == nil {
		panic("gateway not initialized")
	}
	return instance
}

// Chat enqueues a chat request for processing.
func (g *Gateway) Chat(req *ChatRequest) {
	g.mu.Lock()
	agent, ok := g.agents[req.UserID]
	if !ok {
		agent = newAgent(req.UserID)
		g.agents[req.UserID] = agent
	}
	g.mu.Unlock()

	// /stop command: cancel current processing directly, don't enqueue
	if (req.Source == SourceQQ || req.Source == SourceWeChat || req.Source == SourceSchedule) &&
		strings.TrimSpace(req.Content) == "/stop" {
		agent.Stop()
		if req.ResultChan != nil {
			req.ResultChan <- &ChatResponse{Content: "已停止当前回复"}
		}
		if req.QQReplyFn != nil {
			req.QQReplyFn("已停止当前回复")
		}
		if req.WeChatReplyFn != nil {
			req.WeChatReplyFn("已停止当前回复")
		}
		return
	}

	agent.enqueue(req)
}

// Stop signals the agent for the given user to stop current processing.
func (g *Gateway) Stop(userID uint) {
	g.mu.Lock()
	agent, ok := g.agents[userID]
	g.mu.Unlock()
	if ok {
		agent.Stop()
	}
}

// StartQQChat starts a QQ chat listener for the given user.
func (g *Gateway) StartQQChat(userID uint) error {
	g.qqMgr.mu.Lock()
	defer g.qqMgr.mu.Unlock()

	if g.qqMgr.running[userID] {
		return nil // already running
	}

	// Load user credentials
	var user model.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return fmt.Errorf("用户不存在")
	}
	if user.QQBotAppID == "" || user.QQBotAppSecret == "" || user.QQBotOpenID == "" {
		return fmt.Errorf("QQ Bot 未绑定")
	}

	// Start chat listener with onMessage callback
	onMessage := func(content string, imageRefs []string) {
		// 把图片标签拼到文字前面（与 web 端"图在前文字在后"一致）
		var b strings.Builder
		for _, ref := range imageRefs {
			b.WriteString("[image:图片:")
			b.WriteString(ref)
			b.WriteString("]")
		}
		b.WriteString(content)
		finalContent := b.String()

		g.Chat(&ChatRequest{
			UserID:    userID,
			SessionID: "",
			Content:   finalContent,
			Source:    SourceQQ,
			QQReplyFn: func(reply string) {
				sendQQReply(userID, user.QQBotAppID, user.QQBotAppSecret, user.QQBotOpenID, reply)
			},
		})
	}

	if err := qqbot.StartChatListener(userID, user.QQBotAppID, user.QQBotAppSecret, onMessage); err != nil {
		return fmt.Errorf("启动 QQ 对话监听失败: %w", err)
	}

	g.qqMgr.running[userID] = true
	log.Printf("[gateway] 用户 %d QQ 对话已启动", userID)
	return nil
}

// StopQQChat stops the QQ chat listener for the given user.
func (g *Gateway) StopQQChat(userID uint) {
	g.qqMgr.mu.Lock()
	defer g.qqMgr.mu.Unlock()

	if g.qqMgr.running[userID] {
		qqbot.StopChatListener(userID)
		delete(g.qqMgr.running, userID)
		log.Printf("[gateway] 用户 %d QQ 对话已停止", userID)
	}
}

// IsQQChatRunning returns whether the QQ chat listener is running for the given user.
func (g *Gateway) IsQQChatRunning(userID uint) bool {
	g.qqMgr.mu.Lock()
	defer g.qqMgr.mu.Unlock()
	return g.qqMgr.running[userID]
}

// RestoreQQListeners starts QQ chat listeners for all users that have it enabled.
func (g *Gateway) RestoreQQListeners() {
	var users []model.User
	database.DB.Where("qq_bot_chat_enabled = ? AND qq_bot_open_id != ''", true).Find(&users)
	for _, u := range users {
		if err := g.StartQQChat(u.ID); err != nil {
			log.Printf("[gateway] 恢复用户 %d QQ 对话失败: %v", u.ID, err)
		}
	}
}

// StartWeChatChat starts a WeChat chat listener for the given user.
func (g *Gateway) StartWeChatChat(userID uint) error {
	g.wxMgr.mu.Lock()
	defer g.wxMgr.mu.Unlock()

	if g.wxMgr.running[userID] {
		return nil // already running
	}

	statePath := wechat.StatePath(config.DataDir, userID)
	if !wechat.StateFileExists(statePath) {
		return fmt.Errorf("微信未绑定")
	}

	client := wechat.NewClient(statePath)
	if err := client.NotifyStart(); err != nil {
		return fmt.Errorf("微信通知上线失败: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	g.wxMgr.cancel[userID] = cancel
	g.wxMgr.running[userID] = true

	go g.runWeChatListener(ctx, userID, client)

	log.Printf("[gateway] 用户 %d 微信对话已启动", userID)
	return nil
}

func (g *Gateway) runWeChatListener(ctx context.Context, userID uint, client *wechat.Client) {
	defer func() {
		client.NotifyStop()
		client.SaveState()
		g.wxMgr.mu.Lock()
		delete(g.wxMgr.running, userID)
		delete(g.wxMgr.cancel, userID)
		g.wxMgr.mu.Unlock()
		log.Printf("[gateway] 用户 %d 微信对话已停止", userID)
	}()

	consecutiveFailures := 0
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		msgs, err := client.GetUpdates(ctx)
		if err != nil {
			consecutiveFailures++
			log.Printf("[gateway] 用户 %d 微信 GetUpdates 失败: %v (%d)", userID, err, consecutiveFailures)
			sleep := 2 * time.Second
			if consecutiveFailures >= 3 {
				sleep = 30 * time.Second
				consecutiveFailures = 0
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(sleep):
			}
			continue
		}

		consecutiveFailures = 0
		for _, msg := range msgs {
			if msg.MessageType != 1 { // only handle user messages
				continue
			}
			if strings.TrimSpace(msg.FromUserID) == "" {
				continue
			}
			text := wechat.ExtractText(msg)
			if strings.TrimSpace(text) == "" {
				continue
			}

			log.Printf("[gateway] 用户 %d 微信收到消息 from %s: %s", userID, msg.FromUserID, text)

			fromUserID := msg.FromUserID
			contextToken := msg.ContextToken

			g.Chat(&ChatRequest{
				UserID:    userID,
				SessionID: "",
				Content:   text,
				Source:    SourceWeChat,
				WeChatReplyFn: func(reply string) {
					if err := client.SendText(fromUserID, reply, contextToken); err != nil {
						log.Printf("[gateway] 微信 SendText 失败: %v", err)
					}
				},
			})
		}
	}
}

// StopWeChatChat stops the WeChat chat listener for the given user.
func (g *Gateway) StopWeChatChat(userID uint) {
	g.wxMgr.mu.Lock()
	defer g.wxMgr.mu.Unlock()

	if g.wxMgr.running[userID] {
		if cancel, ok := g.wxMgr.cancel[userID]; ok {
			cancel()
		}
		// goroutine will clean up running map
	}
}

// IsWeChatChatRunning returns whether the WeChat chat listener is running for the given user.
func (g *Gateway) IsWeChatChatRunning(userID uint) bool {
	g.wxMgr.mu.Lock()
	defer g.wxMgr.mu.Unlock()
	return g.wxMgr.running[userID]
}

// RestoreWeChatListeners starts WeChat chat listeners for all users that have it enabled.
func (g *Gateway) RestoreWeChatListeners() {
	var users []model.User
	database.DB.Where("wechat_chat_enabled = ? AND wechat_bound = ?", true, true).Find(&users)
	for _, u := range users {
		if err := g.StartWeChatChat(u.ID); err != nil {
			log.Printf("[gateway] 恢复用户 %d 微信对话失败: %v", u.ID, err)
		}
	}
}

// --- QQ 发送：文本/图片分段 ---

// qqImageTagRegex 匹配 [image:名称:路径]，捕获路径（第二个分组）。
var qqImageTagRegex = regexp.MustCompile(`\[image:[^\]]+:([^\]]+)\]`)

// qqSegment 是 reply 拆分后的一段。
type qqSegment struct {
	kind string // "text" 或 "image"
	text string
	ref  string
}

// sendQQReply 解析 reply 中的 [image:...] 标签，按顺序发送文本/图片。
// 纯文本（无标签）直接走原 SendMsg 路径，行为完全不变 —— 这是"文本不能失败"的核心保障。
func sendQQReply(userID uint, appID, secret, openid, reply string) {
	// 快速路径：无图片标签 → 原 SendMsg 路径
	if !qqImageTagRegex.MatchString(reply) {
		if err := qqbot.SendMsg(appID, secret, openid, reply); err != nil {
			log.Printf("[gateway] QQ SendMsg 失败: %v", err)
		}
		return
	}

	for _, seg := range parseQQReply(reply) {
		switch seg.kind {
		case "text":
			if strings.TrimSpace(seg.text) == "" {
				continue
			}
			if err := qqbot.SendMsg(appID, secret, openid, seg.text); err != nil {
				log.Printf("[gateway] QQ SendMsg 失败: %v", err)
			}
		case "image":
			localPath, ok := resolveImageRef(seg.ref, userID)
			if !ok {
				log.Printf("[gateway] QQ 图片路径无法解析 (user=%d): %s", userID, seg.ref)
				qqbot.SendMsg(appID, secret, openid, "(图片发送失败)")
				continue
			}
			log.Printf("[gateway] QQ 发送图片 (user=%d): %s → %s", userID, seg.ref, localPath)
			if err := qqbot.SendImage(appID, secret, openid, localPath); err != nil {
				log.Printf("[gateway] QQ SendImage 失败: %v", err)
				// 降级发文字，不阻断后续段
				if err2 := qqbot.SendMsg(appID, secret, openid, "(图片发送失败)"); err2 != nil {
					log.Printf("[gateway] QQ SendMsg 失败: %v", err2)
				}
			}
		}
	}
}

// parseQQReply 把 reply 拆成有序的文本/图片段。
func parseQQReply(reply string) []qqSegment {
	var segs []qqSegment
	idx := 0
	for _, m := range qqImageTagRegex.FindAllStringSubmatchIndex(reply, -1) {
		start, end := m[0], m[1]
		refStart, refEnd := m[2], m[3]
		if start > idx {
			segs = append(segs, qqSegment{kind: "text", text: reply[idx:start]})
		}
		segs = append(segs, qqSegment{kind: "image", ref: reply[refStart:refEnd]})
		idx = end
	}
	if idx < len(reply) {
		segs = append(segs, qqSegment{kind: "text", text: reply[idx:]})
	}
	return segs
}

// resolveImageRef 把 {source}@{relative_path} 解析为本地绝对路径，
// 支持来源与 handler.GetResource 一致：
//   - upload@    → config.UploadDir()/{rel}
//   - knowledge@ → config.KBDir()/{userID}/{rel}
func resolveImageRef(ref string, userID uint) (string, bool) {
	atIdx := strings.Index(ref, "@")
	if atIdx == -1 {
		return "", false
	}
	source := ref[:atIdx]
	relPath := ref[atIdx+1:]

	var baseDir string
	switch source {
	case "knowledge":
		baseDir = filepath.Join(config.KBDir(), fmt.Sprintf("%d", userID))
	case "upload":
		baseDir = config.UploadDir()
	default:
		return "", false
	}

	cleanRel := filepath.Clean(relPath)
	if strings.Contains(cleanRel, "..") {
		return "", false
	}
	return filepath.Join(baseDir, cleanRel), true
}
