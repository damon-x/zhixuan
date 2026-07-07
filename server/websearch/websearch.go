package websearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"zhixuan/server/config"
)

// Client is a Bocha Web Search API client.
type Client struct {
	apiKey  string
	baseURL string
}

// New creates a new Bocha web search client.
func New() *Client {
	return &Client{
		apiKey:  config.BochaAPIKey,
		baseURL: config.BochaBaseURL,
	}
}

// WebResult represents a single web search result.
type WebResult struct {
	URL     string `json:"url"`
	Summary string `json:"summary"`
	Name    string `json:"name"`
}

type bochaRequest struct {
	Query   string `json:"query"`
	Summary bool   `json:"summary"`
	Count   int    `json:"count"`
}

type bochaResponse struct {
	Code int          `json:"code"`
	Msg  string       `json:"msg"`
	Data *bochaData   `json:"data"`
}

type bochaData struct {
	WebPages *struct {
		Value []struct {
			Name    string `json:"name"`
			URL     string `json:"url"`
			Snippet string `json:"snippet"`
			Summary string `json:"summary"`
		} `json:"value"`
	} `json:"webPages"`
}

// Search calls the Bocha Web Search API and returns results.
func (c *Client) Search(ctx context.Context, query string) ([]WebResult, error) {
	reqBody := bochaRequest{
		Query:   query,
		Summary: true,
		Count:   8,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/web-search", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("web search request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading web search response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("web search API error %d: %s", resp.StatusCode, string(respBody))
	}

	var result bochaResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("decoding web search response: %w", err)
	}

	if result.Data == nil || result.Data.WebPages == nil || len(result.Data.WebPages.Value) == 0 {
		return nil, nil
	}

	out := make([]WebResult, 0, len(result.Data.WebPages.Value))
	for _, w := range result.Data.WebPages.Value {
		summary := w.Summary
		if summary == "" {
			summary = w.Snippet
		}
		out = append(out, WebResult{
			URL:     w.URL,
			Summary: summary,
			Name:    w.Name,
		})
	}
	return out, nil
}
