package recon

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCleanDomain(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"example.com", "example.com"},
		{"https://example.com", "example.com"},
		{"http://example.com/", "example.com"},
		{"www.example.com", "example.com"},
		{"https://www.example.com/", "example.com"},
		{"EXAMPLE.COM", "example.com"},
	}

	for _, tt := range tests {
		result := cleanDomain(tt.input)
		if result != tt.expected {
			t.Errorf("cleanDomain(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestIsValidBucketSeed(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"dev", true},
		{"staging-api", true},
		{"test123", true},
		{"a", false},       // too short
		{"-dev", false},    // starts with hyphen
		{"dev-", false},    // ends with hyphen
		{"DEV", false},     // uppercase
		{"dev.api", false}, // contains dot
	}

	for _, tt := range tests {
		result := isValidBucketSeed(tt.input)
		if result != tt.expected {
			t.Errorf("isValidBucketSeed(%q) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}

func TestSubdomainsToSeeds(t *testing.T) {
	subdomains := []string{
		"dev.example.com",
		"staging.example.com",
		"api.example.com",
	}
	baseDomain := "example.com"

	seeds := SubdomainsToSeeds(subdomains, baseDomain)

	if len(seeds) == 0 {
		t.Error("SubdomainsToSeeds returned no seeds")
	}

	expectedContains := []string{"dev", "staging", "api"}
	seedMap := make(map[string]bool)
	for _, s := range seeds {
		seedMap[s] = true
	}

	for _, expected := range expectedContains {
		if !seedMap[expected] {
			t.Errorf("Expected seed %q not found in results", expected)
		}
	}
}

func TestGenerateSeedCandidates(t *testing.T) {
	candidates := generateSeedCandidates("dev", "example")

	expected := []string{"dev", "dev-example", "example-dev"}
	if len(candidates) != len(expected) {
		t.Errorf("generateSeedCandidates returned %d candidates, want %d", len(candidates), len(expected))
	}

	for i, exp := range expected {
		if candidates[i] != exp {
			t.Errorf("candidates[%d] = %q, want %q", i, candidates[i], exp)
		}
	}
}

func TestCTClientFetchSubdomains(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[
			{"name_value": "dev.example.com"},
			{"name_value": "staging.example.com\napi.example.com"},
			{"name_value": "*.example.com"}
		]`))
	}))
	defer server.Close()

	client := &CTClient{
		httpClient: &http.Client{Timeout: 5 * time.Second},
		maxResults: 100,
	}

	results := client.extractSubdomains([]CTResult{
		{NameValue: "dev.example.com"},
		{NameValue: "staging.example.com\napi.example.com"},
		{NameValue: "*.example.com"},
	}, "example.com")

	if len(results) < 2 {
		t.Errorf("Expected at least 2 subdomains, got %d", len(results))
	}

	hasDevOrStaging := false
	for _, r := range results {
		if r == "dev.example.com" || r == "staging.example.com" {
			hasDevOrStaging = true
			break
		}
	}
	if !hasDevOrStaging {
		t.Error("Expected to find dev or staging subdomain")
	}
}

func TestCTClientMaxResults(t *testing.T) {
	client := &CTClient{
		httpClient: &http.Client{Timeout: 5 * time.Second},
		maxResults: 2,
	}

	results := client.extractSubdomains([]CTResult{
		{NameValue: "a.example.com"},
		{NameValue: "b.example.com"},
		{NameValue: "c.example.com"},
		{NameValue: "d.example.com"},
	}, "example.com")

	if len(results) != 2 {
		t.Errorf("Expected max 2 results, got %d", len(results))
	}
}
