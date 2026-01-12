package ratelimit

import (
	"context"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name     string
		maxRPS   float64
		expected float64
	}{
		{"positive RPS", 500, 500},
		{"zero RPS defaults to 100", 0, 100},
		{"negative RPS defaults to 100", -10, 100},
		{"small RPS", 10, 10},
		{"large RPS", 10000, 10000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limiter := New(tt.maxRPS)
			if limiter.MaxRPS() != tt.expected {
				t.Errorf("MaxRPS() = %v, want %v", limiter.MaxRPS(), tt.expected)
			}
			if limiter.CurrentRPS() != tt.expected {
				t.Errorf("CurrentRPS() = %v, want %v", limiter.CurrentRPS(), tt.expected)
			}
		})
	}
}

func TestWait(t *testing.T) {
	limiter := New(1000)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Should not block with high RPS
	err := limiter.Wait(ctx)
	if err != nil {
		t.Errorf("Wait() error = %v, want nil", err)
	}
}

func TestWait_ContextCanceled(t *testing.T) {
	limiter := New(1) // Very low RPS

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := limiter.Wait(ctx)
	if err == nil {
		t.Error("Wait() error = nil, want context canceled error")
	}
}

func TestRecordResponse_Success(t *testing.T) {
	limiter := New(200) // maxRPS must be higher than initial to allow increase

	// Record 100 successful responses
	for i := 0; i < 100; i++ {
		limiter.RecordResponse(200)
	}

	// Should increase by 10% (200 * 1.1 = 220, but capped at maxRPS 200)
	// Actually starts at 200, so no increase possible
	// Let's test with a limiter that has room to grow
	limiter2 := New(500)
	// First decrease it
	limiter2.RecordResponse(429)
	limiter2.RecordResponse(429)
	limiter2.RecordResponse(429) // Now at 250

	// Record 100 successful responses
	for i := 0; i < 100; i++ {
		limiter2.RecordResponse(200)
	}

	// Should increase by 10% from 250 to 275
	expected := 275.0
	if limiter2.CurrentRPS() != expected {
		t.Errorf("CurrentRPS() after 100 successes = %v, want %v", limiter2.CurrentRPS(), expected)
	}
}

func TestRecordResponse_SuccessCapAtMax(t *testing.T) {
	limiter := New(100)

	// Record many successful responses
	for i := 0; i < 1000; i++ {
		limiter.RecordResponse(200)
	}

	// Should not exceed maxRPS
	if limiter.CurrentRPS() > limiter.MaxRPS() {
		t.Errorf("CurrentRPS() = %v, should not exceed MaxRPS() = %v", limiter.CurrentRPS(), limiter.MaxRPS())
	}
}

func TestRecordResponse_Throttle429(t *testing.T) {
	limiter := New(100)

	// Record 3 consecutive 429 responses
	limiter.RecordResponse(429)
	limiter.RecordResponse(429)
	limiter.RecordResponse(429)

	// Should decrease by 50%
	expected := 50.0
	if limiter.CurrentRPS() != expected {
		t.Errorf("CurrentRPS() after 3x 429 = %v, want %v", limiter.CurrentRPS(), expected)
	}
}

func TestRecordResponse_Throttle503(t *testing.T) {
	limiter := New(100)

	// Record 3 consecutive 503 responses
	limiter.RecordResponse(503)
	limiter.RecordResponse(503)
	limiter.RecordResponse(503)

	// Should decrease by 50%
	expected := 50.0
	if limiter.CurrentRPS() != expected {
		t.Errorf("CurrentRPS() after 3x 503 = %v, want %v", limiter.CurrentRPS(), expected)
	}
}

func TestRecordResponse_ThrottleFloor(t *testing.T) {
	limiter := New(20)

	// Record many throttle responses to hit floor
	for i := 0; i < 30; i++ {
		limiter.RecordResponse(429)
	}

	// Should not go below 10 RPS
	if limiter.CurrentRPS() < 10 {
		t.Errorf("CurrentRPS() = %v, should not go below 10", limiter.CurrentRPS())
	}
}

func TestRecordResponse_ThrottleResetOnSuccess(t *testing.T) {
	limiter := New(100)

	// Record 2 throttle responses
	limiter.RecordResponse(429)
	limiter.RecordResponse(429)

	// Record a success - should reset consecutive counter
	limiter.RecordResponse(200)

	// Record 2 more throttle responses - should not trigger decrease yet
	limiter.RecordResponse(429)
	limiter.RecordResponse(429)

	// Should still be at 100 (no decrease triggered)
	if limiter.CurrentRPS() != 100 {
		t.Errorf("CurrentRPS() = %v, want 100 (throttle counter should have reset)", limiter.CurrentRPS())
	}
}

func TestRecordResponse_MixedStatusCodes(t *testing.T) {
	limiter := New(100)

	// Mix of status codes
	codes := []int{200, 404, 403, 200, 200, 301, 200}
	for _, code := range codes {
		limiter.RecordResponse(code)
	}

	// Should still be at initial RPS (not enough successes for increase)
	if limiter.CurrentRPS() != 100 {
		t.Errorf("CurrentRPS() = %v, want 100", limiter.CurrentRPS())
	}
}

func TestCurrentRPS_Concurrent(t *testing.T) {
	limiter := New(500)

	done := make(chan bool)

	// Concurrent reads
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				_ = limiter.CurrentRPS()
			}
			done <- true
		}()
	}

	// Concurrent writes
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				limiter.RecordResponse(200)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 20; i++ {
		<-done
	}

	// Should not panic and RPS should be valid
	rps := limiter.CurrentRPS()
	if rps < 10 || rps > 500 {
		t.Errorf("CurrentRPS() = %v, should be between 10 and 500", rps)
	}
}
