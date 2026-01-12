package output

import (
	"errors"
	"testing"

	"github.com/xeloxa/s3finder/pkg/scanner"
)

// mockWriter implements Writer for testing
type mockWriter struct {
	results    []*scanner.ScanResult
	flushErr   error
	closeErr   error
	writeErr   error
	flushCount int
	closeCount int
}

func (m *mockWriter) WriteResult(result *scanner.ScanResult) error {
	if m.writeErr != nil {
		return m.writeErr
	}
	m.results = append(m.results, result)
	return nil
}

func (m *mockWriter) Flush() error {
	m.flushCount++
	return m.flushErr
}

func (m *mockWriter) Close() error {
	m.closeCount++
	return m.closeErr
}

func TestNewMultiWriter(t *testing.T) {
	w1 := &mockWriter{}
	w2 := &mockWriter{}

	mw := NewMultiWriter(w1, w2)

	if mw == nil {
		t.Fatal("NewMultiWriter() returned nil")
	}
	if len(mw.writers) != 2 {
		t.Errorf("NewMultiWriter() has %d writers, want 2", len(mw.writers))
	}
}

func TestNewMultiWriter_Empty(t *testing.T) {
	mw := NewMultiWriter()

	if mw == nil {
		t.Fatal("NewMultiWriter() returned nil")
	}
	if len(mw.writers) != 0 {
		t.Errorf("NewMultiWriter() has %d writers, want 0", len(mw.writers))
	}
}

func TestMultiWriter_WriteResult(t *testing.T) {
	w1 := &mockWriter{}
	w2 := &mockWriter{}
	mw := NewMultiWriter(w1, w2)

	result := &scanner.ScanResult{
		Bucket: "test-bucket",
		Probe:  scanner.BucketExists,
	}

	err := mw.WriteResult(result)

	if err != nil {
		t.Errorf("WriteResult() error = %v, want nil", err)
	}
	if len(w1.results) != 1 {
		t.Errorf("w1 has %d results, want 1", len(w1.results))
	}
	if len(w2.results) != 1 {
		t.Errorf("w2 has %d results, want 1", len(w2.results))
	}
}

func TestMultiWriter_WriteResult_Error(t *testing.T) {
	expectedErr := errors.New("write error")
	w1 := &mockWriter{writeErr: expectedErr}
	w2 := &mockWriter{}
	mw := NewMultiWriter(w1, w2)

	result := &scanner.ScanResult{Bucket: "test-bucket"}

	err := mw.WriteResult(result)

	if err != expectedErr {
		t.Errorf("WriteResult() error = %v, want %v", err, expectedErr)
	}
	// w2 should not receive the result since w1 failed
	if len(w2.results) != 0 {
		t.Errorf("w2 has %d results, want 0 (should stop on first error)", len(w2.results))
	}
}

func TestMultiWriter_Flush(t *testing.T) {
	w1 := &mockWriter{}
	w2 := &mockWriter{}
	mw := NewMultiWriter(w1, w2)

	err := mw.Flush()

	if err != nil {
		t.Errorf("Flush() error = %v, want nil", err)
	}
	if w1.flushCount != 1 {
		t.Errorf("w1.flushCount = %d, want 1", w1.flushCount)
	}
	if w2.flushCount != 1 {
		t.Errorf("w2.flushCount = %d, want 1", w2.flushCount)
	}
}

func TestMultiWriter_Flush_Error(t *testing.T) {
	expectedErr := errors.New("flush error")
	w1 := &mockWriter{flushErr: expectedErr}
	w2 := &mockWriter{}
	mw := NewMultiWriter(w1, w2)

	err := mw.Flush()

	if err != expectedErr {
		t.Errorf("Flush() error = %v, want %v", err, expectedErr)
	}
}

func TestMultiWriter_Close(t *testing.T) {
	w1 := &mockWriter{}
	w2 := &mockWriter{}
	mw := NewMultiWriter(w1, w2)

	err := mw.Close()

	if err != nil {
		t.Errorf("Close() error = %v, want nil", err)
	}
	if w1.closeCount != 1 {
		t.Errorf("w1.closeCount = %d, want 1", w1.closeCount)
	}
	if w2.closeCount != 1 {
		t.Errorf("w2.closeCount = %d, want 1", w2.closeCount)
	}
}

func TestMultiWriter_Close_Error(t *testing.T) {
	expectedErr := errors.New("close error")
	w1 := &mockWriter{closeErr: expectedErr}
	w2 := &mockWriter{}
	mw := NewMultiWriter(w1, w2)

	err := mw.Close()

	if err != expectedErr {
		t.Errorf("Close() error = %v, want %v", err, expectedErr)
	}
}
