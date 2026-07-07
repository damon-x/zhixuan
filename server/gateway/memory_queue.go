package gateway

import (
	stdctx "context"
	"log"
	"sync"
	"time"

	"zhixuan/server/llm"
)

// memoryTask 一个待执行的记忆整理任务
type memoryTask struct {
	userID    uint
	sessionID string
	msgs      []llm.Message
}

// memoryQueue 单用户的记忆 agent 串行队列。
// 容量上限 2：1 运行 + 1 等待。
// 新任务到达时若已有任务运行中，则占据等待槽（替换旧 pending，踢出中间任务），
// 保证记忆整理按轮次顺序执行，避免并行写入重复记忆。
type memoryQueue struct {
	mu      sync.Mutex
	running bool
	pending *memoryTask
}

// submit 提交一个任务。若空闲则立即启动；否则占据等待槽（替换旧任务）。
func (q *memoryQueue) submit(task memoryTask) {
	q.mu.Lock()
	if !q.running {
		q.running = true
		q.mu.Unlock()
		go q.loop(task)
		return
	}
	// 占据等待槽，旧 pending 被替换踢出
	q.pending = &task
	q.mu.Unlock()
}

// loop 串行执行任务。跑完一个后检查 pending：有就继续，没有就退出。
func (q *memoryQueue) loop(first memoryTask) {
	task := first
	for {
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("[memory-agent] panic user=%d: %v", task.userID, r)
				}
			}()
			ctx, cancel := stdctx.WithTimeout(stdctx.Background(), 120*time.Second)
			defer cancel()
			if err := runMemoryAgent(ctx, task.userID, task.sessionID, task.msgs); err != nil {
				log.Printf("[memory-agent] 失败 user=%d: %v", task.userID, err)
			}
		}()

		q.mu.Lock()
		next := q.pending
		q.pending = nil
		if next == nil {
			q.running = false
			q.mu.Unlock()
			return
		}
		task = *next
		q.mu.Unlock()
	}
}

// memoryScheduler 全局单例，按 userID 分配独立队列。
type memoryScheduler struct {
	mu     sync.Mutex
	queues map[uint]*memoryQueue
}

var memorySched = &memoryScheduler{queues: make(map[uint]*memoryQueue)}

func (s *memoryScheduler) submit(task memoryTask) {
	s.mu.Lock()
	q, ok := s.queues[task.userID]
	if !ok {
		q = &memoryQueue{}
		s.queues[task.userID] = q
	}
	s.mu.Unlock()
	q.submit(task)
}
