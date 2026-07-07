package rerank

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"zhixuan/server/config"
)

// Client is a DashScope rerank HTTP client.
type Client struct {
	apiKey  string
	baseURL string
	model   string
}

// New creates a new rerank client.
func New() *Client {
	return &Client{
		apiKey:  config.RerankAPIKey,
		baseURL: config.RerankBaseURL,
		model:   config.RerankModel,
	}
}

// Result represents a reranked document with its relevance score.
type Result struct {
	Index          int     `json:"index"`
	RelevanceScore float64 `json:"relevance_score"`
	Text           string  `json:"-"`
}

type rerankRequest struct {
	Model string `json:"model"`
	Query string `json:"query"`
	Docs  []string `json:"documents"`
	TopN  int      `json:"top_n"`
}

type rerankResponse struct {
	Results []rerankResult `json:"results"`
	ErrMsg  string         `json:"message,omitempty"`
}

type rerankResult struct {
	Index          int     `json:"index"`
	RelevanceScore float64 `json:"relevance_score"`
}

// Rerank calls the DashScope rerank API and returns sorted results.
func (c *Client) Rerank(ctx context.Context, query string, docs []string, topN int) ([]Result, error) {
	reqBody := rerankRequest{
		Model: c.model,
		Query: query,
		Docs:  docs,
		TopN:  topN,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/reranks", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("rerank request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading rerank response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("rerank API error %d: %s", resp.StatusCode, string(respBody))
	}

	var result rerankResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("decoding rerank response: %w", err)
	}

	out := make([]Result, len(result.Results))
	for i, r := range result.Results {
		out[i] = Result{
			Index:          r.Index,
			RelevanceScore: r.RelevanceScore,
			Text:           docs[r.Index],
		}
	}
	return out, nil
}
