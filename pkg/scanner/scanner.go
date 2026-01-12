package scanner

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// ScanResult contains the complete result of scanning a bucket.
type ScanResult struct {
	Bucket    string         `json:"bucket"`
	Probe     ProbeResult    `json:"probe_result"`
	Inspect   *InspectResult `json:"inspect,omitempty"`
	Timestamp time.Time      `json:"timestamp"`
}

// Stats tracks scanning statistics.
type Stats struct {
	Total     int64
	Scanned   int64
	Found     int64
	Public    int64
	Private   int64
	Errors    int64
	StartTime time.Time
}

// Scanner orchestrates the bucket enumeration process.
type Scanner struct {
	prober       *Prober
	inspector    *Inspector
	workers      int
	deepInspect  bool
	stats        Stats
	resultsChan  chan *ScanResult
	mu           sync.RWMutex
}

// Config holds scanner configuration.
type Config struct {
	Workers     int
	MaxRPS      float64
	Timeout     time.Duration
	DeepInspect bool
}

// DefaultConfig returns sensible default configuration.
func DefaultConfig() *Config {
	return &Config{
		Workers:     100,
		MaxRPS:      500,
		Timeout:     10 * time.Second,
		DeepInspect: true,
	}
}

// New creates a new Scanner with the given configuration.
func New(cfg *Config) *Scanner {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	proberCfg := &ProberConfig{
		Timeout:             cfg.Timeout,
		MaxIdleConns:        cfg.Workers * 10,
		MaxIdleConnsPerHost: cfg.Workers,
		MaxConnsPerHost:     cfg.Workers,
		MaxRPS:              cfg.MaxRPS,
	}

	return &Scanner{
		prober:      NewProber(proberCfg),
		inspector:   NewInspector(30 * time.Second),
		workers:     cfg.Workers,
		deepInspect: cfg.DeepInspect,
		resultsChan: make(chan *ScanResult, 1000),
	}
}

// Scan starts scanning the provided bucket names.
// Returns a channel that receives results as they're found.
func (s *Scanner) Scan(ctx context.Context, names []string) <-chan *ScanResult {
	s.stats = Stats{
		Total:     int64(len(names)),
		StartTime: time.Now(),
	}

	namesChan := make(chan string, 1000)

	// Producer: feed names into channel
	go func() {
		defer close(namesChan)
		for _, name := range names {
			select {
			case <-ctx.Done():
				return
			case namesChan <- name:
			}
		}
	}()

	// Consumer: worker pool
	var wg sync.WaitGroup
	for i := 0; i < s.workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.worker(ctx, namesChan)
		}()
	}

	// Close results channel when all workers are done
	go func() {
		wg.Wait()
		close(s.resultsChan)
	}()

	return s.resultsChan
}

// worker processes bucket names from the channel.
func (s *Scanner) worker(ctx context.Context, names <-chan string) {
	for {
		select {
		case <-ctx.Done():
			return
		case name, ok := <-names:
			if !ok {
				return
			}
			s.processBucket(ctx, name)
		}
	}
}

// processBucket probes a single bucket and optionally inspects it.
func (s *Scanner) processBucket(ctx context.Context, bucket string) {
	atomic.AddInt64(&s.stats.Scanned, 1)

	probe := s.prober.Check(ctx, bucket)

	result := &ScanResult{
		Bucket:    bucket,
		Probe:     probe.Result,
		Timestamp: time.Now(),
	}

	switch probe.Result {
	case BucketNotFound:
		// Don't send not-found results
		return
	case BucketExists:
		atomic.AddInt64(&s.stats.Found, 1)
		atomic.AddInt64(&s.stats.Public, 1)
		if s.deepInspect {
			result.Inspect = s.inspector.Inspect(ctx, bucket)
		}
	case BucketForbidden:
		atomic.AddInt64(&s.stats.Found, 1)
		atomic.AddInt64(&s.stats.Private, 1)
		if s.deepInspect {
			result.Inspect = s.inspector.Inspect(ctx, bucket)
		}
	case BucketError:
		atomic.AddInt64(&s.stats.Errors, 1)
	}

	select {
	case <-ctx.Done():
	case s.resultsChan <- result:
	}
}

// Results returns the results channel.
func (s *Scanner) Results() <-chan *ScanResult {
	return s.resultsChan
}

// Stats returns current scanning statistics.
func (s *Scanner) Stats() Stats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return Stats{
		Total:     s.stats.Total,
		Scanned:   atomic.LoadInt64(&s.stats.Scanned),
		Found:     atomic.LoadInt64(&s.stats.Found),
		Public:    atomic.LoadInt64(&s.stats.Public),
		Private:   atomic.LoadInt64(&s.stats.Private),
		Errors:    atomic.LoadInt64(&s.stats.Errors),
		StartTime: s.stats.StartTime,
	}
}

// CurrentRPS returns the current rate limit.
func (s *Scanner) CurrentRPS() float64 {
	return s.prober.CurrentRPS()
}
