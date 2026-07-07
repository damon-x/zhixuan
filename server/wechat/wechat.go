package wechat

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	pluginVersion            = "2.4.3"
	iLinkAppID               = "bot"
	defaultBotAgent          = "GoWeixinEcho/0.1.0"
	defaultBaseURL           = "https://ilinkai.weixin.qq.com"
	defaultBotType           = "3"
	defaultLongPollTimeoutMs = 35000
	defaultAPITimeoutMs      = 15000
	sessionExpiredErrCode    = -14
	messageTypeUser          = 1
	messageTypeBot           = 2
	messageStateFinish       = 2
	messageItemTypeText      = 1
	defaultNotifyTimeoutMs   = 5000
)

// State holds the persistent state for a WeChat client.
type State struct {
	Token         string `json:"token,omitempty"`
	AccountID     string `json:"account_id,omitempty"`
	UserID        string `json:"user_id,omitempty"`
	BaseURL       string `json:"base_url,omitempty"`
	GetUpdatesBuf string `json:"get_updates_buf,omitempty"`
}

// WeixinMessage represents a single WeChat message.
type WeixinMessage struct {
	Seq          int           `json:"seq,omitempty"`
	MessageID    int64         `json:"message_id,omitempty"`
	FromUserID   string        `json:"from_user_id,omitempty"`
	ToUserID     string        `json:"to_user_id,omitempty"`
	ClientID     string        `json:"client_id,omitempty"`
	CreateTimeMS int64         `json:"create_time_ms,omitempty"`
	UpdateTimeMS int64         `json:"update_time_ms,omitempty"`
	SessionID    string        `json:"session_id,omitempty"`
	GroupID      string        `json:"group_id,omitempty"`
	MessageType  int           `json:"message_type,omitempty"`
	MessageState int           `json:"message_state,omitempty"`
	ItemList     []MessageItem `json:"item_list,omitempty"`
	ContextToken string        `json:"context_token,omitempty"`
}

// MessageItem represents an item within a WeChat message.
type MessageItem struct {
	Type     int       `json:"type,omitempty"`
	TextItem *TextItem `json:"text_item,omitempty"`
}

// TextItem represents a text content item.
type TextItem struct {
	Text string `json:"text,omitempty"`
}

// Client is the WeChat iLink client.
type Client struct {
	statePath         string
	baseURL           string
	token             string
	accountID         string
	userID            string
	getUpdatesBuf     string
	nextPollTimeoutMs int
	httpClient        *http.Client
}

// internal response types

type baseInfo struct {
	ChannelVersion string `json:"channel_version,omitempty"`
	BotAgent       string `json:"bot_agent,omitempty"`
}

type qRCodeResponse struct {
	QRCode         string `json:"qrcode"`
	QRCodeImgValue string `json:"qrcode_img_content"`
}

type qRStatusResponse struct {
	Status       string `json:"status"`
	BotToken     string `json:"bot_token"`
	ILinkBotID   string `json:"ilink_bot_id"`
	BaseURL      string `json:"baseurl"`
	ILinkUserID  string `json:"ilink_user_id"`
	RedirectHost string `json:"redirect_host"`
}

type basicRetResponse struct {
	Ret    *int   `json:"ret,omitempty"`
	ErrMsg string `json:"errmsg,omitempty"`
}

type getUpdatesResponse struct {
	Ret                  *int            `json:"ret,omitempty"`
	ErrCode              *int            `json:"errcode,omitempty"`
	ErrMsg               string          `json:"errmsg,omitempty"`
	Msgs                 []WeixinMessage `json:"msgs,omitempty"`
	GetUpdatesBuf        string          `json:"get_updates_buf,omitempty"`
	LongPollingTimeoutMS int             `json:"longpolling_timeout_ms,omitempty"`
}

type sendMessageRequest struct {
	Msg      WeixinMessage `json:"msg"`
	BaseInfo baseInfo      `json:"base_info"`
}

type qRCodeRequest struct {
	LocalTokenList []string `json:"local_token_list"`
}

type getUpdatesRequest struct {
	GetUpdatesBuf string   `json:"get_updates_buf"`
	BaseInfo      baseInfo `json:"base_info"`
}

type notifyRequest struct {
	BaseInfo baseInfo `json:"base_info"`
}

func buildClientVersion(version string) int {
	parts := strings.Split(version, ".")
	read := func(idx int) int {
		if idx >= len(parts) {
			return 0
		}
		n, err := strconv.Atoi(parts[idx])
		if err != nil {
			return 0
		}
		return n
	}
	major := read(0)
	minor := read(1)
	patch := read(2)
	return ((major & 0xFF) << 16) | ((minor & 0xFF) << 8) | (patch & 0xFF)
}

var iLinkAppClientVersion = buildClientVersion(pluginVersion)

func trimBaseURL(raw string) string {
	return strings.TrimRight(strings.TrimSpace(raw), "/") + "/"
}

func randomWechatUIN() (string, error) {
	var raw [4]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	n := uint32(raw[0])<<24 | uint32(raw[1])<<16 | uint32(raw[2])<<8 | uint32(raw[3])
	return base64.StdEncoding.EncodeToString([]byte(strconv.FormatUint(uint64(n), 10))), nil
}

func newHTTPClient() *http.Client {
	pool, err := x509.SystemCertPool()
	tlsConfig := &tls.Config{MinVersion: tls.VersionTLS12}
	if err == nil && pool != nil {
		tlsConfig.RootCAs = pool
	}
	transport := &http.Transport{
		Proxy:               http.ProxyFromEnvironment,
		TLSClientConfig:     tlsConfig,
		DialContext:         (&net.Dialer{Timeout: 10 * time.Second}).DialContext,
		ForceAttemptHTTP2:   true,
		TLSHandshakeTimeout: 10 * time.Second,
	}
	return &http.Client{Transport: transport}
}

func loadState(path string) State {
	raw, err := os.ReadFile(path)
	if err != nil {
		return State{}
	}
	var state State
	if json.Unmarshal(raw, &state) != nil {
		return State{}
	}
	return state
}

func saveStateFile(path string, state State) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func (c *Client) baseInfo() baseInfo {
	return baseInfo{
		ChannelVersion: pluginVersion,
		BotAgent:       defaultBotAgent,
	}
}

func (c *Client) commonHeaders() map[string]string {
	return map[string]string{
		"iLink-App-Id":            iLinkAppID,
		"iLink-App-ClientVersion": strconv.Itoa(iLinkAppClientVersion),
	}
}

func (c *Client) headers(includeAuth bool, isJSON bool) (map[string]string, error) {
	headers := c.commonHeaders()
	uin, err := randomWechatUIN()
	if err != nil {
		return nil, err
	}
	headers["X-WECHAT-UIN"] = uin
	if isJSON {
		headers["Content-Type"] = "application/json"
	}
	if includeAuth {
		headers["AuthorizationType"] = "ilink_bot_token"
		if strings.TrimSpace(c.token) != "" {
			headers["Authorization"] = "Bearer " + strings.TrimSpace(c.token)
		}
	}
	return headers, nil
}

func (c *Client) newRequest(ctx context.Context, method, baseURL, endpoint string, body []byte, includeAuth bool, isJSON bool) (*http.Request, error) {
	fullURL, err := url.Parse(trimBaseURL(baseURL))
	if err != nil {
		return nil, err
	}
	rel, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, method, fullURL.ResolveReference(rel).String(), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	headers, err := c.headers(includeAuth, isJSON)
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return req, nil
}

func doJSON[T any](c *Client, ctx context.Context, method, baseURL, endpoint string, payload any, includeAuth bool, isJSON bool) (T, error) {
	var zero T
	var body []byte
	var err error
	if payload != nil {
		body, err = json.Marshal(payload)
		if err != nil {
			return zero, err
		}
	}
	req, err := c.newRequest(ctx, method, baseURL, endpoint, body, includeAuth, isJSON)
	if err != nil {
		return zero, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return zero, err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return zero, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return zero, fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return zero, nil
	}
	var out T
	if err := json.Unmarshal(raw, &out); err != nil {
		return zero, fmt.Errorf("non-JSON response: %s", strings.TrimSpace(string(raw)))
	}
	return out, nil
}

func isTimeoutErr(err error) bool {
	if err == nil {
		return false
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	return errors.Is(err, context.DeadlineExceeded)
}

func intPtr(v int) *int    { return &v }
func intValue(v *int) int  {
	if v == nil {
		return 0
	}
	return *v
}

// FetchQRCode fetches a QR code for WeChat binding.
// Returns the qrcode identifier and base64 image data.
func FetchQRCode() (qrcode string, qrImage string, err error) {
	c := &Client{
		baseURL:           trimBaseURL(defaultBaseURL),
		nextPollTimeoutMs: defaultLongPollTimeoutMs,
		httpClient:        newHTTPClient(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(defaultAPITimeoutMs)*time.Millisecond)
	defer cancel()

	endpoint := "ilink/bot/get_bot_qrcode?bot_type=" + url.QueryEscape(defaultBotType)
	resp, err := doJSON[qRCodeResponse](c, ctx, "POST", defaultBaseURL, endpoint, qRCodeRequest{}, false, true)
	if err != nil {
		return "", "", err
	}
	if strings.TrimSpace(resp.QRCode) == "" || strings.TrimSpace(resp.QRCodeImgValue) == "" {
		return "", "", fmt.Errorf("missing qrcode fields in response")
	}
	return resp.QRCode, resp.QRCodeImgValue, nil
}

// PollQRStatus polls the QR code scan status.
// Returns status string and on success: botToken, accountID, userID, baseURL.
func PollQRStatus(qrcode string) (status string, botToken string, accountID string, userID string, baseURL string, err error) {
	c := &Client{
		baseURL:           trimBaseURL(defaultBaseURL),
		nextPollTimeoutMs: defaultLongPollTimeoutMs,
		httpClient:        newHTTPClient(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(defaultLongPollTimeoutMs)*time.Millisecond)
	defer cancel()

	endpoint := "ilink/bot/get_qrcode_status?qrcode=" + url.QueryEscape(qrcode)
	resp, err := doJSON[qRStatusResponse](c, ctx, "GET", defaultBaseURL, endpoint, nil, false, false)
	if err != nil {
		if isTimeoutErr(err) {
			return "wait", "", "", "", "", nil
		}
		return "wait", "", "", "", "", nil
	}
	return resp.Status, resp.BotToken, resp.ILinkBotID, resp.ILinkUserID, resp.BaseURL, nil
}

// NewClient loads a WeChat client from a state file.
func NewClient(statePath string) *Client {
	state := loadState(statePath)
	c := &Client{
		statePath:         statePath,
		baseURL:           trimBaseURL(defaultBaseURL),
		token:             strings.TrimSpace(state.Token),
		accountID:         strings.TrimSpace(state.AccountID),
		userID:            strings.TrimSpace(state.UserID),
		getUpdatesBuf:     state.GetUpdatesBuf,
		nextPollTimeoutMs: defaultLongPollTimeoutMs,
		httpClient:        newHTTPClient(),
	}
	if strings.TrimSpace(state.BaseURL) != "" {
		c.baseURL = trimBaseURL(state.BaseURL)
	}
	return c
}

// NotifyStart notifies the WeChat server that the bot is online.
func (c *Client) NotifyStart() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(defaultNotifyTimeoutMs)*time.Millisecond)
	defer cancel()
	resp, err := doJSON[basicRetResponse](c, ctx, "POST", c.baseURL, "ilink/bot/msg/notifystart", notifyRequest{BaseInfo: c.baseInfo()}, true, true)
	if err != nil {
		return fmt.Errorf("notifyStart failed: %w", err)
	}
	if resp.Ret != nil && *resp.Ret != 0 {
		return fmt.Errorf("notifyStart returned ret=%d errmsg=%s", *resp.Ret, resp.ErrMsg)
	}
	return nil
}

// NotifyStop notifies the WeChat server that the bot is going offline.
func (c *Client) NotifyStop() {
	if strings.TrimSpace(c.token) == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(defaultNotifyTimeoutMs)*time.Millisecond)
	defer cancel()
	doJSON[basicRetResponse](c, ctx, "POST", c.baseURL, "ilink/bot/msg/notifystop", notifyRequest{BaseInfo: c.baseInfo()}, true, true)
}

// GetUpdates performs a long-poll to get new messages.
// Automatically saves state after successful poll.
func (c *Client) GetUpdates(ctx context.Context) ([]WeixinMessage, error) {
	timeoutMs := c.nextPollTimeoutMs
	if timeoutMs <= 0 {
		timeoutMs = defaultLongPollTimeoutMs
	}
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	resp, err := doJSON[getUpdatesResponse](c, timeoutCtx, "POST", c.baseURL, "ilink/bot/getupdates", getUpdatesRequest{
		GetUpdatesBuf: c.getUpdatesBuf,
		BaseInfo:      c.baseInfo(),
	}, true, true)
	if err != nil {
		if isTimeoutErr(err) {
			return []WeixinMessage{}, nil
		}
		return nil, err
	}

	ret := intValue(resp.Ret)
	errCode := intValue(resp.ErrCode)
	if errCode == sessionExpiredErrCode || ret == sessionExpiredErrCode {
		return nil, fmt.Errorf("session expired")
	}
	if ret != 0 || (resp.ErrCode != nil && errCode != 0) {
		return nil, fmt.Errorf("getUpdates failed ret=%d errcode=%d errmsg=%s", ret, errCode, resp.ErrMsg)
	}

	if resp.LongPollingTimeoutMS > 0 {
		c.nextPollTimeoutMs = resp.LongPollingTimeoutMS
	}
	if strings.TrimSpace(resp.GetUpdatesBuf) != "" {
		c.getUpdatesBuf = resp.GetUpdatesBuf
		c.SaveState()
	}

	return resp.Msgs, nil
}

// SendText sends a text message to a WeChat user.
func (c *Client) SendText(toUserID, text, contextToken string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(defaultAPITimeoutMs)*time.Millisecond)
	defer cancel()

	var raw [8]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return err
	}
	clientID := fmt.Sprintf("zhixuan-wechat:%d-%x", time.Now().UnixMilli(), raw[:])

	msg := WeixinMessage{
		FromUserID:   "",
		ToUserID:     toUserID,
		ClientID:     clientID,
		MessageType:  messageTypeBot,
		MessageState: messageStateFinish,
		ItemList: []MessageItem{
			{
				Type:     messageItemTypeText,
				TextItem: &TextItem{Text: text},
			},
		},
		ContextToken: contextToken,
	}
	_, err := doJSON[map[string]any](c, ctx, "POST", c.baseURL, "ilink/bot/sendmessage", sendMessageRequest{
		Msg:      msg,
		BaseInfo: c.baseInfo(),
	}, true, true)
	return err
}

// UserID returns the ilink user ID stored in state.
func (c *Client) UserID() string {
	return c.userID
}

// SaveState persists the client state to file.
func (c *Client) SaveState() error {
	state := State{
		Token:         c.token,
		AccountID:     c.accountID,
		UserID:        c.userID,
		BaseURL:       strings.TrimRight(c.baseURL, "/"),
		GetUpdatesBuf: c.getUpdatesBuf,
	}
	return saveStateFile(c.statePath, state)
}

// ExtractText extracts text content from a WeixinMessage.
func ExtractText(msg WeixinMessage) string {
	for _, item := range msg.ItemList {
		if item.Type == messageItemTypeText && item.TextItem != nil {
			return item.TextItem.Text
		}
	}
	return ""
}

// StateFileExists checks if a state file exists and is valid.
func StateFileExists(statePath string) bool {
	state := loadState(statePath)
	return strings.TrimSpace(state.Token) != ""
}

// SaveNewState saves a newly created state (from QR binding) to disk.
func SaveNewState(statePath string, token, accountID, userID, baseURL string) error {
	state := State{
		Token:     token,
		AccountID: accountID,
		UserID:    userID,
		BaseURL:   strings.TrimRight(baseURL, "/"),
	}
	return saveStateFile(statePath, state)
}

// StateDir returns the directory for WeChat state files under dataDir.
func StateDir(dataDir string, userID uint) string {
	return filepath.Join(dataDir, "wechat", fmt.Sprintf("%d", userID))
}

// StatePath returns the state file path for a given user.
func StatePath(dataDir string, userID uint) string {
	return filepath.Join(StateDir(dataDir, userID), "state.json")
}
