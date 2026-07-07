package gateway

import (
	stdctx "context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"zhixuan/server/config"
	"zhixuan/server/database"
	"zhixuan/server/kbindex"
	"zhixuan/server/model"
	"zhixuan/server/qqbot"
	"zhixuan/server/vision"
	"zhixuan/server/wechat"
	"zhixuan/server/websearch"

	"github.com/openai/openai-go"
)

var noteRefRegex = regexp.MustCompile(`\[note:([^]]*):(\d+)\]`)

// qqNotifyTool allows the LLM to send QQ notification to the user.
var qqNotifyTool = openai.ChatCompletionToolParam{
	Function: openai.FunctionDefinitionParam{
		Name:        "qq_notify",
		Description: openai.String("向用户发送QQ通知消息。只在必要且用户明确要求QQ通知时才使用此工具。"),
		Parameters: openai.FunctionParameters{
			"type": "object",
			"properties": map[string]any{
				"user_id": map[string]any{
					"type":        "integer",
					"description": "用户ID",
				},
				"message": map[string]any{
					"type":        "string",
					"description": "要发送的通知消息内容",
				},
			},
			"required": []string{"user_id", "message"},
		},
	},
}

// wechatNotifyTool allows the LLM to send WeChat notification to the user.
var wechatNotifyTool = openai.ChatCompletionToolParam{
	Function: openai.FunctionDefinitionParam{
		Name:        "wechat_notify",
		Description: openai.String("向用户发送微信通知消息。只在必要且用户明确要求微信通知时才使用此工具。"),
		Parameters: openai.FunctionParameters{
			"type": "object",
			"properties": map[string]any{
				"user_id": map[string]any{
					"type":        "integer",
					"description": "用户ID",
				},
				"message": map[string]any{
					"type":        "string",
					"description": "要发送的通知消息内容",
				},
			},
			"required": []string{"user_id", "message"},
		},
	},
}

// getNoteContentTool allows the LLM to fetch note content by ID.
var getNoteContentTool = openai.ChatCompletionToolParam{
	Function: openai.FunctionDefinitionParam{
		Name:        "get_note_content",
		Description: openai.String("获取指定笔记的详细内容。当用户消息中引用了笔记（如 [note:笔记标题:笔记ID] 格式）时，使用此工具获取笔记内容以便回答用户问题。"),
		Parameters: openai.FunctionParameters{
			"type": "object",
			"properties": map[string]any{
				"note_id": map[string]any{
					"type":        "integer",
					"description": "笔记ID",
				},
			},
			"required": []string{"note_id"},
		},
	},
}

// saveNoteTool allows the LLM to save content as a note.
var saveNoteTool = openai.ChatCompletionToolParam{
	Function: openai.FunctionDefinitionParam{
		Name:        "save_note",
		Description: openai.String("保存内容为一篇笔记。当用户要求保存笔记、记录内容、备忘时可以使用此工具。"),
		Parameters: openai.FunctionParameters{
			"type": "object",
			"properties": map[string]any{
				"title": map[string]any{
					"type":        "string",
					"description": "笔记的标题",
				},
				"content": map[string]any{
					"type":        "string",
					"description": "笔记的内容",
				},
			},
			"required": []string{"title", "content"},
		},
	},
}

// searchKBTool allows the LLM to search user's knowledge bases.
var searchKBTool = openai.ChatCompletionToolParam{
	Function: openai.FunctionDefinitionParam{
		Name:        "search_knowledge_base",
		Description: openai.String("搜索用户知识库中的相关内容。当用户的问题可能涉及知识库中存储的文档内容时，使用此工具搜索相关知识。"),
		Parameters: openai.FunctionParameters{
			"type": "object",
			"properties": map[string]any{
				"kb_name": map[string]any{
					"type":        "string",
					"description": "要搜索的知识库名称",
				},
				"query": map[string]any{
					"type":        "string",
					"description": "搜索查询文本",
				},
			},
			"required": []string{"kb_name", "query"},
		},
	},
}

// webSearchTool allows the LLM to search the web.
var webSearchTool = openai.ChatCompletionToolParam{
	Function: openai.FunctionDefinitionParam{
		Name:        "web_search",
		Description: openai.String("搜索互联网获取最新信息。当用户的问题需要最新的网络信息、新闻、实时数据或你不确定的知识时，使用此工具搜索互联网。"),
		Parameters: openai.FunctionParameters{
			"type": "object",
			"properties": map[string]any{
				"query": map[string]any{
					"type":        "string",
					"description": "搜索查询文本",
				},
			},
			"required": []string{"query"},
		},
	},
}

// listKBTool allows the LLM to list user's knowledge bases.
var listKBTool = openai.ChatCompletionToolParam{
	Function: openai.FunctionDefinitionParam{
		Name:        "list_knowledge_bases",
		Description: openai.String("列出用户的所有知识库名称和简介。当你需要了解用户有哪些知识库时，使用此工具。"),
		Parameters: openai.FunctionParameters{
			"type":       "object",
			"properties": map[string]any{},
		},
	},
}

// listNotesTool allows the LLM to list user's notes.
var listNotesTool = openai.ChatCompletionToolParam{
	Function: openai.FunctionDefinitionParam{
		Name:        "list_notes",
		Description: openai.String("列出用户的所有笔记标题。当你需要了解用户有哪些笔记时，使用此工具。"),
		Parameters: openai.FunctionParameters{
			"type":       "object",
			"properties": map[string]any{},
		},
	},
}

// createTodoTool allows the LLM to create a todo item.
var createTodoTool = openai.ChatCompletionToolParam{
	Function: openai.FunctionDefinitionParam{
		Name:        "create_todo",
		Description: openai.String("创建一条待办事项。当用户要求添加待办、提醒、任务时使用此工具。"),
		Parameters: openai.FunctionParameters{
			"type": "object",
			"properties": map[string]any{
				"title": map[string]any{
					"type":        "string",
					"description": "待办标题",
				},
				"content": map[string]any{
					"type":        "string",
					"description": "待办详细内容（可选）",
				},
			},
			"required": []string{"title"},
		},
	},
}

// describeImageTool allows the LLM to understand image content from user uploads.
var describeImageTool = openai.ChatCompletionToolParam{
	Function: openai.FunctionDefinitionParam{
		Name:        "describe_image",
		Description: openai.String("当用户消息中包含 [image:...] 格式的图片引用时，使用此工具理解图片内容。"),
		Parameters: openai.FunctionParameters{
			"type": "object",
			"properties": map[string]any{
				"image_path": map[string]any{
					"type":        "string",
					"description": "图片标识，如 upload@1/xxx.png",
				},
				"prompt": map[string]any{
					"type":        "string",
					"description": "提示词（可选，为空时使用默认提示词）",
				},
			},
			"required": []string{"image_path"},
		},
	},
}

func executeSaveNote(user *model.User, argsJSON string) (string, error) {
	var args struct {
		Title   string `json:"title"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	note := model.Note{
		UserID:  user.ID,
		Title:   args.Title,
		Content: args.Content,
	}
	if err := database.DB.Create(&note).Error; err != nil {
		return "", fmt.Errorf("保存笔记失败: %w", err)
	}

	return fmt.Sprintf("已保存笔记「%s」(id=%d)", args.Title, note.ID), nil
}

func executeGetNoteContent(user *model.User, argsJSON string) (string, error) {
	var args struct {
		NoteID int `json:"note_id"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	var note model.Note
	if err := database.DB.Where("id = ? AND user_id = ?", args.NoteID, user.ID).First(&note).Error; err != nil {
		return "", fmt.Errorf("笔记不存在 (id=%d)", args.NoteID)
	}

	return fmt.Sprintf("笔记标题：%s\n\n笔记内容：\n%s", note.Title, note.Content), nil
}

// ExpandNoteRefs replaces [note:name:id] tags with readable text for display.
func ExpandNoteRefs(content string) string {
	return noteRefRegex.ReplaceAllStringFunc(content, func(match string) string {
		sub := noteRefRegex.FindStringSubmatch(match)
		if len(sub) == 3 {
			return fmt.Sprintf("@%s", sub[1])
		}
		return match
	})
}

// GetNoteRefIDs extracts all note IDs from [note:name:id] tags in the content.
func GetNoteRefIDs(content string) []int {
	matches := noteRefRegex.FindAllStringSubmatch(content, -1)
	var ids []int
	for _, m := range matches {
		if len(m) == 3 {
			if id, err := strconv.Atoi(m[2]); err == nil {
				ids = append(ids, id)
			}
		}
	}
	return ids
}

// buildSystemPrompt constructs the system prompt with specified KB list.
// If kbNames is empty, no knowledge base info is injected.
func buildSystemPrompt(userID uint, kbNames []string) string {
	base := "你是知玄AI助手，可以帮助用户回答问题、总结对话、管理笔记和待办。回答请使用中文。"
	userIDSuffix := fmt.Sprintf("\n\n用户ID: %d", userID)

	if len(kbNames) == 0 {
		return base + userIDSuffix
	}

	userDir := filepath.Join(config.KBDir(), fmt.Sprintf("%d", userID))
	entries, err := os.ReadDir(userDir)
	if err != nil || len(entries) == 0 {
		return base + userIDSuffix
	}

	// Build lookup set for requested KBs
	kbSet := make(map[string]bool, len(kbNames))
	for _, n := range kbNames {
		kbSet[n] = true
	}

	var kbLines []string
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		if !kbSet[e.Name()] {
			continue
		}
		kbPath := filepath.Join(userDir, e.Name())
		cfgData, err := os.ReadFile(filepath.Join(kbPath, "config.json"))
		if err != nil {
			kbLines = append(kbLines, fmt.Sprintf("- %s", e.Name()))
			continue
		}
		var cfg struct {
			Description string `json:"description"`
		}
		json.Unmarshal(cfgData, &cfg)
		if cfg.Description != "" {
			kbLines = append(kbLines, fmt.Sprintf("- %s: %s", e.Name(), cfg.Description))
		} else {
			kbLines = append(kbLines, fmt.Sprintf("- %s", e.Name()))
		}
	}

	if len(kbLines) == 0 {
		return base + userIDSuffix
	}

	return base + "\n\n用户拥有以下知识库：\n" + strings.Join(kbLines, "\n") + userIDSuffix
}

// ensureMainSession ensures the user has a main session, creating one if needed.
func ensureMainSession(userID uint) (*model.Session, error) {
	var session model.Session
	err := database.DB.Where("user_id = ? AND is_main = ?", userID, true).First(&session).Error
	if err == nil {
		return &session, nil
	}

	// Create main session
	session = model.Session{
		UserID:    userID,
		SessionID: model.GenerateSessionID(),
		Title:     "知玄",
		IsMain:    true,
	}
	if err := database.DB.Create(&session).Error; err != nil {
		// Concurrent creation race: re-query
		if err2 := database.DB.Where("user_id = ? AND is_main = ?", userID, true).First(&session).Error; err2 != nil {
			return nil, err
		}
		return &session, nil
	}
	return &session, nil
}

func executeSearchKB(user *model.User, argsJSON string) (string, error) {
	var args struct {
		KBName string `json:"kb_name"`
		Query  string `json:"query"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	log.Printf("[tool] search_knowledge_base 调用: kb_name=%q query=%q", args.KBName, args.Query)

	results, err := kbindex.Get().Search(stdctx.Background(), user.ID, args.KBName, args.Query)
	if err != nil {
		log.Printf("[tool] search_knowledge_base 失败: %v", err)
		return "", fmt.Errorf("搜索失败: %w", err)
	}
	if len(results) == 0 {
		log.Printf("[tool] search_knowledge_base 返回: 无结果")
		return "未找到相关内容。", nil
	}

	log.Printf("[tool] search_knowledge_base 返回: %d 条结果", len(results))

	var sb strings.Builder
	var imagePaths []string // 收集图片来源路径
	for i, r := range results {
		if i > 0 {
			sb.WriteString("\n---\n")
		}
		sb.WriteString(r.Content)
		log.Printf("[tool]   结果%d: score=%.4f content=%q", i+1, r.Score, r.Content)

		// 检测图片来源标记 [source:img:knowledge@...]
		if idx := strings.Index(r.Content, "[source:img:"); idx != -1 {
			tag := r.Content[idx:]
			if end := strings.Index(tag, "]"); end != -1 {
				path := tag[len("[source:img:"):end]
				imagePaths = append(imagePaths, path)
			}
		}
	}

	result := sb.String()

	// 如果结果中包含图片来源，在开头加上提示引导 LLM 按格式输出图片信息
	if len(imagePaths) > 0 {
		var hint strings.Builder
		hint.WriteString("找到如下结果，如果需要在回复中展示图片，请按 [image:图片名称:文件路径] 格式输出。示例：")
		for _, p := range imagePaths {
			// 从路径中提取文件名（不含扩展名）作为图片名称
			fileName := p
			if atIdx := strings.Index(p, "@"); atIdx != -1 {
				fileName = p[atIdx+1:]
			}
			base := filepath.Base(fileName)
			name := strings.TrimSuffix(base, filepath.Ext(base))
			hint.WriteString(fmt.Sprintf(" [image:%s:%s]", name, p))
		}
		hint.WriteString("\n\n")
		result = hint.String() + result
	}

	log.Printf("[tool] search_knowledge_base 完整返回:\n%s", result)
	return result, nil
}

func executeWebSearch(user *model.User, argsJSON string) (string, error) {
	var args struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	log.Printf("[tool] web_search 调用: query=%q", args.Query)

	client := websearch.New()
	results, err := client.Search(stdctx.Background(), args.Query)
	if err != nil {
		log.Printf("[tool] web_search 失败: %v", err)
		return "", fmt.Errorf("网络搜索失败: %w", err)
	}

	if len(results) == 0 {
		log.Printf("[tool] web_search 返回: 无结果")
		return "未找到相关网络信息。", nil
	}

	log.Printf("[tool] web_search 返回: %d 条结果", len(results))

	type resultItem struct {
		URL     string `json:"url"`
		Summary string `json:"summary"`
	}
	items := make([]resultItem, 0, len(results))
	for i, r := range results {
		items = append(items, resultItem{URL: r.URL, Summary: r.Summary})
		preview := r.Summary
		runes := []rune(preview)
		if len(runes) > 100 {
			preview = string(runes[:100]) + "..."
		}
		log.Printf("[tool]   结果%d: name=%q url=%q summary=%q", i+1, r.Name, r.URL, preview)
	}
	resultJSON, _ := json.Marshal(items)
	return string(resultJSON), nil
}

func executeListKB(user *model.User, argsJSON string) (string, error) {
	log.Printf("[tool] list_knowledge_bases 调用")

	userDir := filepath.Join(config.KBDir(), fmt.Sprintf("%d", user.ID))
	entries, err := os.ReadDir(userDir)
	if err != nil || len(entries) == 0 {
		return "用户暂无知识库。", nil
	}

	type kbItem struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	var items []kbItem
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		kbPath := filepath.Join(userDir, e.Name())
		cfgData, err := os.ReadFile(filepath.Join(kbPath, "config.json"))
		desc := ""
		if err == nil {
			var cfg struct {
				Description string `json:"description"`
			}
			json.Unmarshal(cfgData, &cfg)
			desc = cfg.Description
		}
		items = append(items, kbItem{Name: e.Name(), Description: desc})
	}

	if len(items) == 0 {
		return "用户暂无知识库。", nil
	}

	resultJSON, _ := json.Marshal(items)
	return string(resultJSON), nil
}

func executeListNotes(user *model.User, argsJSON string) (string, error) {
	log.Printf("[tool] list_notes 调用")

	var notes []model.Note
	database.DB.Where("user_id = ?", user.ID).Select("id, title").Order("updated_at desc").Find(&notes)

	if len(notes) == 0 {
		return "用户暂无笔记。", nil
	}

	type noteItem struct {
		ID    uint   `json:"id"`
		Title string `json:"title"`
	}
	items := make([]noteItem, 0, len(notes))
	for _, n := range notes {
		items = append(items, noteItem{ID: n.ID, Title: n.Title})
	}

	resultJSON, _ := json.Marshal(items)
	return string(resultJSON), nil
}

func executeCreateTodo(user *model.User, argsJSON string) (string, error) {
	var args struct {
		Title   string `json:"title"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	todo := model.Todo{
		UserID:  user.ID,
		Title:   args.Title,
		Content: args.Content,
	}
	if err := database.DB.Create(&todo).Error; err != nil {
		return "", fmt.Errorf("创建待办失败: %w", err)
	}

	return fmt.Sprintf("已创建待办「%s」(id=%d)", args.Title, todo.ID), nil
}

func executeDescribeImage(user *model.User, argsJSON string) (string, error) {
	var args struct {
		ImagePath string `json:"image_path"`
		Prompt    string `json:"prompt"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	// Parse image_path: upload@{userID}/{filename}
	atIdx := strings.Index(args.ImagePath, "@")
	if atIdx == -1 {
		return "", fmt.Errorf("image_path 格式错误")
	}
	source := args.ImagePath[:atIdx]
	relPath := args.ImagePath[atIdx+1:]

	if source != "upload" {
		return "", fmt.Errorf("不支持的图片来源: %s", source)
	}

	// Security: prevent path traversal
	cleanRel := filepath.Clean(relPath)
	if strings.Contains(cleanRel, "..") {
		return "", fmt.Errorf("路径不合法")
	}

	filePath := filepath.Join(config.UploadDir(), cleanRel)
	imgData, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("读取图片失败: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(cleanRel))
	description, err := vision.Describe(stdctx.Background(), imgData, ext)
	if err != nil {
		return "", fmt.Errorf("图片识别失败: %w", err)
	}

	return description, nil
}

func executeQQNotify(user *model.User, argsJSON string) (string, error) {
	var args struct {
		UserID  int    `json:"user_id"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if uint(args.UserID) != user.ID {
		return "", fmt.Errorf("无权发送通知给其他用户")
	}

	if user.QQBotAppID == "" || user.QQBotAppSecret == "" || user.QQBotOpenID == "" {
		return "", fmt.Errorf("用户未绑定QQBot")
	}

	if err := qqbot.SendMsg(user.QQBotAppID, user.QQBotAppSecret, user.QQBotOpenID, args.Message); err != nil {
		return "", fmt.Errorf("发送QQ通知失败: %w", err)
	}

	return "已发送QQ通知", nil
}

// readFileTool allows the LLM to read a file from the user's workspace.
var readFileTool = openai.ChatCompletionToolParam{
	Function: openai.FunctionDefinitionParam{
		Name:        "read_file",
		Description: openai.String("读取工作区中的文件内容。可以指定起始行号和读取行数来分段读取大文件。"),
		Parameters: openai.FunctionParameters{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "文件路径（相对于工作区根目录，如 notes/plan.md）",
				},
				"offset": map[string]any{
					"type":        "integer",
					"description": "起始行号（从1开始，默认1）",
				},
				"limit": map[string]any{
					"type":        "integer",
					"description": "最多读取的行数（默认2000）",
				},
			},
			"required": []string{"path"},
		},
	},
}

// writeFileTool allows the LLM to write content to a file in the user's workspace.
var writeFileTool = openai.ChatCompletionToolParam{
	Function: openai.FunctionDefinitionParam{
		Name:        "write_file",
		Description: openai.String("将内容写入工作区中的文件。如果文件已存在则覆盖，如果父目录不存在则自动创建。"),
		Parameters: openai.FunctionParameters{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "文件路径（相对于工作区根目录）",
				},
				"content": map[string]any{
					"type":        "string",
					"description": "要写入的完整文件内容",
				},
			},
			"required": []string{"path", "content"},
		},
	},
}

// editFileTool allows the LLM to edit a file by replacing text in the user's workspace.
var editFileTool = openai.ChatCompletionToolParam{
	Function: openai.FunctionDefinitionParam{
		Name:        "edit_file",
		Description: openai.String("编辑工作区中的文件，将文件中的 old_string 替换为 new_string。可用于精确修改文件的部分内容。"),
		Parameters: openai.FunctionParameters{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "文件路径（相对于工作区根目录）",
				},
				"old_string": map[string]any{
					"type":        "string",
					"description": "要被替换的原文本",
				},
				"new_string": map[string]any{
					"type":        "string",
					"description": "替换后的新文本",
				},
				"replace_all": map[string]any{
					"type":        "boolean",
					"description": "是否替换所有匹配项（默认false，仅替换第一处）",
				},
			},
			"required": []string{"path", "old_string", "new_string"},
		},
	},
}

// listFilesTool allows the LLM to list files and directories in the user's workspace.
var listFilesTool = openai.ChatCompletionToolParam{
	Function: openai.FunctionDefinitionParam{
		Name:        "list_files",
		Description: openai.String("列出工作区中指定目录下的文件和子目录。显示每个条目的类型、大小、修改时间和名称。目录以 / 结尾。"),
		Parameters: openai.FunctionParameters{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "要列出的目录路径（相对于工作区根目录，默认为根目录）",
				},
			},
			"required": []string{"path"},
		},
	},
}

// workspaceFilePath validates and returns the absolute path for a workspace file.
func workspaceFilePath(userID uint, relPath string) (string, error) {
	cleanRel := filepath.Clean(relPath)
	if strings.Contains(cleanRel, "..") {
		return "", fmt.Errorf("路径不合法：禁止路径逃逸")
	}
	absPath := filepath.Join(config.WorkspaceDir(), fmt.Sprintf("%d", userID), cleanRel)
	return absPath, nil
}

func executeReadFile(user *model.User, argsJSON string) (string, error) {
	var args struct {
		Path   string `json:"path"`
		Offset int    `json:"offset"`
		Limit  int    `json:"limit"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	absPath, err := workspaceFilePath(user.ID, args.Path)
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return "", fmt.Errorf("读取文件失败: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	totalLines := len(lines)

	offset := args.Offset
	if offset < 1 {
		offset = 1
	}
	limit := args.Limit
	if limit <= 0 {
		limit = 2000
	}

	start := offset - 1
	if start > totalLines {
		start = totalLines
	}
	end := start + limit
	if end > totalLines {
		end = totalLines
	}

	var sb strings.Builder
	for i := start; i < end; i++ {
		fmt.Fprintf(&sb, "%d\t%s\n", i+1, lines[i])
	}

	if end < totalLines && limit <= 2000 {
		fmt.Fprintf(&sb, "\n文件较长（共 %d 行），可使用 offset/limit 参数分段读取。", totalLines)
	}

	return sb.String(), nil
}

func executeWriteFile(user *model.User, argsJSON string) (string, error) {
	var args struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	absPath, err := workspaceFilePath(user.ID, args.Path)
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
		return "", fmt.Errorf("创建目录失败: %w", err)
	}

	if err := os.WriteFile(absPath, []byte(args.Content), 0644); err != nil {
		return "", fmt.Errorf("写入文件失败: %w", err)
	}

	return fmt.Sprintf("已写入文件 %s（%d 字节）", args.Path, len(args.Content)), nil
}

func executeEditFile(user *model.User, argsJSON string) (string, error) {
	var args struct {
		Path       string `json:"path"`
		OldString  string `json:"old_string"`
		NewString  string `json:"new_string"`
		ReplaceAll bool   `json:"replace_all"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	absPath, err := workspaceFilePath(user.ID, args.Path)
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return "", fmt.Errorf("读取文件失败: %w", err)
	}

	content := string(data)
	count := strings.Count(content, args.OldString)
	if count == 0 {
		return "", fmt.Errorf("未找到要替换的文本")
	}
	if count > 1 && !args.ReplaceAll {
		return "", fmt.Errorf("匹配到 %d 处，请缩小替换范围或设置 replace_all 为 true", count)
	}

	var newContent string
	var replaced int
	if args.ReplaceAll {
		newContent = strings.ReplaceAll(content, args.OldString, args.NewString)
		replaced = count
	} else {
		newContent = strings.Replace(content, args.OldString, args.NewString, 1)
		replaced = 1
	}

	if err := os.WriteFile(absPath, []byte(newContent), 0644); err != nil {
		return "", fmt.Errorf("写入文件失败: %w", err)
	}

	return fmt.Sprintf("已替换 %d 处", replaced), nil
}

func executeListFiles(user *model.User, argsJSON string) (string, error) {
	var args struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if args.Path == "" {
		args.Path = "."
	}

	absPath, err := workspaceFilePath(user.ID, args.Path)
	if err != nil {
		return "", err
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return "", fmt.Errorf("路径不存在: %s", args.Path)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("不是目录: %s", args.Path)
	}

	entries, err := os.ReadDir(absPath)
	if err != nil {
		return "", fmt.Errorf("读取目录失败: %w", err)
	}

	if len(entries) == 0 {
		return "(空目录)", nil
	}

	// Sort: directories first, then files, case-insensitive
	sort.Slice(entries, func(i, j int) bool {
		iDir := entries[i].IsDir()
		jDir := entries[j].IsDir()
		if iDir != jDir {
			return iDir
		}
		return strings.ToLower(entries[i].Name()) < strings.ToLower(entries[j].Name())
	})

	type entryItem struct {
		Name    string `json:"name"`
		Type    string `json:"type"`
		Size    int64  `json:"size,omitempty"`
		ModTime string `json:"mod_time"`
	}

	items := make([]entryItem, 0, len(entries))
	for _, e := range entries {
		item := entryItem{
			Name: e.Name(),
		}
		if e.IsDir() {
			item.Type = "dir"
			item.Name += "/"
		} else {
			item.Type = "file"
			info, err := e.Info()
			if err == nil {
				item.Size = info.Size()
			}
		}
		info, err := e.Info()
		if err == nil {
			item.ModTime = info.ModTime().Format("2006-01-02 15:04")
		}
		items = append(items, item)
	}

	resultJSON, _ := json.Marshal(items)
	return string(resultJSON), nil
}

func executeWeChatNotify(user *model.User, argsJSON string) (string, error) {
	var args struct {
		UserID  int    `json:"user_id"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	log.Printf("[tool] wechat_notify 调用: user_id=%d message=%q", args.UserID, args.Message)

	if uint(args.UserID) != user.ID {
		return "", fmt.Errorf("无权发送通知给其他用户")
	}

	statePath := wechat.StatePath(config.DataDir, user.ID)
	if !wechat.StateFileExists(statePath) {
		return "", fmt.Errorf("用户未绑定微信")
	}

	client := wechat.NewClient(statePath)
	toUserID := client.UserID()
	log.Printf("[tool] wechat_notify 发送: to=%s state_path=%s", toUserID, statePath)

	if toUserID == "" {
		return "", fmt.Errorf("微信用户ID为空，无法发送")
	}

	if err := client.SendText(toUserID, args.Message, ""); err != nil {
		log.Printf("[tool] wechat_notify 发送失败: %v", err)
		return "", fmt.Errorf("发送微信通知失败: %w", err)
	}

	log.Printf("[tool] wechat_notify 发送成功: to=%s", toUserID)
	return "已发送微信通知", nil
}

// --- Agent SQLite 数据库工具 ---

// listTablesTool allows the LLM to list tables in its private SQLite database.
var listTablesTool = openai.ChatCompletionToolParam{
	Function: openai.FunctionDefinitionParam{
		Name:        "list_tables",
		Description: openai.String("列出你的独立数据库中的所有表。首次使用数据库前先调用此工具了解现状，避免重复建表。"),
		Parameters: openai.FunctionParameters{
			"type":       "object",
			"properties": map[string]any{},
		},
	},
}

// describeTableTool allows the LLM to inspect a table's schema.
var describeTableTool = openai.ChatCompletionToolParam{
	Function: openai.FunctionDefinitionParam{
		Name:        "describe_table",
		Description: openai.String("查看指定表的建表语句和字段结构。"),
		Parameters: openai.FunctionParameters{
			"type": "object",
			"properties": map[string]any{
				"table_name": map[string]any{
					"type":        "string",
					"description": "表名（仅允许字母、数字、下划线）",
				},
			},
			"required": []string{"table_name"},
		},
	},
}

// dumpSchemaTool allows the LLM to see all tables and their CREATE statements at once.
var dumpSchemaTool = openai.ChatCompletionToolParam{
	Function: openai.FunctionDefinitionParam{
		Name:        "dump_schema",
		Description: openai.String("查看你的数据库中所有表及其建表语句。首次使用数据库或需要了解现有数据结构时调用，避免重复建表。"),
		Parameters: openai.FunctionParameters{
			"type":       "object",
			"properties": map[string]any{},
		},
	},
}

// queryTool allows the LLM to run read-only SELECT/WITH queries.
var queryTool = openai.ChatCompletionToolParam{
	Function: openai.FunctionDefinitionParam{
		Name:        "query",
		Description: openai.String("在你的独立数据库中执行只读查询（SELECT 或 WITH 开头）。返回 JSON 数组，最多 200 行，单字段值超过 500 字符会被截断。"),
		Parameters: openai.FunctionParameters{
			"type": "object",
			"properties": map[string]any{
				"sql": map[string]any{
					"type":        "string",
					"description": "要执行的只读 SQL 语句（SELECT 或 WITH 开头，禁止分号）",
				},
			},
			"required": []string{"sql"},
		},
	},
}

// executeSQLTool allows the LLM to run DDL/DML statements.
var executeSQLTool = openai.ChatCompletionToolParam{
	Function: openai.FunctionDefinitionParam{
		Name:        "execute",
		Description: openai.String("在你的独立数据库中执行 DDL/DML（建表、插入、更新、删除等）。返回影响行数。禁止 ATTACH/DETACH/PRAGMA。"),
		Parameters: openai.FunctionParameters{
			"type": "object",
			"properties": map[string]any{
				"sql": map[string]any{
					"type":        "string",
					"description": "要执行的 SQL 语句（CREATE/INSERT/UPDATE/DELETE/DROP/ALTER 等）",
				},
			},
			"required": []string{"sql"},
		},
	},
}

func executeListTables(user *model.User, argsJSON string) (string, error) {
	db, err := database.GetAgentDB(user.ID)
	if err != nil {
		return "", err
	}

	var names []string
	if err := db.Raw("SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name").Scan(&names).Error; err != nil {
		return "", fmt.Errorf("查询表列表失败: %w", err)
	}

	type tableItem struct {
		Name string `json:"name"`
	}
	items := make([]tableItem, 0, len(names))
	for _, n := range names {
		items = append(items, tableItem{Name: n})
	}

	resultJSON, _ := json.Marshal(items)
	return string(resultJSON), nil
}

func executeDumpSchema(user *model.User, argsJSON string) (string, error) {
	db, err := database.GetAgentDB(user.ID)
	if err != nil {
		return "", err
	}

	type schemaItem struct {
		Name      string `json:"name"`
		CreateSQL string `json:"create_sql"`
	}
	items := make([]schemaItem, 0)
	if err := db.Raw("SELECT name, sql FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name").Scan(&items).Error; err != nil {
		return "", fmt.Errorf("查询表结构失败: %w", err)
	}

	resultJSON, _ := json.Marshal(items)
	return string(resultJSON), nil
}

func executeDescribeTable(user *model.User, argsJSON string) (string, error) {
	var args struct {
		TableName string `json:"table_name"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if !validTableName(args.TableName) {
		return "", fmt.Errorf("表名只能包含字母、数字、下划线")
	}

	db, err := database.GetAgentDB(user.ID)
	if err != nil {
		return "", err
	}

	var createSQL string
	if err := db.Raw("SELECT sql FROM sqlite_master WHERE type='table' AND name=?", args.TableName).Scan(&createSQL).Error; err != nil {
		return "", fmt.Errorf("查询表结构失败: %w", err)
	}
	if createSQL == "" {
		return "", fmt.Errorf("表 %s 不存在", args.TableName)
	}

	type columnInfo struct {
		CID       int    `json:"cid"`
		Name      string `json:"name"`
		Type      string `json:"type"`
		NotNull   int    `json:"not_null"`
		DfltValue string `json:"dflt_value"`
		PK        int    `json:"pk"`
	}
	var columns []columnInfo
	if err := db.Raw(fmt.Sprintf("PRAGMA table_info(%s)", args.TableName)).Scan(&columns).Error; err != nil {
		return "", fmt.Errorf("查询字段信息失败: %w", err)
	}

	result := map[string]interface{}{
		"create_sql": createSQL,
		"columns":    columns,
	}
	resultJSON, _ := json.Marshal(result)
	return string(resultJSON), nil
}

func executeQuerySQL(user *model.User, argsJSON string) (string, error) {
	var args struct {
		SQL string `json:"sql"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	sql := strings.TrimSpace(args.SQL)
	if sql == "" {
		return "", fmt.Errorf("SQL 不能为空")
	}

	// Read-only check: first word must be SELECT or WITH
	fields := strings.Fields(sql)
	if len(fields) == 0 {
		return "", fmt.Errorf("SQL 不能为空")
	}
	upperFirst := strings.ToUpper(fields[0])
	if upperFirst != "SELECT" && upperFirst != "WITH" {
		return "", fmt.Errorf("query 仅允许执行 SELECT 或 WITH 查询")
	}

	// Multi-statement protection: no semicolons allowed
	if strings.Contains(sql, ";") {
		return "", fmt.Errorf("SQL 中不允许出现分号")
	}

	// Auto-append LIMIT 200 if not present
	if !hasLimitClause(sql) {
		sql = sql + " LIMIT 200"
	}

	ctx, cancel := stdctx.WithTimeout(stdctx.Background(), 5*time.Second)
	defer cancel()

	db, err := database.GetAgentDB(user.ID)
	if err != nil {
		return "", err
	}

	rows, err := db.WithContext(ctx).Raw(sql).Rows()
	if err != nil {
		return "", fmt.Errorf("查询失败: %w", err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return "", fmt.Errorf("获取列信息失败: %w", err)
	}

	results := make([]map[string]interface{}, 0)
	for rows.Next() {
		values := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range values {
			ptrs[i] = &values[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return "", fmt.Errorf("读取数据失败: %w", err)
		}
		row := make(map[string]interface{})
		for i, col := range cols {
			row[col] = truncateValue(values[i], 500)
		}
		results = append(results, row)
	}

	if err := rows.Err(); err != nil {
		return "", fmt.Errorf("查询失败: %w", err)
	}

	resultJSON, _ := json.Marshal(results)
	return string(resultJSON), nil
}

func executeExecSQL(user *model.User, argsJSON string) (string, error) {
	var args struct {
		SQL string `json:"sql"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	sql := strings.TrimSpace(args.SQL)
	if sql == "" {
		return "", fmt.Errorf("SQL 不能为空")
	}

	// Isolation bypass & connection behavior protection
	upper := strings.ToUpper(sql)
	if strings.Contains(upper, "ATTACH") {
		return "", fmt.Errorf("禁止执行 ATTACH 语句")
	}
	if strings.Contains(upper, "DETACH") {
		return "", fmt.Errorf("禁止执行 DETACH 语句")
	}
	if strings.Contains(upper, "PRAGMA") {
		return "", fmt.Errorf("禁止执行 PRAGMA 语句")
	}

	ctx, cancel := stdctx.WithTimeout(stdctx.Background(), 5*time.Second)
	defer cancel()

	db, err := database.GetAgentDB(user.ID)
	if err != nil {
		return "", err
	}

	result := db.WithContext(ctx).Exec(sql)
	if result.Error != nil {
		return "", fmt.Errorf("执行失败: %w", result.Error)
	}

	return fmt.Sprintf(`{"rows_affected":%d}`, result.RowsAffected), nil
}

// validTableName checks that a table name only contains [A-Za-z0-9_].
func validTableName(name string) bool {
	if name == "" {
		return false
	}
	for _, r := range name {
		if !(r >= 'A' && r <= 'Z') && !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9') && r != '_' {
			return false
		}
	}
	return true
}

var limitClauseRegex = regexp.MustCompile(`(?i)\blimit\s+\d+`)

// hasLimitClause checks if the SQL already contains a LIMIT clause.
func hasLimitClause(sql string) bool {
	return limitClauseRegex.MatchString(sql)
}

// truncateValue truncates long string/[]byte values to maxLen runes.
func truncateValue(v interface{}, maxLen int) interface{} {
	switch val := v.(type) {
	case string:
		runes := []rune(val)
		if len(runes) > maxLen {
			return string(runes[:maxLen]) + "...(截断)"
		}
		return val
	case []byte:
		s := string(val)
		runes := []rune(s)
		if len(runes) > maxLen {
			return string(runes[:maxLen]) + "...(截断)"
		}
		return s
	default:
		return v
	}
}

// loadSkillTool allows the LLM to fetch the detail of a user-defined skill.
var loadSkillTool = openai.ChatCompletionToolParam{
	Function: openai.FunctionDefinitionParam{
		Name: "load_skill",
		Description: openai.String("加载用户自定义 skill 的详细提示词。每轮对话上下文会以列表形式提供已启用的 skill（含名称、摘要、是否带有详情）。当用户当前意图与某个 skill 的摘要相关、且该 skill 标注为有详情时，调用本工具获取详情，并按详情中的指令执行。列表中标注无详情的 skill 无需调用，直接依据其摘要判断即可。"),
		Parameters: openai.FunctionParameters{
			"type": "object",
			"properties": map[string]any{
				"name": map[string]any{
					"type":        "string",
					"description": "要加载详情的 skill 名称",
				},
			},
			"required": []string{"name"},
		},
	},
}

func executeLoadSkill(user *model.User, argsJSON string) (string, error) {
	var args struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	name := strings.TrimSpace(args.Name)
	if name == "" {
		return "", fmt.Errorf("name 不能为空")
	}
	var skill model.Skill
	if err := database.DB.Where("name = ? AND user_id = ?", name, user.ID).First(&skill).Error; err != nil {
		return "", fmt.Errorf("skill 不存在: %s", name)
	}
	if strings.TrimSpace(skill.Detail) == "" {
		return fmt.Sprintf("skill %q 没有详情，请直接依据其摘要判断", name), nil
	}
	return skill.Detail, nil
}

// buildContextPayload 构造注入在伪造 tool call 中的动态上下文：
// 当前时间 + 用户启用的 skill 列表（仅 name/summary/has_detail）+ 会话 LRU 记忆窗口快照。
// 这些信息不进 system prompt，避免污染稳定前缀、保住 prompt cache。
func buildContextPayload(userID uint, now string, mems []model.Memory) string {
	type skillItem struct {
		Name      string `json:"name"`
		Summary   string `json:"summary"`
		HasDetail bool   `json:"has_detail"`
	}
	var skills []model.Skill
	database.DB.Where("user_id = ? AND enabled = ?", userID, true).Order("sort asc, updated_at desc").Find(&skills)
	skillItems := make([]skillItem, 0, len(skills))
	for _, s := range skills {
		skillItems = append(skillItems, skillItem{
			Name:      s.Name,
			Summary:   s.Summary,
			HasDetail: strings.TrimSpace(s.Detail) != "",
		})
	}

	type memItem struct {
		ID      uint   `json:"id"`
		Type    string `json:"type"`
		Content string `json:"content"`
	}
	memItems := make([]memItem, 0, len(mems))
	for _, m := range mems {
		memItems = append(memItems, memItem{ID: m.ID, Type: m.Type, Content: m.Content})
	}

	payload := struct {
		CurrentTime string      `json:"current_time"`
		Skills      []skillItem `json:"skills"`
		Memories    []memItem   `json:"memories"`
	}{
		CurrentTime: now,
		Skills:      skillItems,
		Memories:    memItems,
	}
	b, _ := json.Marshal(payload)
	return string(b)
}
