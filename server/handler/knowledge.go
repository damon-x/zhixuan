package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"zhixuan/server/config"
	"zhixuan/server/kbindex"
	"zhixuan/server/vision"

	"github.com/gin-gonic/gin"
)

func kbBaseDir() string { return config.KBDir() }

// KBConfig is the metadata stored in config.json for each knowledge base.
type KBConfig struct {
	Description string `json:"description"`
}

func init() {
	os.MkdirAll(kbBaseDir(), 0755)
}

func userKBDir(userID uint) string {
	return filepath.Join(kbBaseDir(), fmt.Sprintf("%d", userID))
}

func readKBConfig(dir string) KBConfig {
	var cfg KBConfig
	data, err := os.ReadFile(filepath.Join(dir, "config.json"))
	if err == nil {
		json.Unmarshal(data, &cfg)
	}
	return cfg
}

func writeKBConfig(dir string, cfg KBConfig) error {
	data, _ := json.MarshalIndent(cfg, "", "  ")
	return os.WriteFile(filepath.Join(dir, "config.json"), data, 0644)
}

func isValidName(name string) bool {
	if name == "" {
		return false
	}
	if strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return false
	}
	if name == "." || name == ".." {
		return false
	}
	if strings.Contains(name, "..") {
		return false
	}
	return true
}

func CreateKB(c *gin.Context) {
	user, ok := GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 1, "msg": "未登录"})
		return
	}

	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "参数错误"})
		return
	}

	if !isValidName(req.Name) {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "名称不合法"})
		return
	}

	dir := filepath.Join(userKBDir(user.ID), req.Name)
	if _, err := os.Stat(dir); err == nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "知识库已存在"})
		return
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "创建失败"})
		return
	}

	writeKBConfig(dir, KBConfig{Description: req.Description})

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success"})
}

func UpdateKB(c *gin.Context) {
	user, ok := GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 1, "msg": "未登录"})
		return
	}

	name := c.Param("name")
	if !isValidName(name) {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "名称不合法"})
		return
	}

	var req struct {
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "参数错误"})
		return
	}

	dir := filepath.Join(userKBDir(user.ID), name)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "知识库不存在"})
		return
	}

	if err := writeKBConfig(dir, KBConfig{Description: req.Description}); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "更新失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success"})
}

func ListKB(c *gin.Context) {
	user, ok := GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 1, "msg": "未登录"})
		return
	}

	userDir := userKBDir(user.ID)
	entries, err := os.ReadDir(userDir)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success", "data": []interface{}{}})
		return
	}

	var result []gin.H
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		kbPath := filepath.Join(userDir, entry.Name())
		cfg := readKBConfig(kbPath)

		// Count docs (excluding config.json and .index)
		docEntries, _ := os.ReadDir(kbPath)
		docCount := 0
		for _, de := range docEntries {
			if !de.IsDir() && de.Name() != "config.json" && !strings.HasSuffix(de.Name(), ".ocr") {
				docCount++
			}
		}
		result = append(result, gin.H{
			"name":        entry.Name(),
			"description": cfg.Description,
			"doc_count":   docCount,
			"updated_at":  info.ModTime().Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	if result == nil {
		result = []gin.H{}
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success", "data": result})
}

func DeleteKB(c *gin.Context) {
	user, ok := GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 1, "msg": "未登录"})
		return
	}

	name := c.Param("name")
	if !isValidName(name) {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "名称不合法"})
		return
	}

	dir := filepath.Join(userKBDir(user.ID), name)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "知识库不存在"})
		return
	}

	kbindex.Get().DeleteKB(user.ID, name)

	if err := os.RemoveAll(dir); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "删除失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success"})
}

func ListDocs(c *gin.Context) {
	user, ok := GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 1, "msg": "未登录"})
		return
	}

	kbName := c.Param("name")
	if !isValidName(kbName) {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "名称不合法"})
		return
	}

	kbPath := filepath.Join(userKBDir(user.ID), kbName)
	entries, err := os.ReadDir(kbPath)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success", "data": []interface{}{}})
		return
	}

	var result []gin.H
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		// Skip config.json and .ocr files
		if entry.Name() == "config.json" || strings.HasSuffix(entry.Name(), ".ocr") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		result = append(result, gin.H{
			"name":       entry.Name(),
			"size":       info.Size(),
			"updated_at": info.ModTime().Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	if result == nil {
		result = []gin.H{}
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success", "data": result})
}

func UploadDoc(c *gin.Context) {
	user, ok := GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 1, "msg": "未登录"})
		return
	}

	kbName := c.Param("name")
	if !isValidName(kbName) {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "名称不合法"})
		return
	}

	kbPath := filepath.Join(userKBDir(user.ID), kbName)
	if _, err := os.Stat(kbPath); os.IsNotExist(err) {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "知识库不存在"})
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "请选择文件"})
		return
	}
	defer file.Close()

	if !isValidName(header.Filename) {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "文件名不合法"})
		return
	}

	dst := filepath.Join(kbPath, header.Filename)
	if err := c.SaveUploadedFile(header, dst); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "上传失败"})
		return
	}

	// Async index for text files
	ext := strings.ToLower(filepath.Ext(header.Filename))
	switch ext {
	case ".txt", ".md":
		go func() {
			if err := kbindex.Get().IndexDocument(context.Background(), user.ID, kbName, header.Filename); err != nil {
				fmt.Printf("[knowledge] index error: %v\n", err)
			}
		}()
	case ".jpg", ".jpeg", ".png", ".gif", ".webp":
		go func() {
			imgData, err := os.ReadFile(dst)
			if err != nil {
				fmt.Printf("[knowledge] read image error: %v\n", err)
				return
			}
			ocrText, err := vision.Describe(context.Background(), imgData, ext)
			if err != nil {
				fmt.Printf("[knowledge] vision describe error: %v\n", err)
				return
			}
			// Save OCR text file
			ocrPath := dst + ".ocr"
			os.WriteFile(ocrPath, []byte(ocrText), 0644)
			if err := kbindex.Get().IndexImage(context.Background(), user.ID, kbName, header.Filename, ocrText); err != nil {
				fmt.Printf("[knowledge] index image error: %v\n", err)
			}
		}()
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success"})
}

// PreviewDoc 以 inline 方式返回知识库原始文件，供前端在线预览。
// 按当前用户隔离目录，复用名字校验防目录穿越；Content-Type 由扩展名推断。
func PreviewDoc(c *gin.Context) {
	user, ok := GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 1, "msg": "未登录"})
		return
	}

	kbName := c.Param("name")
	docName := c.Param("doc")
	if !isValidName(kbName) || !isValidName(docName) {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "名称不合法"})
		return
	}

	docPath := filepath.Join(userKBDir(user.ID), kbName, docName)
	if _, err := os.Stat(docPath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"code": 1, "msg": "文件不存在"})
		return
	}

	c.Header("Content-Disposition", "inline")
	c.File(docPath)
}

func DeleteDoc(c *gin.Context) {
	user, ok := GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 1, "msg": "未登录"})
		return
	}

	kbName := c.Param("name")
	docName := c.Param("doc")
	if !isValidName(kbName) || !isValidName(docName) {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "名称不合法"})
		return
	}

	docPath := filepath.Join(userKBDir(user.ID), kbName, docName)
	if _, err := os.Stat(docPath); os.IsNotExist(err) {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "文件不存在"})
		return
	}

	// Delete from index before removing file
	kbindex.Get().DeleteDocument(c.Request.Context(), user.ID, kbName, docName)

	if err := os.Remove(docPath); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "删除失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success"})
}
