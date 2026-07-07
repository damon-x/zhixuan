package gateway

import (
	stdctx "context"
	"encoding/json"
	"fmt"
	"log"

	"zhixuan/server/memory"
	"zhixuan/server/model"

	"github.com/openai/openai-go"
)

// saveMemoryTool 记忆 agent 专用：写入一条长期记忆。
var saveMemoryTool = openai.ChatCompletionToolParam{
	Function: openai.FunctionDefinitionParam{
		Name:        "save_memory",
		Description: openai.String("将一条值得长期记住的信息写入记忆。仅记录用户偏好、重要事实、人际关系、关键事件、长期目标。不要记录寒暄、临时任务或已在笔记/待办中的内容。"),
		Parameters: openai.FunctionParameters{
			"type": "object",
			"properties": map[string]any{
				"type": map[string]any{
					"type": "string",
					"enum": []string{
						model.MemoryTypePreference,
						model.MemoryTypeFact,
						model.MemoryTypeRelationship,
						model.MemoryTypeEvent,
						model.MemoryTypeGoal,
					},
					"description": "记忆类型：preference=偏好, fact=事实, relationship=人际关系, event=事件, goal=目标",
				},
				"content": map[string]any{
					"type":        "string",
					"description": "记忆内容，用简洁的陈述句记录，例如「用户喜欢喝美式咖啡」",
				},
				"tags": map[string]any{
					"type":        "string",
					"description": "可选标签，逗号分隔，例如「饮食,咖啡」",
				},
			},
			"required": []string{"type", "content"},
		},
	},
}

// updateMemoryTool 记忆 agent 专用：更新已存在的记忆（合并去重）。
var updateMemoryTool = openai.ChatCompletionToolParam{
	Function: openai.FunctionDefinitionParam{
		Name:        "update_memory",
		Description: openai.String("更新一条已有记忆的内容或标签，用于发现相似记忆时合并而非重复写入。"),
		Parameters: openai.FunctionParameters{
			"type": "object",
			"properties": map[string]any{
				"id": map[string]any{
					"type":        "integer",
					"description": "要更新的记忆ID",
				},
				"content": map[string]any{
					"type":        "string",
					"description": "新的记忆内容",
				},
				"tags": map[string]any{
					"type":        "string",
					"description": "新的标签（逗号分隔），留空不改",
				},
			},
			"required": []string{"id", "content"},
		},
	},
}

// searchMemoryTool 记忆 agent 专用：语义检索已有记忆，用于写入前查重。
var searchMemoryTool = openai.ChatCompletionToolParam{
	Function: openai.FunctionDefinitionParam{
		Name:        "search_memory",
		Description: openai.String("语义搜索已有记忆，判断是否已记录过类似内容，避免重复写入。"),
		Parameters: openai.FunctionParameters{
			"type": "object",
			"properties": map[string]any{
				"query": map[string]any{
					"type":        "string",
					"description": "搜索词，通常是即将写入的记忆内容",
				},
				"top_k": map[string]any{
					"type":        "integer",
					"description": "返回数量，默认5",
				},
			},
			"required": []string{"query"},
		},
	},
}

func executeSaveMemory(userID uint, sessionID string, argsJSON string) (string, error) {
	var args struct {
		Type    string `json:"type"`
		Content string `json:"content"`
		Tags    string `json:"tags"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	if args.Content == "" {
		return "", fmt.Errorf("content 不能为空")
	}
	log.Printf("[tool] save_memory user=%d type=%s content=%q", userID, args.Type, args.Content)

	mem, err := memory.Get().Save(stdctx.Background(), userID, args.Type, args.Content, args.Tags, sessionID)
	if err != nil {
		return "", fmt.Errorf("保存记忆失败: %w", err)
	}
	return fmt.Sprintf("已保存记忆 (id=%d, type=%s)", mem.ID, mem.Type), nil
}

func executeUpdateMemory(userID uint, argsJSON string) (string, error) {
	var args struct {
		ID      uint   `json:"id"`
		Content string `json:"content"`
		Tags    string `json:"tags"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	log.Printf("[tool] update_memory user=%d id=%d", userID, args.ID)

	if err := memory.Get().Update(stdctx.Background(), userID, args.ID, args.Content, args.Tags); err != nil {
		return "", fmt.Errorf("更新记忆失败: %w", err)
	}
	return fmt.Sprintf("已更新记忆 id=%d", args.ID), nil
}

func executeSearchMemory(userID uint, argsJSON string) (string, error) {
	var args struct {
		Query string `json:"query"`
		TopK  int    `json:"top_k"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	if args.Query == "" {
		return "", fmt.Errorf("query 不能为空")
	}
	log.Printf("[tool] search_memory user=%d query=%q", userID, args.Query)

	mems, err := memory.Get().Search(stdctx.Background(), userID, args.Query, args.TopK)
	if err != nil {
		return "", fmt.Errorf("搜索记忆失败: %w", err)
	}
	if len(mems) == 0 {
		return "无相似记忆", nil
	}
	out, _ := json.Marshal(mems)
	return string(out), nil
}
