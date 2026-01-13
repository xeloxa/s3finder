package output

import (
	"fmt"
	"io"
	"os"
	"sync"
	"sync/atomic"

	"github.com/xeloxa/s3finder/pkg/scanner"
)

// Colors for terminal output
const (
	colorReset  = "\033[0m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorRed    = "\033[31m"
	colorCyan   = "\033[36m"
	colorGray   = "\033[90m"
)

// OSC 8 hyperlink escape sequences
// Format: \033]8;;URL\033\\TEXT\033]8;;\033\\
const (
	hyperlinkStart = "\033]8;;"
	hyperlinkMid   = "\033\\"
	hyperlinkEnd   = "\033]8;;\033\\"
)

// RealtimeWriter outputs results to terminal with colors.
type RealtimeWriter struct {
	out        io.Writer
	mu         sync.Mutex
	useColors  bool
	useLinks   bool
	verbose    bool
	progress   *Progress // Reference to progress bar for coordinated output

	// Counters for summary
	found   int64
	public  int64
	private int64
	errors  int64
}

// RealtimeConfig configures the realtime writer.
type RealtimeConfig struct {
	Output     io.Writer
	UseColors  bool
	UseLinks   bool // Enable clickable hyperlinks
	Verbose    bool
	Progress   *Progress // Optional progress bar for coordinated output
}

// NewRealtime creates a new realtime terminal writer.
func NewRealtime(cfg *RealtimeConfig) *RealtimeWriter {
	if cfg == nil {
		cfg = &RealtimeConfig{
			Output:    os.Stdout,
			UseColors: true,
			UseLinks:  true,
			Verbose:   false,
		}
	}

	out := cfg.Output
	if out == nil {
		out = os.Stdout
	}

	return &RealtimeWriter{
		out:       out,
		useColors: cfg.UseColors,
		useLinks:  cfg.UseLinks,
		verbose:   cfg.Verbose,
		progress:  cfg.Progress,
	}
}

// WriteResult outputs a result to the terminal.
func (r *RealtimeWriter) WriteResult(result *scanner.ScanResult) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	atomic.AddInt64(&r.found, 1)

	var line string
	switch result.Probe {
	case scanner.BucketExists:
		atomic.AddInt64(&r.public, 1)
		line = r.formatPublic(result)
	case scanner.BucketForbidden:
		atomic.AddInt64(&r.private, 1)
		line = r.formatPrivate(result)
	case scanner.BucketError:
		atomic.AddInt64(&r.errors, 1)
		if r.verbose {
			line = r.formatError(result)
		}
	default:
		return nil
	}

	if line != "" {
		if r.progress != nil {
			// Use progress bar's PrintAbove to maintain progress at bottom
			r.progress.PrintAbove(line)
		} else {
			fmt.Fprintln(r.out, line)
		}
	}

	return nil
}

func (r *RealtimeWriter) formatPublic(result *scanner.ScanResult) string {
	tag := "[PUBLIC]"
	if r.useColors {
		tag = colorGreen + tag + colorReset
	}

	// Build bucket URL
	bucketURL := fmt.Sprintf("https://%s.s3.amazonaws.com", result.Bucket)
	bucketDisplay := result.Bucket

	// Make bucket name clickable if links enabled
	if r.useLinks {
		bucketDisplay = r.makeHyperlink(bucketURL, result.Bucket)
	}

	details := ""
	if result.Inspect != nil {
		if result.Inspect.ObjectCount > 0 {
			details = fmt.Sprintf(" (objects: %d, region: %s)", result.Inspect.ObjectCount, result.Inspect.Region)
		} else if result.Inspect.ObjectCount == -2 {
			details = fmt.Sprintf(" (objects: 100+, region: %s)", result.Inspect.Region)
		} else {
			details = fmt.Sprintf(" (region: %s)", result.Inspect.Region)
		}
	}

	if r.useColors {
		details = colorGray + details + colorReset
	}

	// For PUBLIC buckets, always show the URL on a new line
	urlLine := ""
	if r.useColors {
		urlLine = fmt.Sprintf("\n         %s%s%s", colorCyan, bucketURL, colorReset)
	} else {
		urlLine = fmt.Sprintf("\n         %s", bucketURL)
	}

	return fmt.Sprintf("%s %s%s%s", tag, bucketDisplay, details, urlLine)
}

func (r *RealtimeWriter) formatPrivate(result *scanner.ScanResult) string {
	tag := "[PRIVATE]"
	if r.useColors {
		tag = colorYellow + tag + colorReset
	}

	// Build bucket URL
	bucketURL := fmt.Sprintf("https://%s.s3.amazonaws.com", result.Bucket)
	bucketDisplay := result.Bucket

	// Make bucket name clickable if links enabled
	if r.useLinks {
		bucketDisplay = r.makeHyperlink(bucketURL, result.Bucket)
	}

	details := ""
	if result.Inspect != nil && result.Inspect.Region != "" && result.Inspect.Region != "unknown" {
		details = fmt.Sprintf(" (region: %s)", result.Inspect.Region)
		if r.useColors {
			details = colorGray + details + colorReset
		}
	}

	return fmt.Sprintf("%s %s%s", tag, bucketDisplay, details)
}

// makeHyperlink creates an OSC 8 terminal hyperlink
// Supported by: iTerm2, Windows Terminal, GNOME Terminal, Konsole, etc.
func (r *RealtimeWriter) makeHyperlink(url, text string) string {
	return hyperlinkStart + url + hyperlinkMid + text + hyperlinkEnd
}

func (r *RealtimeWriter) formatError(result *scanner.ScanResult) string {
	tag := "[ERROR]"
	if r.useColors {
		tag = colorRed + tag + colorReset
	}
	msg := result.Bucket
	if r.verbose && result.Error != "" {
		msg = fmt.Sprintf("%s (%s)", result.Bucket, result.Error)
	}
	return fmt.Sprintf("%s %s", tag, msg)
}

// Flush implements Writer.
func (r *RealtimeWriter) Flush() error {
	return nil
}

// Close implements Writer.
func (r *RealtimeWriter) Close() error {
	return nil
}

// Stats returns current counts.
func (r *RealtimeWriter) Stats() (found, public, private, errors int64) {
	return atomic.LoadInt64(&r.found),
		atomic.LoadInt64(&r.public),
		atomic.LoadInt64(&r.private),
		atomic.LoadInt64(&r.errors)
}
