package embedding

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/alanshabrandi/astria/internal/domain"
)

const (
	openAIEndpoint = "https://api.openai.com/v1/embeddings"
	defaultModel   = "text-embedding-3-small"
	dimensions     = 1536
)

// openAIRequest represents the JSON payload sent to the API.
type openAIRequest struct {
	Input []string `json:"input"`
	Model string   `json:"model"`
}

// openAIResponse represents the JSON response received from the API.
type openAIResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// OpenAIProvider implements domain.EmbeddingProvider using the OpenAI REST API.
type OpenAIProvider struct {
	client *http.Client
	apiKey string
	model  string
}

// NewOpenAIProvider initializes the provider with a robust, production-ready HTTP client.
func NewOpenAIProvider(apiKey string) *OpenAIProvider {
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     90 * time.Second,
		DisableKeepAlives:   false,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}

	return &OpenAIProvider{
		client: client,
		apiKey: apiKey,
		model:  defaultModel,
	}
}

// GenerateEmbeddings converts a batch of texts into vector representations.
func (p *OpenAIProvider) GenerateEmbeddings(ctx context.Context, texts []string) ([]domain.Vector, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	reqBody := openAIRequest{
		Input: texts,
		Model: p.model,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal openai request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, openAIEndpoint, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	var apiResp openAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode api response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		errMsg := "unknown error"
		if apiResp.Error != nil {
			errMsg = apiResp.Error.Message
		}
		return nil, fmt.Errorf("openai api error (status %d): %s", resp.StatusCode, errMsg)
	}

	vectors := make([]domain.Vector, len(apiResp.Data))
	for i, d := range apiResp.Data {
		vectors[i] = d.Embedding
	}

	return vectors, nil
}

// Dimensions returns the exact vector size produced by this provider.
func (p *OpenAIProvider) Dimensions() int {
	return dimensions
}
