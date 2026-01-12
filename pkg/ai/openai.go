package ai

import (
	"context"
	"fmt"
	"strings"

	"github.com/sashabaranov/go-openai"
)

// OpenAI implements Generator using OpenAI's API.
type OpenAI struct {
	client      *openai.Client
	model       string
	temperature float64
}

// NewOpenAI creates an OpenAI generator.
func NewOpenAI(cfg *Config) (*OpenAI, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("OpenAI API key is required")
	}

	clientCfg := openai.DefaultConfig(cfg.APIKey)
	if cfg.BaseURL != "" {
		clientCfg.BaseURL = cfg.BaseURL
	}

	model := cfg.Model
	if model == "" {
		model = "gpt-4o-mini"
	}

	temp := cfg.Temperature
	if temp <= 0 {
		temp = 0.7
	}

	return &OpenAI{
		client:      openai.NewClientWithConfig(clientCfg),
		model:       model,
		temperature: temp,
	}, nil
}

// Generate creates bucket names using OpenAI.
func (o *OpenAI) Generate(ctx context.Context, seed string, count int) ([]string, error) {
	prompt := fmt.Sprintf(BucketPrompt, seed, count, seed)

	resp, err := o.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:       o.model,
		Temperature: float32(o.temperature),
		MaxTokens:   2000,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: prompt,
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("OpenAI API error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from OpenAI")
	}

	return parseBucketNames(resp.Choices[0].Message.Content), nil
}

// Name returns the provider name.
func (o *OpenAI) Name() string {
	return "openai"
}

// Model returns the model name.
func (o *OpenAI) Model() string {
	return o.model
}

// parseBucketNames extracts valid bucket names from LLM response.
func parseBucketNames(content string) []string {
	lines := strings.Split(content, "\n")
	var names []string

	for _, line := range lines {
		name := strings.TrimSpace(line)
		name = strings.ToLower(name)

		// Skip empty lines and lines that look like explanations
		if name == "" || strings.Contains(name, " ") {
			continue
		}

		// Remove any markdown formatting
		name = strings.TrimPrefix(name, "- ")
		name = strings.TrimPrefix(name, "* ")
		name = strings.Trim(name, "`")

		// Basic validation
		if len(name) >= 3 && len(name) <= 63 {
			names = append(names, name)
		}
	}

	return names
}
