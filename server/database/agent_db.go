package database

import (
	"fmt"
	"os"
	"sync"

	"zhixuan/server/config"
	"zhixuan/server/database/dialect"

	"gorm.io/gorm"
)

// agentDBs caches per-user *gorm.DB handles for the agent's private SQLite databases.
var agentDBs sync.Map // map[uint]*gorm.DB

// agentDBMu serializes Open calls per userID to avoid racing on file creation.
var agentDBMu sync.Mutex

// GetAgentDB returns the gorm.DB handle for the given user's private SQLite database.
// The database file is created on first use (no AutoMigrate — the agent manages its own schema).
func GetAgentDB(userID uint) (*gorm.DB, error) {
	if v, ok := agentDBs.Load(userID); ok {
		return v.(*gorm.DB), nil
	}

	agentDBMu.Lock()
	defer agentDBMu.Unlock()

	// Double-check after acquiring the lock.
	if v, ok := agentDBs.Load(userID); ok {
		return v.(*gorm.DB), nil
	}

	dir := config.AgentDBDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("创建 agent 数据库目录失败: %w", err)
	}

	dbPath := config.AgentDBPath(userID)
	dsn := fmt.Sprintf("file:%s?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=on", dbPath)

	db, err := gorm.Open(dialect.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("打开 agent 数据库失败: %w", err)
	}

	// Store the actual instance returned by gorm.Open to avoid the underlying
	// *sql.DB being wrapped twice.
	agentDBs.Store(userID, db)
	return db, nil
}
