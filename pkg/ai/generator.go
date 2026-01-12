package ai

import (
	"context"
)

// Generator defines the interface for AI-powered bucket name generation.
// Implementations can use different LLM providers (OpenAI, Ollama, Anthropic, etc.)
type Generator interface {
	// Generate creates bucket name suggestions based on a seed keyword.
	// count specifies the approximate number of names to generate.
	Generate(ctx context.Context, seed string, count int) ([]string, error)

	// Name returns the provider identifier (e.g., "openai", "ollama", "anthropic").
	Name() string

	// Model returns the specific model being used.
	Model() string
}

// Config holds configuration for AI generators.
type Config struct {
	Provider    string  // "openai", "ollama", "anthropic"
	Model       string  // Model name (e.g., "gpt-4o-mini", "llama3", "claude-3-haiku")
	APIKey      string  // API key for cloud providers
	BaseURL     string  // Base URL (for Ollama or custom endpoints)
	Temperature float64 // Creativity level (0.0-1.0, recommended: 0.7)
}

// DefaultConfig returns sensible defaults for AI generation.
func DefaultConfig() *Config {
	return &Config{
		Provider:    "openai",
		Model:       "gpt-4o-mini",
		Temperature: 0.7,
	}
}

// NewGenerator creates a Generator based on the provided configuration.
func NewGenerator(cfg *Config) (Generator, error) {
	switch cfg.Provider {
	case "openai":
		return NewOpenAI(cfg)
	case "ollama":
		return NewOllama(cfg)
	case "anthropic":
		return NewAnthropic(cfg)
	default:
		return NewOpenAI(cfg)
	}
}

// BucketPrompt is the template used to instruct the LLM.
const BucketPrompt = `You are an S3 bucket name generator for security research.

Given the seed keyword "%s", generate %d realistic S3 bucket names that an organization might use.

Rules:
- Names must be valid S3 bucket names (lowercase, 3-63 chars, no underscores)
- Include variations: backups, logs, assets, internal, dev, prod, staging
- Mix patterns: {seed}-{suffix}, {prefix}-{seed}, {seed}-{date}, {seed}-{region}
- Think like a lazy sysadmin: predictable patterns, years, abbreviations
- Consider: databases, configs, uploads, exports, reports, archives
- NO explanations, just bucket names, one per line

Examples for "acme":
acme-backup
acme-prod-assets
acme-internal-2024
dev-acme-logs
acme-us-east-1
acme-db-exports
acme-config-backup

Generate names for: %s`
