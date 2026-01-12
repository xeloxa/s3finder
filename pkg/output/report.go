package output

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/xeloxa/s3finder/pkg/scanner"
)

// Report represents the final scan report.
type Report struct {
	GeneratedAt   time.Time             `json:"generated_at"`
	ScanDuration  string                `json:"scan_duration"`
	TotalScanned  int64                 `json:"total_scanned"`
	TotalFound    int                   `json:"total_found"`
	PublicBuckets int                   `json:"public_buckets"`
	PrivateBuckets int                  `json:"private_buckets"`
	Results       []*scanner.ScanResult `json:"results"`
}

// ReportWriter writes results to a file in JSON or TXT format.
type ReportWriter struct {
	file      *os.File
	format    string
	results   []*scanner.ScanResult
	mu        sync.Mutex
	startTime time.Time
}

// ReportConfig configures the report writer.
type ReportConfig struct {
	FilePath  string
	Format    string // "json" or "txt"
	StartTime time.Time
}

// NewReport creates a new report writer.
func NewReport(cfg *ReportConfig) (*ReportWriter, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

	file, err := os.Create(cfg.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create report file: %w", err)
	}

	format := cfg.Format
	if format == "" {
		format = "json"
	}

	startTime := cfg.StartTime
	if startTime.IsZero() {
		startTime = time.Now()
	}

	return &ReportWriter{
		file:      file,
		format:    format,
		results:   make([]*scanner.ScanResult, 0),
		startTime: startTime,
	}, nil
}

// WriteResult buffers a result for the final report.
func (r *ReportWriter) WriteResult(result *scanner.ScanResult) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.results = append(r.results, result)
	return nil
}

// Flush writes the final report to the file.
func (r *ReportWriter) Flush() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	switch r.format {
	case "json":
		return r.flushJSON()
	case "txt":
		return r.flushTXT()
	default:
		return r.flushJSON()
	}
}

func (r *ReportWriter) flushJSON() error {
	var public, private int
	for _, result := range r.results {
		switch result.Probe {
		case scanner.BucketExists:
			public++
		case scanner.BucketForbidden:
			private++
		}
	}

	report := Report{
		GeneratedAt:   time.Now(),
		ScanDuration:  time.Since(r.startTime).Round(time.Second).String(),
		TotalFound:    len(r.results),
		PublicBuckets: public,
		PrivateBuckets: private,
		Results:       r.results,
	}

	encoder := json.NewEncoder(r.file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

func (r *ReportWriter) flushTXT() error {
	for _, result := range r.results {
		var line string
		switch result.Probe {
		case scanner.BucketExists:
			line = fmt.Sprintf("[PUBLIC] %s", result.Bucket)
		case scanner.BucketForbidden:
			line = fmt.Sprintf("[PRIVATE] %s", result.Bucket)
		default:
			continue
		}

		if result.Inspect != nil {
			if result.Inspect.Region != "" && result.Inspect.Region != "unknown" {
				line += fmt.Sprintf(" | region: %s", result.Inspect.Region)
			}
			if result.Inspect.ObjectCount > 0 {
				line += fmt.Sprintf(" | objects: %d", result.Inspect.ObjectCount)
			}
			if len(result.Inspect.SampleKeys) > 0 {
				line += fmt.Sprintf(" | sample: %v", result.Inspect.SampleKeys[:min(3, len(result.Inspect.SampleKeys))])
			}
		}

		fmt.Fprintln(r.file, line)
	}

	return nil
}

// Close closes the report file.
func (r *ReportWriter) Close() error {
	if err := r.Flush(); err != nil {
		r.file.Close()
		return err
	}
	return r.file.Close()
}

// ResultCount returns the number of buffered results.
func (r *ReportWriter) ResultCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.results)
}
