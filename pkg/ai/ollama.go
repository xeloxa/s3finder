package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Ollama implements Generator using local Ollama instance.
type Ollama struct {
	baseURL     string
	model       string
	temperature float64
	client      *http.Client
}

type ollamaRequest struct {
	Model       string  `json:"model"`
	Prompt      string  `json:"prompt"`
	Stream      bool    `json:"stream"`
	Temperature float64 `json:"temperature,omitempty"`
}

type ollamaResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

// NewOllama creates an Ollama generator.
func NewOllama(cfg *Config) (*Ollama, error) {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	model := cfg.Model
	if model == "" {
		model = "llama3"
	}

	temp := cfg.Temperature
	if temp <= 0 {
		temp = 0.7
	}

	return &Ollama{
		baseURL:     baseURL,
		model:       model,
		temperature: temp,
		client: &http.Client{
			Timeout: 120 * time.Second,
		},
	}, nil
}

// Generate creates bucket names using Ollama.
func (o *Ollama) Generate(ctx context.Context, seed string, count int) ([]string, error) {
	prompt := fmt.Sprintf(BucketPrompt, seed, count, seed)

	reqBody := ollamaRequest{
		Model:       o.model,
		Prompt:      prompt,
		Stream:      false,
		Temperature: o.temperature,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", o.baseURL+"/api/generate", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Ollama API error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Ollama returned status %d", resp.StatusCode)
	}

	var result ollamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return parseBucketNames(result.Response), nil
}

// Name returns the provider name.
func (o *Ollama) Name() string {
	return "ollama"
}

// Model returns the model name.
func (o *Ollama) Model() string {
	return o.model
}
