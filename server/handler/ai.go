package handler

import (
	"encoding/json"
	"net/http"

	"zhixuan/server/llm"

	"github.com/gin-gonic/gin"
	"github.com/openai/openai-go"
)

var summarizeNoteTool = openai.ChatCompletionToolParam{
	Function: openai.FunctionDefinitionParam{
		Name:        "summarize_note",
		Description: openai.String("总结笔记内容，生成结构化的摘要。请将总结内容写入 summary 字段。"),
		Parameters: openai.FunctionParameters{
			"type": "object",
			"properties": map[string]any{
				"summary": map[string]any{
					"type":        "string",
					"description": "笔记的总结内容",
				},
			},
			"required": []string{"summary"},
		},
	},
}

var generateTodosTool = openai.ChatCompletionToolParam{
	Function: openai.FunctionDefinitionParam{
		Name:        "generate_todos",
		Description: openai.String("根据内容生成待办事项列表。请将待办列表写入 todos 字段。"),
		Parameters: openai.FunctionParameters{
			"type": "object",
			"properties": map[string]any{
				"todos": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"title": map[string]any{
								"type":        "string",
								"description": "待办标题",
							},
							"content": map[string]any{
								"type":        "string",
								"description": "待办内容描述",
							},
							"priority": map[string]any{
								"type":        "integer",
								"description": "优先级：0低 1中 2高",
							},
						},
						"required": []string{"title"},
					},
				},
			},
			"required": []string{"todos"},
		},
	},
}

func getToolByName(name string) (openai.ChatCompletionToolParam, bool) {
	switch name {
	case "summarize_note":
		return summarizeNoteTool, true
	case "generate_todos":
		return generateTodosTool, true
	default:
		return openai.ChatCompletionToolParam{}, false
	}
}

func ExecuteAiTask(c *gin.Context) {
	_, ok := GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 1, "msg": "未登录"})
		return
	}

	var req struct {
		Prompt   string `json:"prompt" binding:"required"`
		ToolName string `json:"tool_name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "参数错误"})
		return
	}

	tool, found := getToolByName(req.ToolName)
	if !found {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "未知的工具: " + req.ToolName})
		return
	}

	messages := []llm.Message{
		{Role: "system", Content: "你是一个AI助手。请分析用户的内容，并通过调用提供的工具来返回结果。你必须调用工具来返回结构化数据。"},
		{Role: "user", Content: req.Prompt},
	}

	result, err := llm.ChatForToolCall(c.Request.Context(), messages, []openai.ChatCompletionToolParam{tool})
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "AI 任务失败: " + err.Error()})
		return
	}

	if result.ToolCalled {
		// Parse the tool arguments as the response data
		c.JSON(http.StatusOK, gin.H{
			"code": 0,
			"data": json.RawMessage(result.ToolArgs),
		})
	} else {
		// LLM returned text instead of tool call — treat as failure
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": result.TextReply})
	}
}
