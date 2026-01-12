package output

import (
	"github.com/xeloxa/s3finder/pkg/scanner"
)

// Writer defines the interface for output handlers.
type Writer interface {
	// WriteResult writes a single scan result.
	WriteResult(result *scanner.ScanResult) error

	// Flush ensures all buffered data is written.
	Flush() error

	// Close releases resources.
	Close() error
}

// MultiWriter combines multiple writers.
type MultiWriter struct {
	writers []Writer
}

// NewMultiWriter creates a writer that writes to multiple destinations.
func NewMultiWriter(writers ...Writer) *MultiWriter {
	return &MultiWriter{writers: writers}
}

// WriteResult writes to all underlying writers.
func (m *MultiWriter) WriteResult(result *scanner.ScanResult) error {
	for _, w := range m.writers {
		if err := w.WriteResult(result); err != nil {
			return err
		}
	}
	return nil
}

// Flush flushes all underlying writers.
func (m *MultiWriter) Flush() error {
	for _, w := range m.writers {
		if err := w.Flush(); err != nil {
			return err
		}
	}
	return nil
}

// Close closes all underlying writers.
func (m *MultiWriter) Close() error {
	for _, w := range m.writers {
		if err := w.Close(); err != nil {
			return err
		}
	}
	return nil
}
