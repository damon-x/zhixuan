package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"zhixuan/server/config"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

type ToolCall struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	CreatedAt  time.Time  `json:"created_at,omitempty"`
}

var (
	client openai.Client
	inited bool

	models   []string
	modelsMu sync.Mutex
)

func Init() {
	opts := []option.RequestOption{
		option.WithAPIKey(config.LLMAPIKey),
	}
	if config.LLMBaseURL != "" {
		opts = append(opts, option.WithBaseURL(config.LLMBaseURL))
	}
	client = openai.NewClient(opts...)
	initModels()
	inited = true
}

func initModels() {
	modelsMu.Lock()
	defer modelsMu.Unlock()
	models = make([]string, len(config.LLMModels))
	copy(models, config.LLMModels)
}

// doCompletion calls client.Chat.Completions.New with automatic model fallback.
// It tries each model in the priority list; on failure it moves that model to
// the end and tries the next one.
func doCompletion(ctx context.Context, params openai.ChatCompletionNewParams) (*openai.ChatCompletion, error) {
	modelsMu.Lock()
	list := make([]string, len(models))
	copy(list, models)
	modelsMu.Unlock()

	var lastErr error
	for i, m := range list {
		params.Model = openai.ChatModel(m)
		resp, err := client.Chat.Completions.New(ctx, params)
		if err == nil {
			usage := resp.Usage
			cachedTokens := usage.PromptTokensDetails.CachedTokens
			log.Printf("[llm] 模型 %s 调用成功 total_tokens=%d cached_tokens=%d", m, usage.TotalTokens, cachedTokens)
			return resp, nil
		}
		// context canceled — stop immediately, don't try other models
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		lastErr = err
		nextModel := ""
		if i+1 < len(list) {
			nextModel = list[i+1]
		}
		log.Printf("[llm] 模型 %s 调用失败: %v，切换到下一个模型 %s", m, err, nextModel)
		// move failed model to end
		modelsMu.Lock()
		models = append(models[:0], list...)
		models = append(models[:i], models[i+1:]...)
		models = append(models, m)
		modelsMu.Unlock()
	}
	log.Printf("[llm] 所有模型均不可用，最后错误: %v", lastErr)
	return nil, lastErr
}

func ensureInit() {
	if !inited {
		Init()
	}
}

func buildMessageParams(messages []Message) []openai.ChatCompletionMessageParamUnion {
	var params []openai.ChatCompletionMessageParamUnion
	for _, msg := range messages {
		switch msg.Role {
		case "system":
			params = append(params, openai.SystemMessage(msg.Content))
		case "user":
			params = append(params, openai.UserMessage(msg.Content))
		case "assistant":
			if len(msg.ToolCalls) > 0 {
				var tcParams []openai.ChatCompletionMessageToolCallParam
				for _, tc := range msg.ToolCalls {
					tcParams = append(tcParams, openai.ChatCompletionMessageToolCallParam{
						ID: tc.ID,
						Function: openai.ChatCompletionMessageToolCallFunctionParam{
							Name:      tc.Name,
							Arguments: tc.Arguments,
						},
					})
				}
				assistant := openai.ChatCompletionAssistantMessageParam{
					ToolCalls: tcParams,
				}
				if msg.Content != "" {
					assistant.Content.OfString = openai.String(msg.Content)
				}
				params = append(params, openai.ChatCompletionMessageParamUnion{OfAssistant: &assistant})
			} else {
				params = append(params, openai.AssistantMessage(msg.Content))
			}
		case "tool":
			params = append(params, openai.ToolMessage(msg.Content, msg.ToolCallID))
		}
	}
	return params
}

// Chat performs a simple LLM call without tools (used for internal tasks like summarization).
func Chat(ctx context.Context, messages []Message) (string, error) {
	ensureInit()

	msgParams := buildMessageParams(messages)
	response, err := doCompletion(ctx, openai.ChatCompletionNewParams{
		Messages: msgParams,
	})
	if err != nil {
		return "", fmt.Errorf("LLM call failed: %w", err)
	}
	if len(response.Choices) == 0 {
		return "", fmt.Errorf("no response from LLM")
	}
	return response.Choices[0].Message.Content, nil
}

// ToolCallResult holds the result of a single LLM call that may or may not invoke a tool.
type ToolCallResult struct {
	ToolCalled bool
	ToolName   string
	ToolArgs   string // raw JSON arguments
	TextReply  string // text response when no tool is called
}

// ChatForToolCall makes a single LLM call with tools and returns whether a tool was called.
// If the LLM calls a tool, ToolCalled is true and ToolName/ToolArgs are populated.
// If the LLM responds with text, ToolCalled is false and TextReply is populated.
func ChatForToolCall(ctx context.Context, messages []Message, tools []openai.ChatCompletionToolParam) (*ToolCallResult, error) {
	ensureInit()

	msgParams := buildMessageParams(messages)
	params := openai.ChatCompletionNewParams{
		Messages: msgParams,
	}
	if len(tools) > 0 {
		params.Tools = tools
	}

	response, err := doCompletion(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}
	if len(response.Choices) == 0 {
		return nil, fmt.Errorf("no response from LLM")
	}

	choice := response.Choices[0]
	result := &ToolCallResult{}

	if len(choice.Message.ToolCalls) > 0 {
		tc := choice.Message.ToolCalls[0]
		result.ToolCalled = true
		result.ToolName = tc.Function.Name
		result.ToolArgs = tc.Function.Arguments
	} else {
		result.ToolCalled = false
		result.TextReply = choice.Message.Content
	}

	return result, nil
}

// ChatWithTools implements a ReAct agent loop with tool calling capability.
// It repeatedly calls the LLM, executing any tool calls, until the LLM returns
// a final text response or the maximum number of iterations is reached.
// Returns the final text reply, any intermediate messages (tool call + tool result) for caching,
// the total_tokens of the last LLM call (0 if unavailable), and an error.
func ChatWithTools(ctx context.Context, messages []Message, tools []openai.ChatCompletionToolParam, executeTool func(name, args string) (string, error)) (string, []Message, int, error) {
	ensureInit()

	msgParams := buildMessageParams(messages)
	var intermediates []Message

	// 维护完整消息列表用于调试日志
	logMsgs := make([]Message, len(messages))
	copy(logMsgs, messages)

	// ReAct 循环里上下文只增不减，最后一次 doCompletion 的 total_tokens 即本轮峰值
	lastTotalTokens := 0

	const maxIterations = 5
	for i := 0; i < maxIterations; i++ {
		// Check for cancellation
		select {
		case <-ctx.Done():
			return "", intermediates, lastTotalTokens, ctx.Err()
		default:
		}

		// 打印完整的 messages 列表
		if data, err := json.Marshal(logMsgs); err == nil {
			log.Printf("[llm] iter=%d 完整messages(共%d条): %s", i, len(logMsgs), string(data))
		} else {
			log.Printf("[llm] iter=%d messages序列化失败: %v", i, err)
		}

		params := openai.ChatCompletionNewParams{
			Messages: msgParams,
		}
		if len(tools) > 0 {
			params.Tools = tools
		}

		response, err := doCompletion(ctx, params)
		if err != nil {
			return "", intermediates, lastTotalTokens, fmt.Errorf("LLM call failed: %w", err)
		}
		if len(response.Choices) == 0 {
			return "", intermediates, lastTotalTokens, fmt.Errorf("no response from LLM")
		}
		lastTotalTokens = int(response.Usage.TotalTokens)

		choice := response.Choices[0]

		// No tool calls — return the text response
		if len(choice.Message.ToolCalls) == 0 {
			log.Printf("[llm] iter=%d 无tool call，直接回复: %q", i, choice.Message.Content)
			return choice.Message.Content, intermediates, lastTotalTokens, nil
		}

		// Log tool calls
		for ti, tc := range choice.Message.ToolCalls {
			log.Printf("[llm] iter=%d tool_call[%d]: name=%s args=%s", i, ti, tc.Function.Name, tc.Function.Arguments)
		}

		// Track assistant message with tool calls
		var tcs []ToolCall
		for _, tc := range choice.Message.ToolCalls {
			tcs = append(tcs, ToolCall{
				ID:        tc.ID,
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			})
		}
		intermediates = append(intermediates, Message{
			Role:      "assistant",
			Content:   choice.Message.Content,
			ToolCalls: tcs,
		})

		// 同步更新日志消息列表
		logMsgs = append(logMsgs, Message{
			Role:      "assistant",
			Content:   choice.Message.Content,
			ToolCalls: tcs,
		})

		// Add assistant message (with tool calls) to conversation history
		msgParams = append(msgParams, choice.Message.ToParam())

		// Execute each tool call and append results
		for _, toolCall := range choice.Message.ToolCalls {
			result, err := executeTool(toolCall.Function.Name, toolCall.Function.Arguments)
			if err != nil {
				result = fmt.Sprintf("Error executing tool %s: %v", toolCall.Function.Name, err)
			}
			log.Printf("[llm] iter=%d tool_result for %s: %q", i, toolCall.Function.Name, result)
			intermediates = append(intermediates, Message{
				Role:       "tool",
				Content:    result,
				ToolCallID: toolCall.ID,
			})
			logMsgs = append(logMsgs, Message{
				Role:       "tool",
				Content:    result,
				ToolCallID: toolCall.ID,
			})
			msgParams = append(msgParams, openai.ToolMessage(result, toolCall.ID))
		}
	}

	return "", intermediates, lastTotalTokens, fmt.Errorf("agent reached maximum iterations (%d)", maxIterations)
}
