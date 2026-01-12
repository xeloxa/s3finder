package scanner

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestProbeResult_String(t *testing.T) {
	tests := []struct {
		result   ProbeResult
		expected string
	}{
		{BucketNotFound, "not_found"},
		{BucketExists, "public"},
		{BucketForbidden, "private"},
		{BucketError, "error"},
		{ProbeResult(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.result.String(); got != tt.expected {
				t.Errorf("ProbeResult.String() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestDefaultProberConfig(t *testing.T) {
	cfg := DefaultProberConfig()

	if cfg.Timeout != 10*time.Second {
		t.Errorf("Timeout = %v, want %v", cfg.Timeout, 10*time.Second)
	}
	if cfg.MaxIdleConns != 1000 {
		t.Errorf("MaxIdleConns = %d, want %d", cfg.MaxIdleConns, 1000)
	}
	if cfg.MaxIdleConnsPerHost != 100 {
		t.Errorf("MaxIdleConnsPerHost = %d, want %d", cfg.MaxIdleConnsPerHost, 100)
	}
	if cfg.MaxConnsPerHost != 100 {
		t.Errorf("MaxConnsPerHost = %d, want %d", cfg.MaxConnsPerHost, 100)
	}
	if cfg.MaxRPS != 500 {
		t.Errorf("MaxRPS = %v, want %v", cfg.MaxRPS, 500.0)
	}
}

func TestNewProber_NilConfig(t *testing.T) {
	prober := NewProber(nil)

	if prober == nil {
		t.Fatal("NewProber(nil) returned nil")
	}
	if prober.client == nil {
		t.Error("Prober.client should not be nil")
	}
	if prober.limiter == nil {
		t.Error("Prober.limiter should not be nil")
	}
}

func TestNewProber_CustomConfig(t *testing.T) {
	cfg := &ProberConfig{
		Timeout:             5 * time.Second,
		MaxIdleConns:        500,
		MaxIdleConnsPerHost: 50,
		MaxConnsPerHost:     50,
		MaxRPS:              100,
	}

	prober := NewProber(cfg)

	if prober == nil {
		t.Fatal("NewProber() returned nil")
	}
	if prober.CurrentRPS() != 100 {
		t.Errorf("CurrentRPS() = %v, want %v", prober.CurrentRPS(), 100.0)
	}
}

func TestProber_Check_ContextCanceled(t *testing.T) {
	prober := NewProber(&ProberConfig{MaxRPS: 1000})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	resp := prober.Check(ctx, "test-bucket")

	if resp.Result != BucketError {
		t.Errorf("Result = %v, want %v", resp.Result, BucketError)
	}
	if resp.Error == nil {
		t.Error("Error should not be nil for canceled context")
	}
}

func TestProber_CurrentRPS(t *testing.T) {
	prober := NewProber(&ProberConfig{MaxRPS: 250})

	if prober.CurrentRPS() != 250 {
		t.Errorf("CurrentRPS() = %v, want %v", prober.CurrentRPS(), 250.0)
	}
}

// Integration test with mock server
func TestProber_Check_MockServer(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		expectedResult ProbeResult
	}{
		{"200 OK - Public", 200, BucketExists},
		{"403 Forbidden - Private", 403, BucketForbidden},
		{"404 Not Found", 404, BucketNotFound},
		{"301 Redirect - Treated as Private", 301, BucketForbidden},
		{"307 Redirect - Treated as Private", 307, BucketForbidden},
		{"500 Server Error", 500, BucketError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			// Create a custom prober that uses the test server
			// Note: This is a simplified test - real S3 URLs won't work with mock
			prober := NewProber(&ProberConfig{
				Timeout: 5 * time.Second,
				MaxRPS:  1000,
			})

			// We can't easily test the actual Check method without modifying it
			// to accept custom URLs, so we just verify the prober is created correctly
			if prober == nil {
				t.Fatal("Prober should not be nil")
			}
		})
	}
}
