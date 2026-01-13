package recon

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// CTResult represents a certificate transparency log entry.
type CTResult struct {
	IssuerCAID     int    `json:"issuer_ca_id"`
	IssuerName     string `json:"issuer_name"`
	CommonName     string `json:"common_name"`
	NameValue      string `json:"name_value"`
	ID             int64  `json:"id"`
	EntryTimestamp string `json:"entry_timestamp"`
	NotBefore      string `json:"not_before"`
	NotAfter       string `json:"not_after"`
	SerialNumber   string `json:"serial_number"`
}

// CTClient queries Certificate Transparency logs.
type CTClient struct {
	httpClient *http.Client
	maxResults int
}

// NewCTClient creates a new CT logs client.
func NewCTClient(timeout time.Duration, maxResults int) *CTClient {
	return &CTClient{
		httpClient: &http.Client{Timeout: timeout},
		maxResults: maxResults,
	}
}

// FetchSubdomains queries crt.sh for subdomains of the given domain.
func (c *CTClient) FetchSubdomains(ctx context.Context, domain string) ([]string, error) {
	domain = cleanDomain(domain)
	if domain == "" {
		return nil, fmt.Errorf("invalid domain")
	}

	apiURL := fmt.Sprintf("https://crt.sh/?q=%%25.%s&output=json", url.QueryEscape(domain))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "s3finder/1.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("crt.sh request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("crt.sh returned status %d", resp.StatusCode)
	}

	var results []CTResult
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("failed to parse crt.sh response: %w", err)
	}

	return c.extractSubdomains(results, domain), nil
}

// extractSubdomains deduplicates and filters subdomains from CT results.
func (c *CTClient) extractSubdomains(results []CTResult, baseDomain string) []string {
	seen := make(map[string]struct{})
	var subdomains []string

	for _, r := range results {
		names := strings.Split(r.NameValue, "\n")
		for _, name := range names {
			name = strings.TrimSpace(strings.ToLower(name))
			name = strings.TrimPrefix(name, "*.")

			if name == "" || name == baseDomain {
				continue
			}
			if !strings.HasSuffix(name, "."+baseDomain) && name != baseDomain {
				continue
			}
			if _, ok := seen[name]; ok {
				continue
			}

			seen[name] = struct{}{}
			subdomains = append(subdomains, name)

			if len(subdomains) >= c.maxResults {
				return subdomains
			}
		}
	}

	return subdomains
}

// SubdomainsToSeeds converts subdomains to potential S3 bucket seed names.
func SubdomainsToSeeds(subdomains []string, baseDomain string) []string {
	seen := make(map[string]struct{})
	var seeds []string

	baseDomain = cleanDomain(baseDomain)
	baseParts := strings.Split(baseDomain, ".")
	basePrefix := baseParts[0]

	for _, sub := range subdomains {
		sub = strings.TrimSuffix(sub, "."+baseDomain)
		if sub == "" || sub == baseDomain {
			continue
		}

		candidates := generateSeedCandidates(sub, basePrefix)
		for _, seed := range candidates {
			if _, ok := seen[seed]; !ok && isValidBucketSeed(seed) {
				seen[seed] = struct{}{}
				seeds = append(seeds, seed)
			}
		}
	}

	return seeds
}

// generateSeedCandidates creates potential bucket names from a subdomain part.
func generateSeedCandidates(subPart, basePrefix string) []string {
	var candidates []string

	subPart = strings.ToLower(subPart)
	subPart = strings.ReplaceAll(subPart, ".", "-")

	candidates = append(candidates, subPart)
	candidates = append(candidates, subPart+"-"+basePrefix)
	candidates = append(candidates, basePrefix+"-"+subPart)

	return candidates
}

// cleanDomain removes protocol and trailing slashes from domain.
func cleanDomain(domain string) string {
	domain = strings.ToLower(domain)
	domain = strings.TrimPrefix(domain, "http://")
	domain = strings.TrimPrefix(domain, "https://")
	domain = strings.TrimSuffix(domain, "/")
	domain = strings.TrimPrefix(domain, "www.")
	return domain
}

// isValidBucketSeed checks if a string could be a valid S3 bucket name component.
func isValidBucketSeed(s string) bool {
	if len(s) < 2 || len(s) > 63 {
		return false
	}
	matched, _ := regexp.MatchString(`^[a-z0-9][a-z0-9\-]*[a-z0-9]$`, s)
	return matched
}
