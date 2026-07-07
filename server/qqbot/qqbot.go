package qqbot

import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"zhixuan/server/config"
)

const (
	tokenURL = "https://bots.qq.com/app/getAppAccessToken"
	apiBase  = "https://api.sgroup.qq.com"
	intentC2C = 1 << 25

	// md5_10m 计算时取文件前 N 字节（与开放平台协议一致）
	md5_10m_size = 10002432
)

// BindState tracks an in-progress binding.
type BindState struct {
	Code     string
	UserID   uint
	AppID    string
	Secret   string
	Done     bool
	OpenID   string
	cancel   func()
}

var (
	mu       sync.Mutex
	bindings = make(map[uint]*BindState) // key: userID

	chatMu        sync.Mutex
	chatListeners = make(map[uint]*ChatListener)
)

// StartBinding generates a 4-digit code, starts a WebSocket listener, returns the code.
func StartBinding(userID uint, appID, secret string) (string, error) {
	mu.Lock()
	// Cancel any existing binding for this user
	if old, ok := bindings[userID]; ok {
		if old.cancel != nil {
			old.cancel()
		}
	}
	code := fmt.Sprintf("%04d", rand.Intn(10000))
	ctx := &BindState{
		Code:   code,
		UserID: userID,
		AppID:  appID,
		Secret: secret,
	}
	bindings[userID] = ctx
	mu.Unlock()

	cancel := startListener(ctx)
	mu.Lock()
	ctx.cancel = cancel
	mu.Unlock()

	return code, nil
}

// CheckBinding returns (done, openID, nil).
func CheckBinding(userID uint) (bool, string) {
	mu.Lock()
	defer mu.Unlock()
	b, ok := bindings[userID]
	if !ok {
		return false, ""
	}
	return b.Done, b.OpenID
}

// CancelBinding stops any in-progress binding for the user.
func CancelBinding(userID uint) {
	mu.Lock()
	defer mu.Unlock()
	if b, ok := bindings[userID]; ok {
		if b.cancel != nil {
			b.cancel()
		}
		delete(bindings, userID)
	}
}

// --- QQ Bot API helpers ---

type tokenResponse struct {
	AccessToken string `json:"access_token"`
}

func getAccessToken(appID, secret string) (string, error) {
	body, _ := json.Marshal(map[string]string{"appId": appID, "clientSecret": secret})
	resp, err := http.Post(tokenURL, "application/json", strings.NewReader(string(body)))
	if err != nil {
		return "", fmt.Errorf("get token: %w", err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return "", fmt.Errorf("decode token: %w", err)
	}
	token, _ := raw["access_token"].(string)
	if token == "" {
		return "", fmt.Errorf("empty token: %s", string(data))
	}
	return token, nil
}

type gatewayResponse struct {
	URL string `json:"url"`
}

func getGatewayURL(token string) (string, error) {
	req, _ := http.NewRequest("GET", apiBase+"/gateway", nil)
	req.Header.Set("Authorization", "QQBot "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("get gateway: %w", err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	var gr gatewayResponse
	if err := json.Unmarshal(data, &gr); err != nil {
		return "", fmt.Errorf("decode gateway: %w", err)
	}
	if gr.URL == "" {
		return "", fmt.Errorf("empty gateway URL: %s", string(data))
	}
	return gr.URL, nil
}

// SendMsg sends a C2C text message to the given openid.
func SendMsg(appID, secret, openid, content string) error {
	token, err := getAccessToken(appID, secret)
	if err != nil {
		return err
	}
	body, _ := json.Marshal(map[string]any{"content": content, "msg_type": 0})
	url := fmt.Sprintf("%s/v2/users/%s/messages", apiBase, openid)
	req, _ := http.NewRequest("POST", url, strings.NewReader(string(body)))
	req.Header.Set("Authorization", "QQBot "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("send msg: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		data, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("send msg error %d: %s", resp.StatusCode, string(data))
	}
	return nil
}

// --- WebSocket listener ---

func startListener(ctx *BindState) func() {
	done := make(chan struct{})
	go func() {
		for {
			select {
			case <-done:
				return
			default:
			}
			if err := listenOnce(ctx, done); err != nil {
				log.Printf("[qqbot] listener error: %v", err)
			}
			mu.Lock()
			if ctx.Done || ctx.cancel == nil {
				mu.Unlock()
				return
			}
			mu.Unlock()
			time.Sleep(3 * time.Second)
		}
	}()
	return func() { close(done) }
}

func listenOnce(ctx *BindState, done chan struct{}) error {
	token, err := getAccessToken(ctx.AppID, ctx.Secret)
	if err != nil {
		return err
	}
	gwURL, err := getGatewayURL(token)
	if err != nil {
		return err
	}

	log.Printf("[qqbot] 连接 gateway: %s", gwURL)
	ws, _, err := websocket.DefaultDialer.Dial(gwURL, nil)
	if err != nil {
		return fmt.Errorf("ws dial: %w", err)
	}
	defer ws.Close()

	// Read loop
	var heartbeatInterval int
	var sessionID string
	var lastSeq int

	for {
		select {
		case <-done:
			return nil
		default:
		}

		_, msg, err := ws.ReadMessage()
		if err != nil {
			return fmt.Errorf("ws read: %w", err)
		}

		var packet map[string]any
		json.Unmarshal(msg, &packet)

		op := int(packet["op"].(float64))
		d, _ := packet["d"].(map[string]any)
		s, hasS := packet["s"]
		t, _ := packet["t"].(string)

		// Log all dispatch events
		if op == 0 {
			log.Printf("[qqbot] 收到事件: %s", t)
		}

		if hasS {
			lastSeq = int(s.(float64))
		}

		switch op {
		case 10: // Hello
			heartbeatInterval = int(d["heartbeat_interval"].(float64))
			// Identify
			identify := map[string]any{
				"op": 2,
				"d": map[string]any{
					"token":   "QQBot " + token,
					"intents": intentC2C,
					"shard":   []int{0, 1},
				},
			}
			ws.WriteJSON(identify)
			// Start heartbeat
			go func() {
				ticker := time.NewTicker(time.Duration(heartbeatInterval) * time.Millisecond)
				defer ticker.Stop()
				for {
					select {
					case <-ticker.C:
						ws.WriteJSON(map[string]any{"op": 1, "d": lastSeq})
					case <-done:
						return
					}
				}
			}()
		case 11: // Heartbeat ACK
			continue
		case 7: // Reconnect
			log.Printf("[qqbot] 服务端要求重连")
			return nil
		case 9: // Invalid session
			log.Printf("[qqbot] 无效 session")
			return nil
		case 0: // Dispatch
			if t == "READY" {
				sessionID = d["session_id"].(string)
				log.Printf("[qqbot] READY session_id=%s", sessionID)
				continue
			}
			if t == "C2C_MESSAGE_CREATE" {
				author, _ := d["author"].(map[string]any)
				if author == nil {
					continue
				}
				openid, _ := author["user_openid"].(string)
				content, _ := d["content"].(string)
				content = strings.TrimSpace(content)
				log.Printf("[qqbot] C2C_MESSAGE_CREATE openid=%s content=%q", openid, content)

				// Check if content matches the binding code
				mu.Lock()
				if content == ctx.Code && !ctx.Done {
					ctx.Done = true
					ctx.OpenID = openid
					log.Printf("[qqbot] 绑定成功: user=%d openid=%s", ctx.UserID, openid)
				}
				mu.Unlock()
			}
		}
	}
}

// --- Chat Listener (persistent, for QQ chat conversations) ---

// ChatListener manages a persistent WebSocket connection for QQ chat.
type ChatListener struct {
	userID uint
	appID  string
	secret string
	done   chan struct{}
	cancel func()
}

// StartChatListener starts a persistent chat listener for the given user.
func StartChatListener(userID uint, appID, secret string, onMessage func(content string, imageRefs []string)) error {
	chatMu.Lock()
	// Cancel existing listener
	if old, ok := chatListeners[userID]; ok {
		if old.cancel != nil {
			old.cancel()
		}
	}
	chatMu.Unlock()

	cl := &ChatListener{
		userID: userID,
		appID:  appID,
		secret: secret,
	}

	done := make(chan struct{})
	cl.done = done
	cancel := startChatListener(cl, onMessage, done)
	cl.cancel = cancel

	chatMu.Lock()
	chatListeners[userID] = cl
	chatMu.Unlock()

	return nil
}

// StopChatListener stops the chat listener for the given user.
func StopChatListener(userID uint) {
	chatMu.Lock()
	defer chatMu.Unlock()
	if cl, ok := chatListeners[userID]; ok {
		if cl.cancel != nil {
			cl.cancel()
		}
		delete(chatListeners, userID)
	}
}

// IsChatListenerRunning returns whether a chat listener is active for the given user.
func IsChatListenerRunning(userID uint) bool {
	chatMu.Lock()
	defer chatMu.Unlock()
	_, ok := chatListeners[userID]
	return ok
}

func startChatListener(cl *ChatListener, onMessage func(content string, imageRefs []string), done chan struct{}) func() {
	go func() {
		for {
			select {
			case <-done:
				return
			default:
			}
			if err := listenForChat(cl, onMessage, done); err != nil {
				log.Printf("[qqbot-chat] listener error (user=%d): %v", cl.userID, err)
			}
			select {
			case <-done:
				return
			default:
			}
			time.Sleep(3 * time.Second)
		}
	}()
	return func() { close(done) }
}

func listenForChat(cl *ChatListener, onMessage func(content string, imageRefs []string), done chan struct{}) error {
	token, err := getAccessToken(cl.appID, cl.secret)
	if err != nil {
		return err
	}
	gwURL, err := getGatewayURL(token)
	if err != nil {
		return err
	}

	log.Printf("[qqbot-chat] 连接 gateway (user=%d): %s", cl.userID, gwURL)
	ws, _, err := websocket.DefaultDialer.Dial(gwURL, nil)
	if err != nil {
		return fmt.Errorf("ws dial: %w", err)
	}
	defer ws.Close()

	var heartbeatInterval int
	var lastSeq int

	for {
		select {
		case <-done:
			return nil
		default:
		}

		_, msg, err := ws.ReadMessage()
		if err != nil {
			return fmt.Errorf("ws read: %w", err)
		}

		var packet map[string]any
		json.Unmarshal(msg, &packet)

		op := int(packet["op"].(float64))
		d, _ := packet["d"].(map[string]any)
		s, hasS := packet["s"]
		t, _ := packet["t"].(string)

		if op == 0 {
			log.Printf("[qqbot-chat] 收到事件 (user=%d): %s", cl.userID, t)
		}

		if hasS {
			lastSeq = int(s.(float64))
		}

		switch op {
		case 10: // Hello
			heartbeatInterval = int(d["heartbeat_interval"].(float64))
			identify := map[string]any{
				"op": 2,
				"d": map[string]any{
					"token":   "QQBot " + token,
					"intents": intentC2C,
					"shard":   []int{0, 1},
				},
			}
			ws.WriteJSON(identify)
			go func() {
				ticker := time.NewTicker(time.Duration(heartbeatInterval) * time.Millisecond)
				defer ticker.Stop()
				for {
					select {
					case <-ticker.C:
						ws.WriteJSON(map[string]any{"op": 1, "d": lastSeq})
					case <-done:
						return
					}
				}
			}()
		case 11: // Heartbeat ACK
			continue
		case 7: // Reconnect
			log.Printf("[qqbot-chat] 服务端要求重连 (user=%d)", cl.userID)
			return nil
		case 9: // Invalid session
			log.Printf("[qqbot-chat] 无效 session (user=%d)", cl.userID)
			return nil
		case 0: // Dispatch
			if t == "READY" {
				log.Printf("[qqbot-chat] READY (user=%d)", cl.userID)
				continue
			}
			if t == "C2C_MESSAGE_CREATE" {
				author, _ := d["author"].(map[string]any)
				if author == nil {
					continue
				}
				content, _ := d["content"].(string)
				content = strings.TrimSpace(content)
				log.Printf("[qqbot-chat] C2C_MESSAGE_CREATE (user=%d) content=%q", cl.userID, content)

				// 解析并下载图片附件，失败跳过不影响文本
				var imageRefs []string
				if attachments, ok := d["attachments"].([]any); ok {
					for _, att := range attachments {
						a, ok := att.(map[string]any)
						if !ok {
							continue
						}
						ct, _ := a["content_type"].(string)
						if !strings.HasPrefix(ct, "image/") {
							continue // 本期仅处理图片
						}
						url, _ := a["url"].(string)
						if url == "" {
							continue
						}
						if strings.HasPrefix(url, "//") {
							url = "https:" + url
						}
						ref, err := downloadImage(url, cl.userID, ct)
						if err != nil {
							log.Printf("[qqbot-chat] 下载图片失败 (user=%d): %v", cl.userID, err)
							continue
						}
						imageRefs = append(imageRefs, ref)
					}
				}

				if content != "" || len(imageRefs) > 0 {
					onMessage(content, imageRefs)
				}
			}
		}
	}
}

// --- 图片下载（收图）---

// downloadImage 下载图片到 config.UploadDir()/{userID}/{filename}，返回 upload@ 索引。
// 与 web 端 UploadChatImage 的存储约定一致，使 agent 的 describe_image 能直接理解。
func downloadImage(url string, userID uint, contentType string) (string, error) {
	ext := imageExtFromContentType(contentType)
	userDir := filepath.Join(config.UploadDir(), fmt.Sprintf("%d", userID))
	if err := os.MkdirAll(userDir, 0755); err != nil {
		return "", fmt.Errorf("mkdir: %w", err)
	}
	// 多图可能同一毫秒到达，追加随机后缀避免重名
	filename := fmt.Sprintf("%d%04d%s", time.Now().UnixMilli(), rand.Intn(10000), ext)
	savePath := filepath.Join(userDir, filename)

	client := &http.Client{Timeout: 2 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download status %d", resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read body: %w", err)
	}
	if err := os.WriteFile(savePath, data, 0644); err != nil {
		return "", fmt.Errorf("write: %w", err)
	}
	return fmt.Sprintf("upload@%d/%s", userID, filename), nil
}

func imageExtFromContentType(ct string) string {
	switch ct {
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	default:
		return ".jpg"
	}
}

// --- 分片上传发图（发图）---
//
// 封装 QQ 开放平台 C2C 富媒体分片上传 4 步：
//  1. upload_prepare → upload_id + block_size + 分片预签名链接
//  2. 每个 part: PUT 预签名 URL → upload_part_finish
//  3. files(完成上传) → file_info
//  4. messages(msg_type:7, media.file_info)

// SendImage 向给定 openid 发送本地图片。
func SendImage(appID, secret, openid, localPath string) error {
	fileName := filepath.Base(localPath)
	md5Hex, sha1Hex, md5_10m, fileSize, err := computeHashes(localPath)
	if err != nil {
		return fmt.Errorf("compute hashes: %w", err)
	}

	token, err := getAccessToken(appID, secret)
	if err != nil {
		return err
	}
	prep, err := uploadPrepare(token, openid, fileName, fileSize, md5Hex, sha1Hex, md5_10m)
	if err != nil {
		return fmt.Errorf("upload prepare: %w", err)
	}

	blockSize := toInt64(prep.BlockSize)
	if blockSize <= 0 {
		blockSize = fileSize // 单分片兜底
	}

	f, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	for _, part := range prep.Parts {
		offset := int64(part.Index-1) * blockSize
		length := blockSize
		if offset+length > fileSize {
			length = fileSize - offset
		}
		if length <= 0 {
			break
		}
		chunk := make([]byte, length)
		if _, err := f.ReadAt(chunk, offset); err != nil {
			return fmt.Errorf("read chunk %d: %w", part.Index, err)
		}
		sum := md5.Sum(chunk)
		partMD5 := hex.EncodeToString(sum[:])

		if err := putPart(part.PresignedURL, chunk); err != nil {
			return fmt.Errorf("put part %d: %w", part.Index, err)
		}
		if err := retryPartFinish(appID, secret, openid, prep.UploadID, part.Index, length, partMD5); err != nil {
			return fmt.Errorf("part finish %d: %w", part.Index, err)
		}
	}

	token, err = getAccessToken(appID, secret)
	if err != nil {
		return err
	}
	fileInfo, err := retryCompleteUpload(token, openid, prep.UploadID)
	if err != nil {
		return fmt.Errorf("complete upload: %w", err)
	}

	token, err = getAccessToken(appID, secret)
	if err != nil {
		return err
	}
	return sendMediaMessage(token, openid, fileInfo)
}

// computeHashes 计算全文件 md5、sha1、md5_10m（文件 >md5_10m_size 取前 N 字节，否则等于 md5）。
func computeHashes(path string) (md5Hex, sha1Hex, md5_10m string, fileSize int64, err error) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	stat, err := f.Stat()
	if err != nil {
		return
	}
	fileSize = stat.Size()

	md5h := md5.New()
	sha1h := sha1.New()
	if _, err = io.Copy(io.MultiWriter(md5h, sha1h), f); err != nil {
		return
	}
	md5Hex = hex.EncodeToString(md5h.Sum(nil))
	sha1Hex = hex.EncodeToString(sha1h.Sum(nil))

	if fileSize > md5_10m_size {
		if _, err = f.Seek(0, io.SeekStart); err != nil {
			return
		}
		md5_10mh := md5.New()
		if _, err = io.CopyN(md5_10mh, f, md5_10m_size); err != nil {
			return
		}
		md5_10m = hex.EncodeToString(md5_10mh.Sum(nil))
	} else {
		md5_10m = md5Hex
	}
	return
}

type uploadPrepareResp struct {
	UploadID  string `json:"upload_id"`
	BlockSize any    `json:"block_size"` // 开放平台可能返回字符串
	Parts     []struct {
		Index        int    `json:"index"`
		PresignedURL string `json:"presigned_url"`
	} `json:"parts"`
}

func uploadPrepare(token, openid, fileName string, fileSize int64, md5Hex, sha1Hex, md5_10m string) (*uploadPrepareResp, error) {
	body, _ := json.Marshal(map[string]any{
		"file_type": 1, // 1=图片
		"file_name": fileName,
		"file_size": fileSize,
		"md5":       md5Hex,
		"sha1":      sha1Hex,
		"md5_10m":   md5_10m,
	})
	url := fmt.Sprintf("%s/v2/users/%s/upload_prepare", apiBase, openid)
	req, _ := http.NewRequest("POST", url, strings.NewReader(string(body)))
	req.Header.Set("Authorization", "QQBot "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("upload_prepare %d: %s", resp.StatusCode, string(data))
	}
	var r uploadPrepareResp
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("decode upload_prepare: %w", err)
	}
	if r.UploadID == "" {
		return nil, fmt.Errorf("empty upload_id: %s", string(data))
	}
	return &r, nil
}

// putPart PUT 分片二进制到预签名 URL。
func putPart(presignedURL string, data []byte) error {
	req, err := http.NewRequest("PUT", presignedURL, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.ContentLength = int64(len(data))
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("PUT %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

func uploadPartFinish(token, openid, uploadID string, partIndex int, blockSize int64, md5Hex string) error {
	body, _ := json.Marshal(map[string]any{
		"upload_id":  uploadID,
		"part_index": partIndex,
		"block_size": blockSize,
		"md5":        md5Hex,
	})
	url := fmt.Sprintf("%s/v2/users/%s/upload_part_finish", apiBase, openid)
	req, _ := http.NewRequest("POST", url, strings.NewReader(string(body)))
	req.Header.Set("Authorization", "QQBot "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upload_part_finish %d: %d %s", partIndex, resp.StatusCode, string(data))
	}
	return nil
}

// retryPartFinish part_finish 失败重试 2 次（共 3 次尝试）。
func retryPartFinish(appID, secret, openid, uploadID string, partIndex int, blockSize int64, md5Hex string) error {
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		token, err := getAccessToken(appID, secret)
		if err != nil {
			lastErr = err
		} else if err := uploadPartFinish(token, openid, uploadID, partIndex, blockSize, md5Hex); err != nil {
			lastErr = err
		} else {
			return nil
		}
		if attempt < 2 {
			time.Sleep(time.Duration(1000*(1<<attempt)) * time.Millisecond)
		}
	}
	return lastErr
}

type mediaUploadResp struct {
	FileUUID string `json:"file_uuid"`
	FileInfo string `json:"file_info"`
	TTL      int    `json:"ttl"`
}

func completeUpload(token, openid, uploadID string) (*mediaUploadResp, error) {
	body, _ := json.Marshal(map[string]any{"upload_id": uploadID})
	url := fmt.Sprintf("%s/v2/users/%s/files", apiBase, openid)
	req, _ := http.NewRequest("POST", url, strings.NewReader(string(body)))
	req.Header.Set("Authorization", "QQBot "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("complete upload %d: %s", resp.StatusCode, string(data))
	}
	var r mediaUploadResp
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("decode complete upload: %w", err)
	}
	if r.FileInfo == "" {
		return nil, fmt.Errorf("empty file_info: %s", string(data))
	}
	return &r, nil
}

// retryCompleteUpload 完成上传失败重试 2 次（共 3 次尝试）。
func retryCompleteUpload(token, openid, uploadID string) (string, error) {
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		r, err := completeUpload(token, openid, uploadID)
		if err != nil {
			lastErr = err
		} else {
			return r.FileInfo, nil
		}
		if attempt < 2 {
			time.Sleep(time.Duration(1000*(1<<attempt)) * time.Millisecond)
		}
	}
	return "", lastErr
}

// sendMediaMessage 发送 msg_type:7 富媒体消息。
func sendMediaMessage(token, openid, fileInfo string) error {
	body, _ := json.Marshal(map[string]any{
		"msg_type": 7,
		"media":    map[string]string{"file_info": fileInfo},
		"msg_seq":  nextMsgSeq(),
	})
	url := fmt.Sprintf("%s/v2/users/%s/messages", apiBase, openid)
	req, _ := http.NewRequest("POST", url, strings.NewReader(string(body)))
	req.Header.Set("Authorization", "QQBot "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("send media msg %d: %s", resp.StatusCode, string(data))
	}
	return nil
}

// nextMsgSeq 生成 0~65535 的消息序号（毫秒时间戳低位异或随机数）。
func nextMsgSeq() int {
	timePart := int(time.Now().UnixMilli()) % 100000000
	random := rand.Intn(65536)
	return (timePart ^ random) % 65536
}

// toInt64 兼容开放平台返回的数字/字符串形式。
func toInt64(v any) int64 {
	switch n := v.(type) {
	case float64:
		return int64(n)
	case string:
		i, _ := strconv.ParseInt(n, 10, 64)
		return i
	case json.Number:
		i, _ := n.Int64()
		return i
	}
	return 0
}
