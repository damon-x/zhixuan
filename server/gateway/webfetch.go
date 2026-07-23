package gateway

import (
	"fmt"
	"html"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

const (
	// 用 Baiduspider 爬虫 UA，触发服务端渲染，避免拿到 JS 空壳。
	webFetchUA = "Mozilla/5.0 (compatible; Baiduspider/2.0; +http://www.baidu.com/search/spider.html)"
	// 最多读 5MB 原始 HTML，防止超大页面吃内存
	webFetchMaxBytes = 5 * 1024 * 1024
	// 返回给 LLM 的纯文本最多 5000 字（按 rune 计）
	webFetchMaxChars = 5000
)

var (
	scriptRe     = regexp.MustCompile(`(?is)<script\b[^>]*>.*?</script>`)
	styleRe      = regexp.MustCompile(`(?is)<style\b[^>]*>.*?</style>`)
	tagRe        = regexp.MustCompile(`<[^>]+>`)
	whitespaceRe = regexp.MustCompile(`[ \t\r\f\v]+`)
	newlineSpRe  = regexp.MustCompile(` *\n *`)
	multiNLRe    = regexp.MustCompile(`\n{2,}`)
	charsetRe    = regexp.MustCompile(`(?i)charset=([\w-]+)`)
)

// fetchWebPage 抓取 url 并提取纯文本。
// 策略：用 Baiduspider 爬虫 UA + Referer 触发服务端渲染；
// 按响应头/页面 meta 识别编码（支持 GBK 系），剔除 script/style 与所有标签；
// 解码 HTML 实体，规整空白，截断到 webFetchMaxChars 字返回。
func fetchWebPage(rawURL string) (string, error) {
	lower := strings.ToLower(rawURL)
	if !strings.HasPrefix(lower, "http://") && !strings.HasPrefix(lower, "https://") {
		return "", fmt.Errorf("URL 必须以 http:// 或 https:// 开头")
	}

	client := &http.Client{Timeout: 20 * time.Second}
	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return "", fmt.Errorf("构造请求失败: %w", err)
	}
	req.Header.Set("User-Agent", webFetchUA)
	req.Header.Set("Referer", rawURL)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	// 限制最多读 webFetchMaxBytes，防止超大页面吃内存
	body, err := io.ReadAll(io.LimitReader(resp.Body, webFetchMaxBytes))
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %w", err)
	}

	// 识别编码：优先 Content-Type，其次页面 meta
	charset := detectCharset(resp.Header.Get("Content-Type"), body)
	text := decodeToString(body, charset)

	// 提纯文本
	text = extractPlainText(text)
	if text == "" {
		return "", fmt.Errorf("页面无有效文本内容")
	}

	// 截断到 webFetchMaxChars 字（按 rune）
	if runes := []rune(text); len(runes) > webFetchMaxChars {
		text = string(runes[:webFetchMaxChars]) + "...(已截断)"
	}
	return text, nil
}

// detectCharset 从 Content-Type 或 HTML meta 中识别字符集。
func detectCharset(contentType string, body []byte) string {
	// 1) Content-Type header
	if m := charsetRe.FindStringSubmatch(contentType); len(m) > 1 {
		return strings.TrimSpace(m[1])
	}
	// 2) meta charset in first 1KB
	head := body
	if len(head) > 1024 {
		head = head[:1024]
	}
	if m := charsetRe.FindStringSubmatch(string(head)); len(m) > 1 {
		return strings.TrimSpace(m[1])
	}
	return "utf-8"
}

// decodeToString 按字符集名把 body 解码为 UTF-8 字符串；未知/不支持的按 UTF-8 原样返回。
func decodeToString(body []byte, charset string) string {
	switch strings.ToLower(charset) {
	case "gbk", "gb2312":
		enc := simplifiedchinese.GBK.NewDecoder()
		decoded, err := io.ReadAll(transform.NewReader(strings.NewReader(string(body)), enc))
		if err == nil {
			return string(decoded)
		}
	case "gb18030":
		enc := simplifiedchinese.GB18030.NewDecoder()
		decoded, err := io.ReadAll(transform.NewReader(strings.NewReader(string(body)), enc))
		if err == nil {
			return string(decoded)
		}
	}
	// 默认 utf-8
	return string(body)
}

// extractPlainText 整页删标签、留纯文字：
// 1) 整段剔除 script/style（含内部 JS/CSS）
// 2) 删掉所有剩余 <...> 标签
// 3) 解码 HTML 实体
// 4) 规整空白、压缩空行
func extractPlainText(s string) string {
	s = scriptRe.ReplaceAllString(s, "")
	s = styleRe.ReplaceAllString(s, "")
	s = tagRe.ReplaceAllString(s, "")
	s = html.UnescapeString(s)
	s = whitespaceRe.ReplaceAllString(s, " ")
	s = newlineSpRe.ReplaceAllString(s, "\n")
	s = multiNLRe.ReplaceAllString(s, "\n")
	return strings.TrimSpace(s)
}
