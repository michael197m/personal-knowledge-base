package embeddings

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"
)

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestOllamaClientEmbed(t *testing.T) {
	var gotPath string
	var gotBody map[string]any

	client := NewOllamaClient("http://ollama.test", "nomic-embed-text", 2*time.Second)
	client.httpClient = &http.Client{
		Timeout: 2 * time.Second,
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			gotPath = r.URL.Path
			if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
				t.Fatalf("decode request body: %v", err)
			}

			payload, err := json.Marshal(map[string]any{
				"embeddings": [][]float32{{0.1, 0.2, 0.3}},
			})
			if err != nil {
				t.Fatalf("marshal response body: %v", err)
			}

			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(payload)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	vector, err := client.Embed(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Embed returned error: %v", err)
	}

	if gotPath != "/api/embed" {
		t.Fatalf("expected /api/embed, got %q", gotPath)
	}

	if gotBody["model"] != "nomic-embed-text" || gotBody["input"] != "hello" {
		t.Fatalf("unexpected request body: %#v", gotBody)
	}

	if len(vector) != 3 || vector[0] != 0.1 {
		t.Fatalf("unexpected embedding vector: %v", vector)
	}
}
