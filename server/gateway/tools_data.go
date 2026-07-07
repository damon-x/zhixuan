package gateway

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"zhixuan/server/database"
	"zhixuan/server/model"

	cronsched "github.com/robfig/cron/v3"
	"github.com/openai/openai-go"
)

// RegisterScheduleFunc 由 main.go 注入（scheduler.AddJob），用于打破 gateway ↔ scheduler 循环依赖。
// gateway 通过此回调注册新建的定时任务，而无需直接 import scheduler。
var RegisterScheduleFunc func(*model.Schedule) error

// validateScheduleTime 校验定时时间格式（cron 表达式或单次时间）。
func validateScheduleTime(mode, cronStr string) error {
	if mode == "once" {
		t, err := time.ParseInLocation("2006-01-02 15:04", cronStr, time.Local)
		if err != nil {
			return fmt.Errorf("时间格式不合法，请使用 YYYY-MM-DD HH:mm 格式")
		}
		if time.Until(t) <= 0 {
			return fmt.Errorf("目标时间已过")
		}
		return nil
	}
	if _, err := cronsched.ParseStandard(cronStr); err != nil {
		return fmt.Errorf("cron 表达式不合法")
	}
	return nil
}

// --- 笔记工具 ---

// updateNoteTool 允许 LLM 更新指定笔记的标题或内容。
var updateNoteTool = openai.ChatCompletionToolParam{
	Function: openai.FunctionDefinitionParam{
		Name:        "update_note",
		Description: openai.String("更新指定笔记的标题或内容。只需提供要修改的字段。"),
		Parameters: openai.FunctionParameters{
			"type": "object",
			"properties": map[string]any{
				"note_id": map[string]any{
					"type":        "integer",
					"description": "要更新的笔记ID",
				},
				"title": map[string]any{
					"type":        "string",
					"description": "新的标题（可选，不传则不变）",
				},
				"content": map[string]any{
					"type":        "string",
					"description": "新的内容（可选，不传则不变）",
				},
			},
			"required": []string{"note_id"},
		},
	},
}

// deleteNoteTool 允许 LLM 删除指定笔记。
var deleteNoteTool = openai.ChatCompletionToolParam{
	Function: openai.FunctionDefinitionParam{
		Name:        "delete_note",
		Description: openai.String("删除指定笔记。"),
		Parameters: openai.FunctionParameters{
			"type": "object",
			"properties": map[string]any{
				"note_id": map[string]any{
					"type":        "integer",
					"description": "要删除的笔记ID",
				},
			},
			"required": []string{"note_id"},
		},
	},
}

// --- 待办工具 ---

// listTodosTool 允许 LLM 列出用户的待办事项。
var listTodosTool = openai.ChatCompletionToolParam{
	Function: openai.FunctionDefinitionParam{
		Name:        "list_todos",
		Description: openai.String("列出用户的所有待办事项（含完成状态）。当需要了解用户有哪些待办、哪些未完成时使用此工具。"),
		Parameters: openai.FunctionParameters{
			"type":       "object",
			"properties": map[string]any{},
		},
	},
}

// updateTodoTool 允许 LLM 更新待办（如标记完成）。
var updateTodoTool = openai.ChatCompletionToolParam{
	Function: openai.FunctionDefinitionParam{
		Name:        "update_todo",
		Description: openai.String("更新指定待办事项。常用于标记完成/未完成，也可修改标题或内容。只需提供要修改的字段。"),
		Parameters: openai.FunctionParameters{
			"type": "object",
			"properties": map[string]any{
				"todo_id": map[string]any{
					"type":        "integer",
					"description": "要更新的待办ID",
				},
				"title": map[string]any{
					"type":        "string",
					"description": "新的标题（可选，不传则不变）",
				},
				"content": map[string]any{
					"type":        "string",
					"description": "新的内容（可选，不传则不变）",
				},
				"done": map[string]any{
					"type":        "boolean",
					"description": "是否完成（可选，不传则不变）",
				},
			},
			"required": []string{"todo_id"},
		},
	},
}

// deleteTodoTool 允许 LLM 删除指定待办。
var deleteTodoTool = openai.ChatCompletionToolParam{
	Function: openai.FunctionDefinitionParam{
		Name:        "delete_todo",
		Description: openai.String("删除指定待办事项。"),
		Parameters: openai.FunctionParameters{
			"type": "object",
			"properties": map[string]any{
				"todo_id": map[string]any{
					"type":        "integer",
					"description": "要删除的待办ID",
				},
			},
			"required": []string{"todo_id"},
		},
	},
}

// --- 计划工具 ---

// listPlansTool 允许 LLM 列出用户的计划。
var listPlansTool = openai.ChatCompletionToolParam{
	Function: openai.FunctionDefinitionParam{
		Name:        "list_plans",
		Description: openai.String("列出用户的所有计划（含标题和状态）。当需要了解用户有哪些计划时使用此工具。"),
		Parameters: openai.FunctionParameters{
			"type":       "object",
			"properties": map[string]any{},
		},
	},
}

// getPlanTool 允许 LLM 获取计划详情（含其关联的笔记和待办）。
var getPlanTool = openai.ChatCompletionToolParam{
	Function: openai.FunctionDefinitionParam{
		Name:        "get_plan",
		Description: openai.String("获取指定计划的详情，包括计划内容以及该计划下关联的所有笔记和待办。"),
		Parameters: openai.FunctionParameters{
			"type": "object",
			"properties": map[string]any{
				"plan_id": map[string]any{
					"type":        "integer",
					"description": "计划ID",
				},
			},
			"required": []string{"plan_id"},
		},
	},
}

// --- 定时任务工具 ---

// createScheduleTool 允许 LLM 创建定时任务。
var createScheduleTool = openai.ChatCompletionToolParam{
	Function: openai.FunctionDefinitionParam{
		Name:        "create_schedule",
		Description: openai.String("创建一个定时任务，到点后会以 Agent 方式自动执行指定提示词。当用户要求定时提醒、定时执行某事时使用。schedule_mode 为 cron 时按 cron 表达式周期执行；为 once 时在指定时间执行一次（格式 YYYY-MM-DD HH:mm）。"),
		Parameters: openai.FunctionParameters{
			"type": "object",
			"properties": map[string]any{
				"name": map[string]any{
					"type":        "string",
					"description": "任务名称",
				},
				"prompt": map[string]any{
					"type":        "string",
					"description": "任务到点时要执行的提示词/内容（Agent 会据此自动处理）",
				},
				"schedule_mode": map[string]any{
					"type":        "string",
					"description": "调度模式：cron（周期，默认）或 once（单次）",
				},
				"cron": map[string]any{
					"type":        "string",
					"description": "cron 模式下为 cron 表达式（如 \"0 9 * * *\" 表示每天9点）；once 模式下为具体时间（如 \"2026-06-15 09:00\"）",
				},
				"qq_notify": map[string]any{
					"type":        "boolean",
					"description": "执行结果是否通过 QQ 通知用户（可选，默认 false）",
				},
			},
			"required": []string{"name", "prompt", "cron"},
		},
	},
}

// listSchedulesTool 允许 LLM 列出用户的定时任务。
var listSchedulesTool = openai.ChatCompletionToolParam{
	Function: openai.FunctionDefinitionParam{
		Name:        "list_schedules",
		Description: openai.String("列出用户的所有定时任务。当需要了解用户有哪些定时任务、是否启用时使用此工具。"),
		Parameters: openai.FunctionParameters{
			"type":       "object",
			"properties": map[string]any{},
		},
	},
}

// --- 执行函数 ---

func executeUpdateNote(user *model.User, argsJSON string) (string, error) {
	var args struct {
		NoteID  uint    `json:"note_id"`
		Title   *string `json:"title"`
		Content *string `json:"content"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	updates := map[string]interface{}{}
	if args.Title != nil {
		updates["title"] = *args.Title
	}
	if args.Content != nil {
		updates["content"] = *args.Content
	}
	if len(updates) == 0 {
		return "", fmt.Errorf("未提供要更新的字段")
	}

	result := database.DB.Model(&model.Note{}).Where("id = ? AND user_id = ?", args.NoteID, user.ID).Updates(updates)
	if result.RowsAffected == 0 {
		return "", fmt.Errorf("笔记不存在 (id=%d)", args.NoteID)
	}
	return fmt.Sprintf("已更新笔记 (id=%d)", args.NoteID), nil
}

func executeDeleteNote(user *model.User, argsJSON string) (string, error) {
	var args struct {
		NoteID uint `json:"note_id"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	result := database.DB.Where("id = ? AND user_id = ?", args.NoteID, user.ID).Delete(&model.Note{})
	if result.RowsAffected == 0 {
		return "", fmt.Errorf("笔记不存在 (id=%d)", args.NoteID)
	}
	return fmt.Sprintf("已删除笔记 (id=%d)", args.NoteID), nil
}

func executeListTodos(user *model.User, argsJSON string) (string, error) {
	log.Printf("[tool] list_todos 调用")

	var todos []model.Todo
	database.DB.Where("user_id = ?", user.ID).
		Select("id, title, done, priority").
		Order("done asc, priority desc, created_at asc").Find(&todos)

	if len(todos) == 0 {
		return "用户暂无待办。", nil
	}

	type todoItem struct {
		ID       uint   `json:"id"`
		Title    string `json:"title"`
		Done     bool   `json:"done"`
		Priority int    `json:"priority"`
	}
	items := make([]todoItem, 0, len(todos))
	for _, t := range todos {
		items = append(items, todoItem{ID: t.ID, Title: t.Title, Done: t.Done, Priority: t.Priority})
	}

	resultJSON, _ := json.Marshal(items)
	return string(resultJSON), nil
}

func executeUpdateTodo(user *model.User, argsJSON string) (string, error) {
	var args struct {
		TodoID  uint    `json:"todo_id"`
		Title   *string `json:"title"`
		Content *string `json:"content"`
		Done    *bool   `json:"done"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	updates := map[string]interface{}{}
	if args.Title != nil {
		updates["title"] = *args.Title
	}
	if args.Content != nil {
		updates["content"] = *args.Content
	}
	if args.Done != nil {
		updates["done"] = *args.Done
	}
	if len(updates) == 0 {
		return "", fmt.Errorf("未提供要更新的字段")
	}

	result := database.DB.Model(&model.Todo{}).Where("id = ? AND user_id = ?", args.TodoID, user.ID).Updates(updates)
	if result.RowsAffected == 0 {
		return "", fmt.Errorf("待办不存在 (id=%d)", args.TodoID)
	}
	return fmt.Sprintf("已更新待办 (id=%d)", args.TodoID), nil
}

func executeDeleteTodo(user *model.User, argsJSON string) (string, error) {
	var args struct {
		TodoID uint `json:"todo_id"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	result := database.DB.Where("id = ? AND user_id = ?", args.TodoID, user.ID).Delete(&model.Todo{})
	if result.RowsAffected == 0 {
		return "", fmt.Errorf("待办不存在 (id=%d)", args.TodoID)
	}
	return fmt.Sprintf("已删除待办 (id=%d)", args.TodoID), nil
}

func executeListPlans(user *model.User, argsJSON string) (string, error) {
	log.Printf("[tool] list_plans 调用")

	var plans []model.Plan
	database.DB.Where("user_id = ?", user.ID).
		Select("id, title, status").Order("updated_at desc").Find(&plans)

	if len(plans) == 0 {
		return "用户暂无计划。", nil
	}

	type planItem struct {
		ID     uint   `json:"id"`
		Title  string `json:"title"`
		Status string `json:"status"`
	}
	items := make([]planItem, 0, len(plans))
	for _, p := range plans {
		items = append(items, planItem{ID: p.ID, Title: p.Title, Status: p.Status})
	}

	resultJSON, _ := json.Marshal(items)
	return string(resultJSON), nil
}

func executeGetPlan(user *model.User, argsJSON string) (string, error) {
	var args struct {
		PlanID uint `json:"plan_id"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	var plan model.Plan
	if err := database.DB.Where("id = ? AND user_id = ?", args.PlanID, user.ID).First(&plan).Error; err != nil {
		return "", fmt.Errorf("计划不存在 (id=%d)", args.PlanID)
	}

	var notes []model.Note
	database.DB.Where("plan_id = ? AND user_id = ?", args.PlanID, user.ID).
		Select("id, title").Order("updated_at desc").Find(&notes)

	var todos []model.Todo
	database.DB.Where("plan_id = ? AND user_id = ?", args.PlanID, user.ID).
		Select("id, title, done, priority").Order("done asc, priority desc, created_at asc").Find(&todos)

	type noteItem struct {
		ID    uint   `json:"id"`
		Title string `json:"title"`
	}
	type todoItem struct {
		ID       uint   `json:"id"`
		Title    string `json:"title"`
		Done     bool   `json:"done"`
		Priority int    `json:"priority"`
	}
	type planDetail struct {
		ID      uint       `json:"id"`
		Title   string     `json:"title"`
		Content string     `json:"content"`
		Status  string     `json:"status"`
		Notes   []noteItem `json:"notes"`
		Todos   []todoItem `json:"todos"`
	}

	detail := planDetail{
		ID:      plan.ID,
		Title:   plan.Title,
		Content: plan.Content,
		Status:  plan.Status,
		Notes:   make([]noteItem, 0, len(notes)),
		Todos:   make([]todoItem, 0, len(todos)),
	}
	for _, n := range notes {
		detail.Notes = append(detail.Notes, noteItem{ID: n.ID, Title: n.Title})
	}
	for _, t := range todos {
		detail.Todos = append(detail.Todos, todoItem{ID: t.ID, Title: t.Title, Done: t.Done, Priority: t.Priority})
	}

	resultJSON, _ := json.Marshal(detail)
	return string(resultJSON), nil
}

func executeCreateSchedule(user *model.User, argsJSON string) (string, error) {
	var args struct {
		Name         string `json:"name"`
		Prompt       string `json:"prompt"`
		ScheduleMode string `json:"schedule_mode"`
		Cron         string `json:"cron"`
		QQNotify     bool   `json:"qq_notify"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	if args.Name == "" || args.Prompt == "" || args.Cron == "" {
		return "", fmt.Errorf("name、prompt、cron 不能为空")
	}

	mode := args.ScheduleMode
	if mode == "" {
		mode = "cron"
	}
	if err := validateScheduleTime(mode, args.Cron); err != nil {
		return "", err
	}

	params, _ := json.Marshal(map[string]string{"prompt": args.Prompt})
	sched := model.Schedule{
		UserID:       user.ID,
		Name:         args.Name,
		Type:         "agent",
		ScheduleMode: mode,
		Cron:         args.Cron,
		Params:       string(params),
		Enabled:      true,
		QQNotify:     args.QQNotify,
	}
	if err := database.DB.Create(&sched).Error; err != nil {
		return "", fmt.Errorf("创建定时任务失败: %w", err)
	}

	if RegisterScheduleFunc != nil {
		if err := RegisterScheduleFunc(&sched); err != nil {
			log.Printf("[tool] create_schedule 注册任务失败: %v", err)
		}
	}

	return fmt.Sprintf("已创建定时任务「%s」(id=%d)", args.Name, sched.ID), nil
}

func executeListSchedules(user *model.User, argsJSON string) (string, error) {
	log.Printf("[tool] list_schedules 调用")

	var schedules []model.Schedule
	database.DB.Where("user_id = ?", user.ID).
		Select("id, name, schedule_mode, cron, params, enabled").
		Order("created_at desc").Find(&schedules)

	if len(schedules) == 0 {
		return "用户暂无定时任务。", nil
	}

	type schedItem struct {
		ID           uint   `json:"id"`
		Name         string `json:"name"`
		ScheduleMode string `json:"schedule_mode"`
		Cron         string `json:"cron"`
		Params       string `json:"params"`
		Enabled      bool   `json:"enabled"`
	}
	items := make([]schedItem, 0, len(schedules))
	for _, s := range schedules {
		items = append(items, schedItem{
			ID:           s.ID,
			Name:         s.Name,
			ScheduleMode: s.ScheduleMode,
			Cron:         s.Cron,
			Params:       s.Params,
			Enabled:      s.Enabled,
		})
	}

	resultJSON, _ := json.Marshal(items)
	return string(resultJSON), nil
}
