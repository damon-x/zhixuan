package vision

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image"
	"image/jpeg"
	"log"
	"strings"
	"sync"

	"zhixuan/server/config"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

const maxImageSize = 1 << 20 // 1MB

var (
	client   openai.Client
	models   []string
	modelsMu sync.Mutex
	inited   bool
)

func Init() {
	opts := []option.RequestOption{
		option.WithAPIKey(config.VisionAPIKey),
	}
	if config.VisionBaseURL != "" {
		opts = append(opts, option.WithBaseURL(config.VisionBaseURL))
	}
	client = openai.NewClient(opts...)

	modelsMu.Lock()
	models = make([]string, len(config.VisionModels))
	copy(models, config.VisionModels)
	modelsMu.Unlock()

	inited = true
	log.Printf("[vision] 初始化完成，models=%v", models)
}

func ensureInit() {
	if !inited {
		Init()
	}
}

// Describe sends an image to a vision model and returns the description text.
// ext should include the dot, e.g. ".jpg", ".png".
func Describe(ctx context.Context, imgData []byte, ext string) (string, error) {
	ensureInit()

	if len(models) == 0 {
		return "", fmt.Errorf("vision: no models configured")
	}

	// Compress if needed
	processed, err := compressImage(imgData, ext)
	if err != nil {
		return "", fmt.Errorf("vision: compress: %w", err)
	}

	b64 := base64.StdEncoding.EncodeToString(processed)
	mimeType := extToMIME(ext)
	dataURL := fmt.Sprintf("data:%s;base64,%s", mimeType, b64)

	const prompt = "描述一下图片内容,返回json 数组，两个字段, text:文字内容(可为空)，content:图片描述(不超过500字)"

	// Try models with auto-switch
	modelsMu.Lock()
	list := make([]string, len(models))
	copy(list, models)
	modelsMu.Unlock()

	var lastErr error
	for i, m := range list {
		msg := []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage([]openai.ChatCompletionContentPartUnionParam{
				openai.TextContentPart(prompt),
				openai.ImageContentPart(openai.ChatCompletionContentPartImageImageURLParam{
					URL: dataURL,
				}),
			}),
		}
		params := openai.ChatCompletionNewParams{
			Model:    openai.ChatModel(m),
			Messages: msg,
		}
		resp, err := client.Chat.Completions.New(ctx, params)
		if err == nil {
			log.Printf("[vision] 模型 %s 调用成功", m)
			if len(resp.Choices) == 0 {
				return "", fmt.Errorf("vision: no response")
			}
			return resp.Choices[0].Message.Content, nil
		}
		lastErr = err
		nextModel := ""
		if i+1 < len(list) {
			nextModel = list[i+1]
		}
		log.Printf("[vision] 模型 %s 调用失败: %v，切换到下一个模型 %s", m, err, nextModel)
		// move failed model to end
		modelsMu.Lock()
		models = append(models[:0], list...)
		models = append(models[:i], models[i+1:]...)
		models = append(models, m)
		modelsMu.Unlock()
	}
	return "", fmt.Errorf("vision: all models failed, last error: %w", lastErr)
}

func compressImage(data []byte, ext string) ([]byte, error) {
	if len(data) <= maxImageSize {
		return data, nil
	}

	// Decode image
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}

	// Try re-encoding as JPEG with decreasing quality
	for quality := 80; quality >= 20; quality -= 20 {
		var buf bytes.Buffer
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
			return nil, fmt.Errorf("encode jpeg q=%d: %w", quality, err)
		}
		if buf.Len() <= maxImageSize {
			log.Printf("[vision] 压缩完成 quality=%d size=%d", quality, buf.Len())
			return buf.Bytes(), nil
		}
	}

	// Scale down to 50% and retry
	bounds := img.Bounds()
	newW := bounds.Dx() / 2
	newH := bounds.Dy() / 2
	scaled := scaleDown(img, newW, newH)

	for quality := 80; quality >= 20; quality -= 20 {
		var buf bytes.Buffer
		if err := jpeg.Encode(&buf, scaled, &jpeg.Options{Quality: quality}); err != nil {
			return nil, fmt.Errorf("encode scaled jpeg q=%d: %w", quality, err)
		}
		if buf.Len() <= maxImageSize {
			log.Printf("[vision] 缩放+压缩完成 quality=%d size=%d", quality, buf.Len())
			return buf.Bytes(), nil
		}
	}

	return nil, fmt.Errorf("vision: unable to compress image to <= 1MB")
}

func scaleDown(img image.Image, w, h int) image.Image {
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	// Simple nearest-neighbor scaling
	srcW := img.Bounds().Dx()
	srcH := img.Bounds().Dy()
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			sx := x * srcW / w
			sy := y * srcH / h
			dst.Set(x, y, img.At(sx, sy))
		}
	}
	return dst
}

func extToMIME(ext string) string {
	switch strings.ToLower(ext) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	default:
		return "image/jpeg"
	}
}
