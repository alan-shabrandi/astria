package embedding

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"time"

	"github.com/alanshabrandi/astria/internal/domain"
)

const (
	openAIEndpoint = "https://api.openai.com/v1/embeddings"
	defaultModel   = "text-embedding-3-small"

	maxRetries = 4
	baseDelay  = 1 * time.Second
	maxDelay   = 15 * time.Second
)

type openAIRequest struct {
	Input []string `json:"input"`
	Model string   `json:"model"`
}

type openAIResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

type OpenAIProvider struct {
	client *http.Client
	apiKey string
	model  string
}

func NewOpenAIProvider(apiKey string) *OpenAIProvider {
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     90 * time.Second,
	}
	return &OpenAIProvider{
		client: &http.Client{
			Transport: transport,
			Timeout:   30 * time.Second,
		},
		apiKey: apiKey,
		model:  defaultModel,
	}
}

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

	var lastErr error
	delay := baseDelay

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("context cancelled during embedding generation: %w", err)
		}

		apiResp, retryable, err := p.doSingleRequest(ctx, jsonData)
		if err == nil {
			vectors := make([]domain.Vector, len(apiResp.Data))
			for i, d := range apiResp.Data {
				vectors[i] = d.Embedding
			}
			return vectors, nil
		}

		lastErr = err

		if !retryable || attempt == maxRetries {
			break
		}

		jitter := time.Duration(rand.Int63n(int64(delay) / 2))
		sleepTime := delay + jitter

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("context cancelled during retry wait: %w", ctx.Err())
		case <-time.After(sleepTime):
		}

		delay *= 2
		if delay > maxDelay {
			delay = maxDelay
		}
	}

	return nil, fmt.Errorf("failed after %d attempts, last error: %w", maxRetries+1, lastErr)
}

func (p *OpenAIProvider) doSingleRequest(ctx context.Context, payload []byte) (*openAIResponse, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, openAIEndpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, false, fmt.Errorf("failed to create http request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, true, fmt.Errorf("network request failed: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, true, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		isRetryable := resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= http.StatusInternalServerError

		var apiErr openAIResponse
		_ = json.Unmarshal(bodyBytes, &apiErr)

		errMsg := "unknown api error"
		if apiErr.Error != nil && apiErr.Error.Message != "" {
			errMsg = apiErr.Error.Message
		} else if len(bodyBytes) > 0 {
			errMsg = string(bodyBytes)
		}

		return nil, isRetryable, fmt.Errorf("openai error (status %d): %s", resp.StatusCode, errMsg)
	}

	var apiResp openAIResponse
	if err := json.Unmarshal(bodyBytes, &apiResp); err != nil {
		return nil, false, fmt.Errorf("failed to decode api response: %w", err)
	}

	return &apiResp, false, nil
}

func (p *OpenAIProvider) Dimensions() int {
	return 1536
}
