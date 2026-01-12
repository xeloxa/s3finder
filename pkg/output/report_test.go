package output

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xeloxa/s3finder/pkg/scanner"
)

func TestNewReport_NilConfig(t *testing.T) {
	_, err := NewReport(nil)

	if err == nil {
		t.Error("NewReport(nil) error = nil, want error")
	}
}

func TestNewReport_InvalidPath(t *testing.T) {
	cfg := &ReportConfig{
		FilePath: "/nonexistent/directory/report.json",
	}

	_, err := NewReport(cfg)

	if err == nil {
		t.Error("NewReport() error = nil, want error for invalid path")
	}
}

func TestNewReport_ValidConfig(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "report.json")
	cfg := &ReportConfig{
		FilePath:  tmpFile,
		Format:    "json",
		StartTime: time.Now(),
	}

	rw, err := NewReport(cfg)

	if err != nil {
		t.Fatalf("NewReport() error = %v, want nil", err)
	}
	defer rw.Close()

	if rw.format != "json" {
		t.Errorf("format = %q, want %q", rw.format, "json")
	}
}

func TestNewReport_DefaultFormat(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "report.json")
	cfg := &ReportConfig{
		FilePath: tmpFile,
		Format:   "", // Empty should default to json
	}

	rw, err := NewReport(cfg)

	if err != nil {
		t.Fatalf("NewReport() error = %v, want nil", err)
	}
	defer rw.Close()

	if rw.format != "json" {
		t.Errorf("format = %q, want %q (default)", rw.format, "json")
	}
}

func TestNewReport_DefaultStartTime(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "report.json")
	cfg := &ReportConfig{
		FilePath: tmpFile,
	}

	before := time.Now()
	rw, err := NewReport(cfg)
	after := time.Now()

	if err != nil {
		t.Fatalf("NewReport() error = %v, want nil", err)
	}
	defer rw.Close()

	if rw.startTime.Before(before) || rw.startTime.After(after) {
		t.Error("startTime should be set to current time when not provided")
	}
}

func TestReportWriter_WriteResult(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "report.json")
	rw, _ := NewReport(&ReportConfig{FilePath: tmpFile})
	defer rw.Close()

	result := &scanner.ScanResult{
		Bucket: "test-bucket",
		Probe:  scanner.BucketExists,
	}

	err := rw.WriteResult(result)

	if err != nil {
		t.Errorf("WriteResult() error = %v, want nil", err)
	}
	if rw.ResultCount() != 1 {
		t.Errorf("ResultCount() = %d, want 1", rw.ResultCount())
	}
}

func TestReportWriter_WriteResult_Multiple(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "report.json")
	rw, _ := NewReport(&ReportConfig{FilePath: tmpFile})
	defer rw.Close()

	for i := 0; i < 5; i++ {
		rw.WriteResult(&scanner.ScanResult{Bucket: "bucket"})
	}

	if rw.ResultCount() != 5 {
		t.Errorf("ResultCount() = %d, want 5", rw.ResultCount())
	}
}

func TestReportWriter_FlushJSON(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "report.json")
	rw, _ := NewReport(&ReportConfig{FilePath: tmpFile, Format: "json"})

	rw.WriteResult(&scanner.ScanResult{Bucket: "public-bucket", Probe: scanner.BucketExists})
	rw.WriteResult(&scanner.ScanResult{Bucket: "private-bucket", Probe: scanner.BucketForbidden})
	rw.Close()

	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to read report file: %v", err)
	}

	var report Report
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatalf("Failed to parse JSON report: %v", err)
	}

	if report.TotalFound != 2 {
		t.Errorf("TotalFound = %d, want 2", report.TotalFound)
	}
	if report.PublicBuckets != 1 {
		t.Errorf("PublicBuckets = %d, want 1", report.PublicBuckets)
	}
	if report.PrivateBuckets != 1 {
		t.Errorf("PrivateBuckets = %d, want 1", report.PrivateBuckets)
	}
	if len(report.Results) != 2 {
		t.Errorf("Results length = %d, want 2", len(report.Results))
	}
}

func TestReportWriter_FlushTXT(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "report.txt")
	rw, _ := NewReport(&ReportConfig{FilePath: tmpFile, Format: "txt"})

	rw.WriteResult(&scanner.ScanResult{
		Bucket: "public-bucket",
		Probe:  scanner.BucketExists,
		Inspect: &scanner.InspectResult{
			Region:      "us-east-1",
			ObjectCount: 100,
		},
	})
	rw.WriteResult(&scanner.ScanResult{
		Bucket: "private-bucket",
		Probe:  scanner.BucketForbidden,
		Inspect: &scanner.InspectResult{
			Region: "eu-west-1",
		},
	})
	rw.Close()

	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to read report file: %v", err)
	}

	content := string(data)

	if !strings.Contains(content, "[PUBLIC] public-bucket") {
		t.Error("TXT report should contain [PUBLIC] public-bucket")
	}
	if !strings.Contains(content, "[PRIVATE] private-bucket") {
		t.Error("TXT report should contain [PRIVATE] private-bucket")
	}
	if !strings.Contains(content, "region: us-east-1") {
		t.Error("TXT report should contain region info")
	}
	if !strings.Contains(content, "objects: 100") {
		t.Error("TXT report should contain object count")
	}
}

func TestReportWriter_FlushTXT_SkipsNotFound(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "report.txt")
	rw, _ := NewReport(&ReportConfig{FilePath: tmpFile, Format: "txt"})

	rw.WriteResult(&scanner.ScanResult{Bucket: "found-bucket", Probe: scanner.BucketExists})
	rw.WriteResult(&scanner.ScanResult{Bucket: "not-found-bucket", Probe: scanner.BucketNotFound})
	rw.Close()

	data, _ := os.ReadFile(tmpFile)
	content := string(data)

	if strings.Contains(content, "not-found-bucket") {
		t.Error("TXT report should not contain not-found buckets")
	}
}

func TestReportWriter_ResultCount_Concurrent(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "report.json")
	rw, _ := NewReport(&ReportConfig{FilePath: tmpFile})
	defer rw.Close()

	done := make(chan bool)

	// Concurrent writes
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				rw.WriteResult(&scanner.ScanResult{Bucket: "bucket"})
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	if rw.ResultCount() != 1000 {
		t.Errorf("ResultCount() = %d, want 1000", rw.ResultCount())
	}
}
