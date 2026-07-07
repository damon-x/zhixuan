package embedding

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"zhixuan/server/config"
)

// Client is a DashScope embedding HTTP client.
type Client struct {
	apiKey  string
	baseURL string
	model   string
}

// New creates a new embedding client using the dedicated embedding config.
func New() *Client {
	return &Client{
		apiKey:  config.EmbeddingAPIKey,
		baseURL: config.EmbeddingBaseURL,
		model:   config.EmbeddingModel,
	}
}

type embedRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type embedResponse struct {
	Data   []embedData `json:"data"`
	ErrMsg string      `json:"message,omitempty"`
}

type embedData struct {
	Embedding []float32 `json:"embedding"`
	Index     int       `json:"index"`
}

// Embed calls the DashScope embedding API and returns vectors for each text.
// It batches requests to respect the API limit of 10 texts per call.
func (c *Client) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	allVectors := make([][]float32, len(texts))

	for start := 0; start < len(texts); start += 10 {
		end := start + 10
		if end > len(texts) {
			end = len(texts)
		}
		batch := texts[start:end]

		reqBody := embedRequest{
			Model: c.model,
			Input: batch,
		}
		body, err := json.Marshal(reqBody)
		if err != nil {
			return nil, err
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/embeddings", bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+c.apiKey)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("embedding request failed: %w", err)
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("reading embedding response: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("embedding API error %d: %s", resp.StatusCode, string(respBody))
		}

		var result embedResponse
		if err := json.Unmarshal(respBody, &result); err != nil {
			return nil, fmt.Errorf("decoding embedding response: %w", err)
		}

		if len(result.Data) == 0 {
			return nil, fmt.Errorf("no embeddings returned")
		}

		for _, item := range result.Data {
			idx := start + item.Index
			if idx < len(allVectors) {
				allVectors[idx] = item.Embedding
			}
		}
	}

	return allVectors, nil
}
