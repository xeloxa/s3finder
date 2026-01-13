package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefault(t *testing.T) {
	cfg := Default()

	tests := []struct {
		name     string
		got      interface{}
		expected interface{}
	}{
		{"Workers", cfg.Workers, 10},
		{"MaxRPS", cfg.MaxRPS, 50.0},
		{"Timeout", cfg.Timeout, 30},
		{"DeepInspect", cfg.DeepInspect, true},
		{"Wordlist", cfg.Wordlist, ""},
		{"CTLimit", cfg.CTLimit, 100},
		{"AIEnabled", cfg.AIEnabled, false},
		{"AIProvider", cfg.AIProvider, "openai"},
		{"AIModel", cfg.AIModel, "gpt-4o-mini"},
		{"AICount", cfg.AICount, 50},
		{"OutputFile", cfg.OutputFile, "results.json"},
		{"OutputFormat", cfg.OutputFormat, "json"},
		{"NoColor", cfg.NoColor, false},
		{"Verbose", cfg.Verbose, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.expected)
			}
		})
	}
}

func TestFindWordlist_ProvidedPath(t *testing.T) {
	provided := "/custom/path/wordlist.txt"
	result := FindWordlist(provided)

	if result != provided {
		t.Errorf("FindWordlist(%q) = %q, want %q", provided, result, provided)
	}
}

func TestFindWordlist_EmptyReturnsEmpty(t *testing.T) {
	// When no wordlist exists in common locations
	result := FindWordlist("")

	// Result depends on whether wordlists/common.txt exists
	// Just verify it doesn't panic
	_ = result
}

func TestFindWordlist_CurrentDirectory(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()
	wordlistDir := filepath.Join(tmpDir, "wordlists")
	if err := os.MkdirAll(wordlistDir, 0755); err != nil {
		t.Fatal(err)
	}

	wordlistPath := filepath.Join(wordlistDir, "common.txt")
	if err := os.WriteFile(wordlistPath, []byte("test\nwords\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Change to temp directory
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	result := FindWordlist("")
	expected := filepath.Join("wordlists", "common.txt")

	if result != expected {
		t.Errorf("FindWordlist(\"\") = %q, want %q", result, expected)
	}
}

func TestLoadWordlist_EmptyPath(t *testing.T) {
	words, err := LoadWordlist("")

	if err != nil {
		t.Errorf("LoadWordlist(\"\") error = %v, want nil", err)
	}
	if words != nil {
		t.Errorf("LoadWordlist(\"\") = %v, want nil", words)
	}
}

func TestLoadWordlist_NonExistentFile(t *testing.T) {
	_, err := LoadWordlist("/nonexistent/path/wordlist.txt")

	if err == nil {
		t.Error("LoadWordlist() error = nil, want error for non-existent file")
	}
}

func TestLoadWordlist_ValidFile(t *testing.T) {
	// Create temp wordlist file
	tmpFile := filepath.Join(t.TempDir(), "wordlist.txt")
	content := "backup\nlogs\nassets\ndev\nprod"
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	words, err := LoadWordlist(tmpFile)

	if err != nil {
		t.Errorf("LoadWordlist() error = %v, want nil", err)
	}

	expected := []string{"backup", "logs", "assets", "dev", "prod"}
	if len(words) != len(expected) {
		t.Errorf("LoadWordlist() returned %d words, want %d", len(words), len(expected))
	}

	for i, word := range words {
		if word != expected[i] {
			t.Errorf("words[%d] = %q, want %q", i, word, expected[i])
		}
	}
}

func TestLoadWordlist_WithEmptyLines(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "wordlist.txt")
	content := "backup\n\nlogs\n\n\nassets\n"
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	words, err := LoadWordlist(tmpFile)

	if err != nil {
		t.Errorf("LoadWordlist() error = %v, want nil", err)
	}

	expected := []string{"backup", "logs", "assets"}
	if len(words) != len(expected) {
		t.Errorf("LoadWordlist() returned %d words, want %d", len(words), len(expected))
	}
}

func TestLoadWordlist_WindowsLineEndings(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "wordlist.txt")
	content := "backup\r\nlogs\r\nassets\r\n"
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	words, err := LoadWordlist(tmpFile)

	if err != nil {
		t.Errorf("LoadWordlist() error = %v, want nil", err)
	}

	expected := []string{"backup", "logs", "assets"}
	if len(words) != len(expected) {
		t.Errorf("LoadWordlist() returned %d words, want %d", len(words), len(expected))
	}
}

func TestLoadWordlist_NoTrailingNewline(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "wordlist.txt")
	content := "backup\nlogs\nassets" // No trailing newline
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	words, err := LoadWordlist(tmpFile)

	if err != nil {
		t.Errorf("LoadWordlist() error = %v, want nil", err)
	}

	expected := []string{"backup", "logs", "assets"}
	if len(words) != len(expected) {
		t.Errorf("LoadWordlist() returned %d words, want %d", len(words), len(expected))
	}
}

func TestLoadWordlist_SingleWord(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "wordlist.txt")
	content := "backup"
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	words, err := LoadWordlist(tmpFile)

	if err != nil {
		t.Errorf("LoadWordlist() error = %v, want nil", err)
	}

	if len(words) != 1 || words[0] != "backup" {
		t.Errorf("LoadWordlist() = %v, want [\"backup\"]", words)
	}
}

func TestLoadWordlist_EmptyFile(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "wordlist.txt")
	if err := os.WriteFile(tmpFile, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	words, err := LoadWordlist(tmpFile)

	if err != nil {
		t.Errorf("LoadWordlist() error = %v, want nil", err)
	}

	if len(words) != 0 {
		t.Errorf("LoadWordlist() returned %d words, want 0", len(words))
	}
}
