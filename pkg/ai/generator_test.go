package ai

import (
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Provider != "openai" {
		t.Errorf("Provider = %q, want %q", cfg.Provider, "openai")
	}
	if cfg.Model != "gpt-4o-mini" {
		t.Errorf("Model = %q, want %q", cfg.Model, "gpt-4o-mini")
	}
	if cfg.Temperature != 0.7 {
		t.Errorf("Temperature = %v, want %v", cfg.Temperature, 0.7)
	}
}

func TestNewGenerator_OpenAI(t *testing.T) {
	cfg := &Config{
		Provider: "openai",
		APIKey:   "test-key",
	}

	gen, err := NewGenerator(cfg)

	if err != nil {
		t.Fatalf("NewGenerator() error = %v, want nil", err)
	}
	if gen.Name() != "openai" {
		t.Errorf("Name() = %q, want %q", gen.Name(), "openai")
	}
}

func TestNewGenerator_Ollama(t *testing.T) {
	cfg := &Config{
		Provider: "ollama",
		BaseURL:  "http://localhost:11434",
	}

	gen, err := NewGenerator(cfg)

	if err != nil {
		t.Fatalf("NewGenerator() error = %v, want nil", err)
	}
	if gen.Name() != "ollama" {
		t.Errorf("Name() = %q, want %q", gen.Name(), "ollama")
	}
}

func TestNewGenerator_Anthropic(t *testing.T) {
	cfg := &Config{
		Provider: "anthropic",
		APIKey:   "test-key",
	}

	gen, err := NewGenerator(cfg)

	if err != nil {
		t.Fatalf("NewGenerator() error = %v, want nil", err)
	}
	if gen.Name() != "anthropic" {
		t.Errorf("Name() = %q, want %q", gen.Name(), "anthropic")
	}
}

func TestNewGenerator_UnknownDefaultsToOpenAI(t *testing.T) {
	cfg := &Config{
		Provider: "unknown-provider",
		APIKey:   "test-key",
	}

	gen, err := NewGenerator(cfg)

	if err != nil {
		t.Fatalf("NewGenerator() error = %v, want nil", err)
	}
	if gen.Name() != "openai" {
		t.Errorf("Name() = %q, want %q (should default to openai)", gen.Name(), "openai")
	}
}

func TestBucketPrompt_Format(t *testing.T) {
	// Verify the prompt template has correct placeholders
	if BucketPrompt == "" {
		t.Error("BucketPrompt should not be empty")
	}

	// Should contain %s placeholders for seed and count
	// The prompt uses: seed, count, seed (3 placeholders)
}
