package scanner

import (
	"context"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Workers != 10 {
		t.Errorf("Workers = %d, want %d", cfg.Workers, 10)
	}
	if cfg.MaxRPS != 50 {
		t.Errorf("MaxRPS = %v, want %v", cfg.MaxRPS, 50.0)
	}
	if cfg.Timeout != 10*time.Second {
		t.Errorf("Timeout = %v, want %v", cfg.Timeout, 10*time.Second)
	}
	if cfg.DeepInspect != true {
		t.Errorf("DeepInspect = %v, want %v", cfg.DeepInspect, true)
	}
}

func TestNew_NilConfig(t *testing.T) {
	scanner := New(nil)

	if scanner == nil {
		t.Fatal("New(nil) returned nil")
	}
	if scanner.prober == nil {
		t.Error("Scanner.prober should not be nil")
	}
	if scanner.inspector == nil {
		t.Error("Scanner.inspector should not be nil")
	}
	if scanner.workers != 10 {
		t.Errorf("workers = %d, want %d (default)", scanner.workers, 10)
	}
}

func TestNew_CustomConfig(t *testing.T) {
	cfg := &Config{
		Workers:     50,
		MaxRPS:      200,
		Timeout:     5 * time.Second,
		DeepInspect: false,
	}

	scanner := New(cfg)

	if scanner.workers != 50 {
		t.Errorf("workers = %d, want %d", scanner.workers, 50)
	}
	if scanner.deepInspect != false {
		t.Errorf("deepInspect = %v, want %v", scanner.deepInspect, false)
	}
}

func TestScanner_Stats_Initial(t *testing.T) {
	scanner := New(nil)
	stats := scanner.Stats()

	if stats.Total != 0 {
		t.Errorf("Total = %d, want 0", stats.Total)
	}
	if stats.Scanned != 0 {
		t.Errorf("Scanned = %d, want 0", stats.Scanned)
	}
	if stats.Found != 0 {
		t.Errorf("Found = %d, want 0", stats.Found)
	}
}

func TestScanner_Scan_EmptyList(t *testing.T) {
	scanner := New(&Config{Workers: 2, MaxRPS: 100, Timeout: time.Second})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	results := scanner.Scan(ctx, []string{})

	count := 0
	for range results {
		count++
	}

	if count != 0 {
		t.Errorf("Received %d results, want 0 for empty input", count)
	}
}

func TestScanner_Scan_ContextCanceled(t *testing.T) {
	scanner := New(&Config{Workers: 2, MaxRPS: 100, Timeout: time.Second})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	names := []string{"bucket1", "bucket2", "bucket3"}
	results := scanner.Scan(ctx, names)

	// Should complete without hanging
	for range results {
		// Drain channel
	}
}

func TestScanner_Results(t *testing.T) {
	scanner := New(nil)

	resultsChan := scanner.Results()

	if resultsChan == nil {
		t.Error("Results() should not return nil")
	}
}

func TestScanner_CurrentRPS(t *testing.T) {
	scanner := New(&Config{MaxRPS: 250})

	rps := scanner.CurrentRPS()

	if rps != 250 {
		t.Errorf("CurrentRPS() = %v, want %v", rps, 250.0)
	}
}

func TestScanResult_Fields(t *testing.T) {
	result := &ScanResult{
		Bucket:    "test-bucket",
		Probe:     BucketExists,
		Timestamp: time.Now(),
	}

	if result.Bucket != "test-bucket" {
		t.Errorf("Bucket = %q, want %q", result.Bucket, "test-bucket")
	}
	if result.Probe != BucketExists {
		t.Errorf("Probe = %v, want %v", result.Probe, BucketExists)
	}
}

func TestStats_Fields(t *testing.T) {
	now := time.Now()
	stats := Stats{
		Total:     100,
		Scanned:   50,
		Found:     5,
		Public:    3,
		Private:   2,
		Errors:    1,
		StartTime: now,
	}

	if stats.Total != 100 {
		t.Errorf("Total = %d, want %d", stats.Total, 100)
	}
	if stats.Scanned != 50 {
		t.Errorf("Scanned = %d, want %d", stats.Scanned, 50)
	}
	if stats.Found != 5 {
		t.Errorf("Found = %d, want %d", stats.Found, 5)
	}
	if stats.Public != 3 {
		t.Errorf("Public = %d, want %d", stats.Public, 3)
	}
	if stats.Private != 2 {
		t.Errorf("Private = %d, want %d", stats.Private, 2)
	}
	if stats.Errors != 1 {
		t.Errorf("Errors = %d, want %d", stats.Errors, 1)
	}
}
