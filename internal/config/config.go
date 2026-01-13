package config

import (
	"os"
	"path/filepath"
)

// Config holds all application configuration.
type Config struct {
	// Scanner settings
	Workers     int     `mapstructure:"workers"`
	MaxRPS      float64 `mapstructure:"max_rps"`
	Timeout     int     `mapstructure:"timeout"` // seconds
	DeepInspect bool    `mapstructure:"deep_inspect"`

	// Input settings
	Seed     string `mapstructure:"seed"`
	Wordlist string `mapstructure:"wordlist"`
	Domain   string `mapstructure:"domain"`
	CTLimit  int    `mapstructure:"ct_limit"`

	// AI settings
	AIEnabled  bool   `mapstructure:"ai_enabled"`
	AIProvider string `mapstructure:"ai_provider"`
	AIModel    string `mapstructure:"ai_model"`
	AIKey      string `mapstructure:"ai_key"`
	AIBaseURL  string `mapstructure:"ai_base_url"`
	AICount    int    `mapstructure:"ai_count"`

	// Output settings
	OutputFile   string `mapstructure:"output_file"`
	OutputFormat string `mapstructure:"output_format"`
	NoColor      bool   `mapstructure:"no_color"`
	Verbose      bool   `mapstructure:"verbose"`
}

// Default returns the default configuration.
func Default() *Config {
	return &Config{
		Workers:      50,
		MaxRPS:       150,
		Timeout:      15,
		DeepInspect:  true,
		Wordlist:     "",
		CTLimit:      100,
		AIEnabled:    false,
		AIProvider:   "openai",
		AIModel:      "gpt-4o-mini",
		AICount:      50,
		OutputFile:   "results.json",
		OutputFormat: "json",
		NoColor:      false,
		Verbose:      false,
	}
}

// FindWordlist looks for a wordlist file in common locations.
func FindWordlist(provided string) string {
	if provided != "" {
		return provided
	}

	// Check relative to executable
	exe, err := os.Executable()
	if err == nil {
		exeDir := filepath.Dir(exe)
		candidate := filepath.Join(exeDir, "wordlists", "common.txt")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	// Check current directory
	candidate := filepath.Join("wordlists", "common.txt")
	if _, err := os.Stat(candidate); err == nil {
		return candidate
	}

	return ""
}

// LoadWordlist reads a wordlist file and returns the words.
func LoadWordlist(path string) ([]string, error) {
	if path == "" {
		return nil, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var words []string
	var current []byte

	for _, b := range data {
		if b == '\n' || b == '\r' {
			if len(current) > 0 {
				words = append(words, string(current))
				current = current[:0]
			}
		} else {
			current = append(current, b)
		}
	}

	if len(current) > 0 {
		words = append(words, string(current))
	}

	return words, nil
}
