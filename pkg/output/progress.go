package output

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Progress bar colors
const (
	progressColorBar     = "\033[36m" // Cyan
	progressColorLabel   = "\033[37m" // White
	progressColorValue   = "\033[97m" // Bright white
	progressColorPublic  = "\033[32m" // Green
	progressColorPrivate = "\033[33m" // Yellow
	progressColorError   = "\033[31m" // Red
	progressColorReset   = "\033[0m"
)

// ProgressConfig configures the progress display.
type ProgressConfig struct {
	Output        io.Writer
	Total         int64
	RefreshRate   time.Duration
	ShowRPS       bool
	UseColors     bool
	BarWidth      int
}

// Progress displays real-time scanning progress.
type Progress struct {
	cfg          ProgressConfig
	total        int64
	scanned      atomic.Int64
	found        atomic.Int64
	public       atomic.Int64
	private      atomic.Int64
	errors       atomic.Int64
	currentRPS   atomic.Value // float64
	startTime    time.Time
	stopChan     chan struct{}
	doneChan     chan struct{}
	mu           sync.Mutex
	lastLineLen  int
	msgOut       io.Writer // Separate output for messages (stdout)
}

// NewProgress creates a new progress display.
func NewProgress(cfg *ProgressConfig) *Progress {
	if cfg == nil {
		cfg = &ProgressConfig{}
	}

	out := cfg.Output
	if out == nil {
		out = os.Stderr
	}

	refreshRate := cfg.RefreshRate
	if refreshRate == 0 {
		refreshRate = 100 * time.Millisecond
	}

	barWidth := cfg.BarWidth
	if barWidth == 0 {
		barWidth = 30
	}

	p := &Progress{
		cfg: ProgressConfig{
			Output:      out,
			Total:       cfg.Total,
			RefreshRate: refreshRate,
			ShowRPS:     cfg.ShowRPS,
			UseColors:   cfg.UseColors,
			BarWidth:    barWidth,
		},
		total:     cfg.Total,
		startTime: time.Now(),
		stopChan:  make(chan struct{}),
		doneChan:  make(chan struct{}),
		msgOut:    os.Stdout, // Messages go to stdout
	}
	p.currentRPS.Store(float64(0))

	return p
}

// Start begins the progress display refresh loop.
func (p *Progress) Start() {
	go p.refreshLoop()
}

// Stop stops the progress display and prints final state.
func (p *Progress) Stop() {
	close(p.stopChan)
	<-p.doneChan
}

// Update updates the progress counters.
func (p *Progress) Update(scanned, found, public, private, errors int64, rps float64) {
	p.scanned.Store(scanned)
	p.found.Store(found)
	p.public.Store(public)
	p.private.Store(private)
	p.errors.Store(errors)
	p.currentRPS.Store(rps)
}

// Increment increments a specific counter.
func (p *Progress) Increment(counter string) {
	switch counter {
	case "scanned":
		p.scanned.Add(1)
	case "found":
		p.found.Add(1)
	case "public":
		p.public.Add(1)
	case "private":
		p.private.Add(1)
	case "errors":
		p.errors.Add(1)
	}
}

// PrintAbove prints a message above the progress bar.
// It clears the progress line, prints the message, then re-renders the progress bar.
func (p *Progress) PrintAbove(msg string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Clear the current progress line on stderr
	if p.lastLineLen > 0 {
		fmt.Fprint(p.cfg.Output, "\r")
		fmt.Fprint(p.cfg.Output, strings.Repeat(" ", p.lastLineLen))
		fmt.Fprint(p.cfg.Output, "\r")
	}

	// Print the message to stdout (goes above the progress bar)
	fmt.Fprintln(p.msgOut, msg)

	// Re-render progress bar below the message
	p.renderUnlocked(false)
}

func (p *Progress) refreshLoop() {
	defer close(p.doneChan)

	ticker := time.NewTicker(p.cfg.RefreshRate)
	defer ticker.Stop()

	for {
		select {
		case <-p.stopChan:
			p.render(true)
			return
		case <-ticker.C:
			p.render(false)
		}
	}
}

func (p *Progress) render(final bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.renderUnlocked(final)
}

func (p *Progress) renderUnlocked(final bool) {
	scanned := p.scanned.Load()
	_ = p.found.Load() // Currently unused but tracked for future use
	public := p.public.Load()
	private := p.private.Load()
	errors := p.errors.Load()
	rps := p.currentRPS.Load().(float64)
	elapsed := time.Since(p.startTime)

	// Calculate percentage
	var pct float64
	if p.total > 0 {
		pct = float64(scanned) / float64(p.total) * 100
	}

	// Calculate ETA
	var eta string
	if scanned > 0 && rps > 0 {
		remaining := p.total - scanned
		etaSeconds := float64(remaining) / rps
		if etaSeconds < 60 {
			eta = fmt.Sprintf("%.0fs", etaSeconds)
		} else if etaSeconds < 3600 {
			eta = fmt.Sprintf("%.1fm", etaSeconds/60)
		} else {
			eta = fmt.Sprintf("%.1fh", etaSeconds/3600)
		}
	} else {
		eta = "--"
	}

	// Build progress bar
	bar := p.buildBar(pct)

	// Build stats line
	var statsLine string
	if p.cfg.UseColors {
		statsLine = fmt.Sprintf(
			"%s%s%s %s%.1f%%%s %s[%d/%d]%s %s%sPublic:%s%d%s %s%sPrivate:%s%d%s %s%sErr:%s%d%s %s%.0f r/s%s %sETA:%s%s%s",
			progressColorBar, bar, progressColorReset,
			progressColorValue, pct, progressColorReset,
			progressColorLabel, scanned, p.total, progressColorReset,
			progressColorPublic, progressColorLabel, progressColorPublic, public, progressColorReset,
			progressColorPrivate, progressColorLabel, progressColorPrivate, private, progressColorReset,
			progressColorError, progressColorLabel, progressColorError, errors, progressColorReset,
			progressColorValue, rps, progressColorReset,
			progressColorLabel, progressColorValue, eta, progressColorReset,
		)
	} else {
		statsLine = fmt.Sprintf(
			"%s %.1f%% [%d/%d] Public:%d Private:%d Err:%d %.0f r/s ETA:%s",
			bar, pct, scanned, p.total, public, private, errors, rps, eta,
		)
	}

	// Add elapsed time
	elapsedStr := formatDuration(elapsed)
	statsLine = fmt.Sprintf("%s [%s]", statsLine, elapsedStr)

	// Clear previous line and write new one
	if p.lastLineLen > 0 {
		fmt.Fprint(p.cfg.Output, "\r")
		fmt.Fprint(p.cfg.Output, strings.Repeat(" ", p.lastLineLen))
		fmt.Fprint(p.cfg.Output, "\r")
	}

	if final {
		fmt.Fprintln(p.cfg.Output, statsLine)
	} else {
		fmt.Fprint(p.cfg.Output, statsLine)
	}

	// Store length for clearing next time (without ANSI codes)
	p.lastLineLen = p.visibleLength(statsLine)
}

func (p *Progress) buildBar(pct float64) string {
	width := p.cfg.BarWidth
	filled := int(pct / 100 * float64(width))
	if filled > width {
		filled = width
	}

	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	return "[" + bar + "]"
}

func (p *Progress) visibleLength(s string) int {
	// Strip ANSI codes and count visible characters
	inEscape := false
	length := 0
	for _, c := range s {
		if c == '\033' {
			inEscape = true
		} else if inEscape && c == 'm' {
			inEscape = false
		} else if !inEscape {
			length++
		}
	}
	return length
}

func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second

	if h > 0 {
		return fmt.Sprintf("%dh%02dm%02ds", h, m, s)
	} else if m > 0 {
		return fmt.Sprintf("%dm%02ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}
