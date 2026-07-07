package database

import (
	"os"
	"path/filepath"

	"zhixuan/server/config"
	"zhixuan/server/database/dialect"
	"zhixuan/server/model"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Init() error {
	var dialector gorm.Dialector
	switch config.DBType {
	case "mysql":
		dialector = mysql.Open(config.DBDSN)
	default:
		// 确保库文件父目录存在（首次运行时 biz_data 可能尚未创建，否则 SQLite 报 CANTOPEN）。
		if err := os.MkdirAll(filepath.Dir(config.DBPath), 0755); err != nil {
			return err
		}
		dialector = dialect.Open(config.DBPath)
	}

	var err error
	DB, err = gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		return err
	}
	if err := DB.AutoMigrate(&model.User{}, &model.Note{}, &model.Todo{}, &model.Plan{}, &model.Chat{}, &model.Session{}, &model.Schedule{}, &model.ScheduleLog{}, &model.Token{}, &model.Memory{}, &model.Skill{}); err != nil {
		return err
	}
	return nil
}
