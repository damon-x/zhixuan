package gateway

import (
	stdctx "context"
	"fmt"

	"zhixuan/server/llm"

	"github.com/openai/openai-go"
)

// memorySystemPrompt 指导记忆 agent 判断与写入。
const memorySystemPrompt = `你是记忆助手。任务是分析下面这轮对话，判断其中是否有值得长期记住的用户信息。

值得记录的：
- 用户偏好（饮食、作息、工作习惯、技术栈等）
- 重要事实（职业、所在地、家庭情况、身体状况等）
- 人际关系（家人、朋友、同事的名字和关系）
- 关键事件（旅行、项目、里程碑、重要决策等）
- 长期目标（学习计划、职业规划、人生愿望等）

不值得记录的：
- 寒暄、闲聊
- 临时的待办和任务（已由待办系统管理）
- 已在笔记/知识库中的内容
- AI 自己的回复内容（只记录用户的信息）

工作流程：
1. 通读整轮对话，一次性识别出所有值得记录的事实。识别阶段就要对"同一信息的不同表述/不同粒度"做合并：同一件事只对应一条记忆，不要把一个事实拆成多条细节分别记录。
2. 在一条回复里并行发起多个 search_memory（每条待写事实一个查询），一次性查重已有记忆，减少请求往返。
3. 对每条事实：已存在相似记忆则 update_memory 合并；否则 save_memory 写入。写入和更新也可以合并到同一条回复里并行调用。
4. 没有值得记录的内容时，不调用任何工具直接结束。

写入原则：
- 粒度适中：一条记忆记一个完整事实，不要过细。例如"用户是程序员"就够了，像"用户自谦为底层大头兵"这种细节应作为同一条记忆的补充合并进去，而不是另开一条。先想清楚每条事实的最终表述再写，一次写到位。
- 同一轮内，同一事实只产生一条记忆：要么 save 一次，要么 update 一条旧的，绝不存两条内容相近的记忆。
- update_memory 只用于合并"本轮开始前就已存在"的旧记忆。本轮你已经 save_memory 写入的，不得再对它调用 update_memory——要改就在那次 save 时一次写到位。
- 每条用简洁的陈述句。宁缺毋滥，只记录确实有长期价值的信息。`

// startMemoryAgent 提交记忆整理任务到全局调度器。
// 调度器按 userID 维护独立队列（容量 2：1 运行 + 1 等待），
// 保证同一用户的记忆 agent 串行执行，避免并行写入重复记忆。
func startMemoryAgent(userID uint, sessionID string, roundMsgs []llm.Message) {
	memorySched.submit(memoryTask{
		userID:    userID,
		sessionID: sessionID,
		msgs:      roundMsgs,
	})
}

// runMemoryAgent 组装记忆 agent 并复用通用 agent 循环。
func runMemoryAgent(ctx stdctx.Context, userID uint, sessionID string, roundMsgs []llm.Message) error {
	messages := make([]llm.Message, 0, len(roundMsgs)+2)
	messages = append(messages, llm.Message{Role: "system", Content: memorySystemPrompt})
	messages = append(messages, roundMsgs...)
	// 引导 agent 开始分析（roundMsgs 末尾是 assistant 回复，需要 user 触发下一轮）
	messages = append(messages, llm.Message{
		Role:    "user",
		Content: "请分析以上对话，按工作流程处理值得记忆的内容。",
	})

	tools := []openai.ChatCompletionToolParam{
		saveMemoryTool,
		updateMemoryTool,
		searchMemoryTool,
	}

	executeTool := func(name, argsJSON string) (string, error) {
		switch name {
		case "save_memory":
			return executeSaveMemory(userID, sessionID, argsJSON)
		case "update_memory":
			return executeUpdateMemory(userID, argsJSON)
		case "search_memory":
			return executeSearchMemory(userID, argsJSON)
		default:
			return "", fmt.Errorf("unknown tool: %s", name)
		}
	}

	// 给 user/assistant 消息加时间戳前缀（仅发送给 LLM，不改原 Content）。
	// 记忆 agent 需要感知消息时间才能从"我每天都这个时间下班"推断出具体下班时刻。
	messages = applyTimestamps(messages)
	_, _, _, err := llm.ChatWithTools(ctx, messages, tools, executeTool)
	return err
}
