package scanner

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/xeloxa/s3finder/pkg/ratelimit"
)

// ProbeResult represents the outcome of a bucket probe.
type ProbeResult int

const (
	BucketNotFound  ProbeResult = iota // 404 - bucket does not exist
	BucketExists                       // 200 - bucket exists and is publicly readable
	BucketForbidden                    // 403 - bucket exists but access denied
	BucketError                        // Network error or unexpected response
)

func (r ProbeResult) String() string {
	switch r {
	case BucketNotFound:
		return "not_found"
	case BucketExists:
		return "public"
	case BucketForbidden:
		return "private"
	case BucketError:
		return "error"
	default:
		return "unknown"
	}
}

// ProbeResponse contains the result of probing a bucket.
type ProbeResponse struct {
	Bucket     string
	Result     ProbeResult
	StatusCode int
	Error      error
}

// Prober performs HTTP checks on S3 bucket names.
type Prober struct {
	client  *http.Client
	limiter *ratelimit.AdaptiveLimiter
}

// ProberConfig holds configuration for the Prober.
type ProberConfig struct {
	Timeout             time.Duration
	MaxIdleConns        int
	MaxIdleConnsPerHost int
	MaxConnsPerHost     int
	MaxRPS              float64
}

// DefaultProberConfig returns optimized defaults for high-throughput scanning.
func DefaultProberConfig() *ProberConfig {
	return &ProberConfig{
		Timeout:             10 * time.Second,
		MaxIdleConns:        1000,
		MaxIdleConnsPerHost: 100,
		MaxConnsPerHost:     100,
		MaxRPS:              500,
	}
}

// NewProber creates a new Prober with the given configuration.
func NewProber(cfg *ProberConfig) *Prober {
	if cfg == nil {
		cfg = DefaultProberConfig()
	}

	transport := &http.Transport{
		MaxIdleConns:        cfg.MaxIdleConns,
		MaxIdleConnsPerHost: cfg.MaxIdleConnsPerHost,
		MaxConnsPerHost:     cfg.MaxConnsPerHost,
		IdleConnTimeout:     30 * time.Second,
		DisableKeepAlives:   false,
		TLSHandshakeTimeout: 5 * time.Second,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: false,
		},
	}

	client := &http.Client{
		Timeout:   cfg.Timeout,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	return &Prober{
		client:  client,
		limiter: ratelimit.New(cfg.MaxRPS),
	}
}

// Check probes a bucket name and returns the result.
func (p *Prober) Check(ctx context.Context, bucket string) *ProbeResponse {
	resp := &ProbeResponse{Bucket: bucket}

	// Wait for rate limiter
	if err := p.limiter.Wait(ctx); err != nil {
		resp.Result = BucketError
		resp.Error = err
		return resp
	}

	url := fmt.Sprintf("https://%s.s3.amazonaws.com", bucket)

	req, err := http.NewRequestWithContext(ctx, "HEAD", url, nil)
	if err != nil {
		resp.Result = BucketError
		resp.Error = err
		return resp
	}

	httpResp, err := p.client.Do(req)
	if err != nil {
		resp.Result = BucketError
		resp.Error = err
		p.limiter.RecordResponse(0)
		return resp
	}
	defer httpResp.Body.Close()

	resp.StatusCode = httpResp.StatusCode
	p.limiter.RecordResponse(httpResp.StatusCode)

	switch httpResp.StatusCode {
	case 200:
		resp.Result = BucketExists
	case 403:
		resp.Result = BucketForbidden
	case 404:
		resp.Result = BucketNotFound
	case 301, 307:
		// Redirect typically means bucket exists in different region
		resp.Result = BucketForbidden
	default:
		resp.Result = BucketError
		resp.Error = fmt.Errorf("unexpected status code: %d", httpResp.StatusCode)
	}

	return resp
}

// CurrentRPS returns the current rate limit.
func (p *Prober) CurrentRPS() float64 {
	return p.limiter.CurrentRPS()
}
