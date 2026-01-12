package scanner

import (
	"testing"
	"time"
)

func TestNewInspector(t *testing.T) {
	tests := []struct {
		name            string
		timeout         time.Duration
		expectedTimeout time.Duration
	}{
		{"positive timeout", 60 * time.Second, 60 * time.Second},
		{"zero timeout defaults to 30s", 0, 30 * time.Second},
		{"negative timeout defaults to 30s", -10 * time.Second, 30 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inspector := NewInspector(tt.timeout)

			if inspector == nil {
				t.Fatal("NewInspector() returned nil")
			}
			if inspector.timeout != tt.expectedTimeout {
				t.Errorf("timeout = %v, want %v", inspector.timeout, tt.expectedTimeout)
			}
		})
	}
}

func TestInspector_parseRegionFromError(t *testing.T) {
	inspector := NewInspector(30 * time.Second)

	tests := []struct {
		name     string
		errMsg   string
		expected string
	}{
		{"us-east-1", "PermanentRedirect to us-east-1", "us-east-1"},
		{"us-west-2", "bucket is in us-west-2 region", "us-west-2"},
		{"eu-west-1", "redirect to eu-west-1.amazonaws.com", "eu-west-1"},
		{"ap-northeast-1", "error in ap-northeast-1", "ap-northeast-1"},
		{"no region found", "some random error message", "us-east-1"},
		{"empty string", "", "us-east-1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := inspector.parseRegionFromError(tt.errMsg)
			if result != tt.expected {
				t.Errorf("parseRegionFromError(%q) = %q, want %q", tt.errMsg, result, tt.expected)
			}
		})
	}
}

func TestInspectResult_Fields(t *testing.T) {
	now := time.Now()
	result := &InspectResult{
		Bucket:      "test-bucket",
		Exists:      true,
		IsPublic:    true,
		ACL:         "public-read",
		Region:      "us-east-1",
		ObjectCount: 100,
		SampleKeys:  []string{"file1.txt", "file2.txt"},
		Timestamp:   now,
	}

	if result.Bucket != "test-bucket" {
		t.Errorf("Bucket = %q, want %q", result.Bucket, "test-bucket")
	}
	if result.Exists != true {
		t.Errorf("Exists = %v, want %v", result.Exists, true)
	}
	if result.IsPublic != true {
		t.Errorf("IsPublic = %v, want %v", result.IsPublic, true)
	}
	if result.ACL != "public-read" {
		t.Errorf("ACL = %q, want %q", result.ACL, "public-read")
	}
	if result.Region != "us-east-1" {
		t.Errorf("Region = %q, want %q", result.Region, "us-east-1")
	}
	if result.ObjectCount != 100 {
		t.Errorf("ObjectCount = %d, want %d", result.ObjectCount, 100)
	}
	if len(result.SampleKeys) != 2 {
		t.Errorf("SampleKeys length = %d, want %d", len(result.SampleKeys), 2)
	}
}

func TestInspectResult_WithError(t *testing.T) {
	result := &InspectResult{
		Bucket: "error-bucket",
		Exists: false,
		Error:  "access denied",
	}

	if result.Error != "access denied" {
		t.Errorf("Error = %q, want %q", result.Error, "access denied")
	}
}
