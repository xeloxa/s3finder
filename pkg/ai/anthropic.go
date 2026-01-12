package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Anthropic implements Generator using Anthropic's Claude API.
type Anthropic struct {
	apiKey      string
	model       string
	temperature float64
	client      *http.Client
}

type anthropicRequest struct {
	Model       string             `json:"model"`
	MaxTokens   int                `json:"max_tokens"`
	Temperature float64            `json:"temperature,omitempty"`
	Messages    []anthropicMessage `json:"messages"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// NewAnthropic creates an Anthropic generator.
func NewAnthropic(cfg *Config) (*Anthropic, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("Anthropic API key is required")
	}

	model := cfg.Model
	if model == "" {
		model = "claude-3-haiku-20240307"
	}

	temp := cfg.Temperature
	if temp <= 0 {
		temp = 0.7
	}

	return &Anthropic{
		apiKey:      cfg.APIKey,
		model:       model,
		temperature: temp,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}, nil
}

// Generate creates bucket names using Anthropic Claude.
func (a *Anthropic) Generate(ctx context.Context, seed string, count int) ([]string, error) {
	prompt := fmt.Sprintf(BucketPrompt, seed, count, seed)

	reqBody := anthropicRequest{
		Model:       a.model,
		MaxTokens:   2000,
		Temperature: a.temperature,
		Messages: []anthropicMessage{
			{Role: "user", Content: prompt},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", a.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Anthropic API error: %w", err)
	}
	defer resp.Body.Close()

	var result anthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if result.Error != nil {
		return nil, fmt.Errorf("Anthropic error: %s", result.Error.Message)
	}

	if len(result.Content) == 0 {
		return nil, fmt.Errorf("no response from Anthropic")
	}

	var text string
	for _, c := range result.Content {
		if c.Type == "text" {
			text = c.Text
			break
		}
	}

	return parseBucketNames(text), nil
}

// Name returns the provider name.
func (a *Anthropic) Name() string {
	return "anthropic"
}

// Model returns the model name.
func (a *Anthropic) Model() string {
	return a.model
}
