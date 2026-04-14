package embeddings

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type OllamaClient struct {
	baseURL    string
	httpClient *http.Client
	model      string
}

type embedRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

type embedResponse struct {
	Embeddings [][]float32 `json:"embeddings"`
}

func NewOllamaClient(baseURL, model string, timeout time.Duration) *OllamaClient {
	return &OllamaClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: timeout,
		},
		model: model,
	}
}

func (c *OllamaClient) Embed(ctx context.Context, input string) ([]float32, error) {
	requestBody, err := json.Marshal(embedRequest{
		Model: c.model,
		Input: input,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/api/embed",
		bytes.NewReader(requestBody),
	)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama embed returned status %d", resp.StatusCode)
	}

	var payload embedResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	if len(payload.Embeddings) == 0 {
		return nil, fmt.Errorf("ollama embed returned no embeddings")
	}

	return payload.Embeddings[0], nil
}
