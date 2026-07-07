package handler

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"zhixuan/server/config"

	"github.com/gin-gonic/gin"
)

func GetResource(c *gin.Context) {
	user, ok := GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 1, "msg": "未登录"})
		return
	}

	path := c.Query("path")
	if path == "" {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "缺少 path 参数"})
		return
	}

	// Parse path format: source@relative_path
	atIdx := strings.Index(path, "@")
	if atIdx == -1 {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "path 格式错误"})
		return
	}

	source := path[:atIdx]
	relativePath := path[atIdx+1:]

	var baseDir string
	switch source {
	case "knowledge":
		baseDir = filepath.Join(config.KBDir(), fmt.Sprintf("%d", user.ID))
	case "upload":
		baseDir = config.UploadDir()
	default:
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "不支持的来源"})
		return
	}

	// Security: prevent path traversal
	cleanRel := filepath.Clean(relativePath)
	if strings.Contains(cleanRel, "..") {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "路径不合法"})
		return
	}

	filePath := filepath.Join(baseDir, cleanRel)
	data, err := os.ReadFile(filePath)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "文件不存在"})
		return
	}

	ext := strings.ToLower(filepath.Ext(cleanRel))
	contentType := "application/octet-stream"
	switch ext {
	case ".jpg", ".jpeg":
		contentType = "image/jpeg"
	case ".png":
		contentType = "image/png"
	case ".gif":
		contentType = "image/gif"
	}

	c.Data(http.StatusOK, contentType, data)
}
