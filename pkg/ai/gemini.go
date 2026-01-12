package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Gemini implements Generator using Google's Gemini API.
type Gemini struct {
	apiKey      string
	model       string
	baseURL     string
	temperature float64
	client      *http.Client
}

type geminiRequest struct {
	Contents         []geminiContent        `json:"contents"`
	GenerationConfig geminiGenerationConfig `json:"generationConfig,omitempty"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiGenerationConfig struct {
	Temperature     float64 `json:"temperature,omitempty"`
	MaxOutputTokens int     `json:"maxOutputTokens,omitempty"`
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// NewGemini creates a Gemini generator.
func NewGemini(cfg *Config) (*Gemini, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("Gemini API key is required")
	}

	model := cfg.Model
	if model == "" {
		model = "gemini-3-flash-preview"
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://generativelanguage.googleapis.com"
	}

	temp := cfg.Temperature
	if temp <= 0 {
		temp = 0.7
	}

	return &Gemini{
		apiKey:      cfg.APIKey,
		model:       model,
		baseURL:     baseURL,
		temperature: temp,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}, nil
}

// Generate creates bucket names using Google Gemini.
func (g *Gemini) Generate(ctx context.Context, seed string, count int) ([]string, error) {
	prompt := fmt.Sprintf(BucketPrompt, seed, count, seed)

	reqBody := geminiRequest{
		Contents: []geminiContent{
			{
				Parts: []geminiPart{
					{Text: prompt},
				},
			},
		},
		GenerationConfig: geminiGenerationConfig{
			Temperature:     g.temperature,
			MaxOutputTokens: 2000,
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/v1beta/models/%s:generateContent?key=%s", g.baseURL, g.model, g.apiKey)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Gemini API error: %w", err)
	}
	defer resp.Body.Close()

	var result geminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if result.Error != nil {
		return nil, fmt.Errorf("Gemini error: %s", result.Error.Message)
	}

	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no response from Gemini")
	}

	return parseBucketNames(result.Candidates[0].Content.Parts[0].Text), nil
}

// Name returns the provider name.
func (g *Gemini) Name() string {
	return "gemini"
}

// Model returns the model name.
func (g *Gemini) Model() string {
	return g.model
}
