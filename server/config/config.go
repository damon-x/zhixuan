package config

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

// Config is the structure loaded from config.json.
type Config struct {
	ServerPort  string      `json:"server_port"`
	DataDir     string      `json:"data_dir"`
	DB          DBConfig    `json:"db"`
	TokenMaxAge int         `json:"token_max_age"`
	LLM         LLMConfig   `json:"llm"`
	Embedding   EmbedConfig `json:"embedding"`
	Rerank      RerankConfig `json:"rerank"`
	Chunk       ChunkConfig `json:"chunk"`
	Bocha       BochaConfig   `json:"bocha"`
	Vision      VisionConfig  `json:"vision"`
	Memory      MemoryConfig  `json:"memory"`
	Context     ContextConfig `json:"context"`
}

// ContextConfig 控制会话上下文压缩行为。
type ContextConfig struct {
	CompressThreshold int `json:"compress_threshold"` // total_tokens 超过该值则触发压缩
	SummaryMaxChars   int `json:"summary_max_chars"`  // 压缩摘要的最大字符数
}

type DBConfig struct {
	Type string `json:"type"` // "sqlite" or "mysql"
	Path string `json:"path"` // SQLite file path
	DSN  string `json:"dsn"`  // MySQL DSN
}

type LLMConfig struct {
	APIKey  string   `json:"api_key"`
	BaseURL string   `json:"base_url"`
	Model   string   `json:"model"`
	Models  []string `json:"models"`
}

type EmbedConfig struct {
	APIKey     string `json:"api_key"`     // empty = fallback to LLM.APIKey
	BaseURL    string `json:"base_url"`    // empty = fallback to LLM.BaseURL
	Model      string `json:"model"`
	Dimensions int    `json:"dimensions"`
}

type RerankConfig struct {
	APIKey  string `json:"api_key"`  // empty = fallback to LLM.APIKey
	BaseURL string `json:"base_url"`
	Model   string `json:"model"`
}

type ChunkConfig struct {
	WindowSize int `json:"window_size"`
	Overlap    int `json:"overlap"`
}

type BochaConfig struct {
	APIKey  string `json:"api_key"`
	BaseURL string `json:"base_url"`
}

type VisionConfig struct {
	APIKey  string   `json:"api_key"`   // empty = fallback LLM.APIKey
	BaseURL string   `json:"base_url"`  // empty = fallback LLM.BaseURL
	Models  []string `json:"models"`
}

type MemoryConfig struct {
	RecallThreshold float64 `json:"recall_threshold"`
	BatchRounds     int     `json:"batch_rounds"`
}

// Package-level variables — keep config.XXX access pattern unchanged.
var (
	ServerPort string
	DataDir    string
	DBType     string
	DBPath     string
	DBDSN      string
	TokenMaxAge int

	LLMAPIKey  string
	LLMBaseURL string
	LLMModel   string
	LLMModels  []string

	EmbeddingAPIKey     string
	EmbeddingBaseURL    string
	EmbeddingModel      string
	EmbeddingDimensions int

	RerankAPIKey  string
	RerankBaseURL string
	RerankModel   string

	ChunkWindowSize int
	ChunkOverlap    int

	BochaAPIKey  string
	BochaBaseURL string

	VisionAPIKey  string
	VisionBaseURL string
	VisionModels  []string

	MemoryRecallThreshold float64
	MemoryBatchRounds     int

	ContextCompressThreshold int
	ContextSummaryMaxChars   int
)

// Load reads config.json and sets package vars.
// config.json 不存在时不会自动生成 —— 请复制 config.example.json 为 config.json 并填入配置。
func Load() {
	data, err := os.ReadFile("config.json")
	if err != nil {
		log.Fatal("config.json 不存在或无法读取，请复制 config.example.json 为 config.json 并填入配置后重试")
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		log.Fatalf("config.json 解析失败: %v", err)
	}
	log.Println("已加载 config.json")

	// Apply to package-level vars
	ServerPort = cfg.ServerPort
	DataDir = cfg.DataDir
	DBType = cfg.DB.Type
	DBPath = cfg.DB.Path
	DBDSN = cfg.DB.DSN
	TokenMaxAge = cfg.TokenMaxAge

	LLMAPIKey = cfg.LLM.APIKey
	LLMBaseURL = cfg.LLM.BaseURL
	LLMModel = cfg.LLM.Model
	if len(cfg.LLM.Models) > 0 {
		LLMModels = cfg.LLM.Models
	} else {
		LLMModels = []string{cfg.LLM.Model}
	}

	EmbeddingAPIKey = cfg.Embedding.APIKey
	if EmbeddingAPIKey == "" {
		EmbeddingAPIKey = LLMAPIKey
	}
	EmbeddingBaseURL = cfg.Embedding.BaseURL
	if EmbeddingBaseURL == "" {
		EmbeddingBaseURL = LLMBaseURL
	}
	EmbeddingModel = cfg.Embedding.Model
	EmbeddingDimensions = cfg.Embedding.Dimensions

	RerankAPIKey = cfg.Rerank.APIKey
	if RerankAPIKey == "" {
		RerankAPIKey = LLMAPIKey
	}
	RerankBaseURL = cfg.Rerank.BaseURL
	RerankModel = cfg.Rerank.Model

	ChunkWindowSize = cfg.Chunk.WindowSize
	ChunkOverlap = cfg.Chunk.Overlap

	BochaAPIKey = cfg.Bocha.APIKey
	BochaBaseURL = cfg.Bocha.BaseURL

	VisionAPIKey = cfg.Vision.APIKey
	if VisionAPIKey == "" {
		VisionAPIKey = LLMAPIKey
	}
	VisionBaseURL = cfg.Vision.BaseURL
	if VisionBaseURL == "" {
		VisionBaseURL = LLMBaseURL
	}
	if len(cfg.Vision.Models) > 0 {
		VisionModels = cfg.Vision.Models
	}

	MemoryRecallThreshold = cfg.Memory.RecallThreshold
	MemoryBatchRounds = cfg.Memory.BatchRounds

	ContextCompressThreshold = cfg.Context.CompressThreshold
	ContextSummaryMaxChars = cfg.Context.SummaryMaxChars
}

// KBDir returns the knowledge bases directory path.
func KBDir() string {
	return filepath.Join(DataDir, "knowledge_bases")
}

// ContextCacheDir returns the context cache directory path.
func ContextCacheDir() string {
	return filepath.Join(DataDir, "context_cache")
}

// UploadDir returns the uploads directory path.
func UploadDir() string {
	return filepath.Join(DataDir, "uploads")
}

// WorkspaceDir returns the user workspace directory path.
func WorkspaceDir() string {
	return filepath.Join(DataDir, "workspace")
}

// DictPath returns the jieba dictionary file path.
func DictPath() string {
	return filepath.Join(DataDir, "data", "dict.txt")
}

// AgentDBDir returns the directory for per-user agent SQLite databases.
func AgentDBDir() string {
	return filepath.Join(DataDir, "agent_db")
}

// ChatHistoryDir returns the root directory for file-based chat history persistence.
// Layout: <ChatHistoryDir>/<sessionID>/<date>.jsonl
func ChatHistoryDir() string {
	return filepath.Join(DataDir, "chat_history")
}

// AgentDBPath returns the SQLite file path for the given user's agent database.
func AgentDBPath(userID uint) string {
	return filepath.Join(AgentDBDir(), fmt.Sprintf("agent_%d.db", userID))
}

// MemoryDir returns the root directory for memory vector stores.
// Layout: <MemoryDir>/<userID>/vectors.db
func MemoryDir() string {
	return filepath.Join(DataDir, "memory")
}

// SessionStateDir returns the directory for per-session runtime state
// (recall window IDs + memory-agent checkpoint).
// Layout: <SessionStateDir>/<sessionID>.json
func SessionStateDir() string {
	return filepath.Join(DataDir, "session_state")
}
