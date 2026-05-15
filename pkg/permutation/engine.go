package permutation

import (
	"regexp"
	"strings"
	"unicode"
)

var validBucketName = regexp.MustCompile(`^[a-z0-9][a-z0-9.-]{1,61}[a-z0-9]$`)

// tokenize splits a seed into lowercase tokens by splitting on spaces, hyphens,
// underscores, dots, and camelCase boundaries.
func tokenize(seed string) []string {
	// Insert space before uppercase letters that follow lowercase (camelCase split)
	var spaced strings.Builder
	runes := []rune(seed)
	for i, r := range runes {
		if i > 0 && unicode.IsUpper(r) && unicode.IsLower(runes[i-1]) {
			spaced.WriteRune(' ')
		}
		spaced.WriteRune(r)
	}

	// Split on non-alphanumeric characters
	raw := regexp.MustCompile(`[^a-zA-Z0-9]+`).Split(spaced.String(), -1)

	var tokens []string
	for _, t := range raw {
		t = strings.ToLower(strings.TrimSpace(t))
		if t != "" {
			tokens = append(tokens, t)
		}
	}
	return tokens
}

// ExpandSeed takes a raw seed (e.g. "Acme Corp" or "my-company") and returns a deduplicated
// list of normalized variants to run permutations against.
//
// For "Acme Corp" it produces:
//
//	acme-corp, acmecorp, acme, corp
func ExpandSeed(seed string) []string {
	tokens := tokenize(seed)
	if len(tokens) == 0 {
		return nil
	}

	seen := make(map[string]struct{})
	var variants []string

	add := func(s string) {
		s = strings.ToLower(s)
		if s == "" {
			return
		}
		if _, ok := seen[s]; !ok {
			seen[s] = struct{}{}
			variants = append(variants, s)
		}
	}

	// Full joined forms
	add(strings.Join(tokens, "-"))
	add(strings.Join(tokens, ""))
	add(strings.Join(tokens, "."))

	// Individual tokens
	for _, t := range tokens {
		add(t)
	}

	// All separator combinations between tokens (e.g. acme-corp, acmecorp)
	if len(tokens) > 1 {
		generateCombinations(tokens, 0, tokens[0], []string{"-", ""}, add)
	}

	// Consecutive sub-sequences (length 2..len)
	for length := 2; length <= len(tokens); length++ {
		for start := 0; start+length <= len(tokens); start++ {
			sub := tokens[start : start+length]
			add(strings.Join(sub, "-"))
			add(strings.Join(sub, ""))
		}
	}

	return variants
}

// Engine generates bucket name permutations from seed keywords.
type Engine struct {
	Suffixes   []string
	Prefixes   []string
	Separators []string
	Years      []string
	Regions    []string
}

// Default returns an Engine with common AWS naming patterns.
func Default() *Engine {
	return &Engine{
		Suffixes: []string{
			"", "-dev", "-prod", "-staging", "-backup", "-backups",
			"-logs", "-assets", "-internal", "-public", "-private",
			"-data", "-files", "-media", "-static", "-cdn",
			"-api", "-web", "-app", "-test", "-temp",
			"-archive", "-old", "-new", "-v2", "-beta",
		},
		Prefixes: []string{
			"", "dev-", "prod-", "staging-", "backup-", "test-",
			"internal-", "public-", "private-", "temp-", "old-",
		},
		Separators: []string{"-", "."},
		Years: []string{
			"", "-2022", "-2023", "-2024", "-2025",
			"-22", "-23", "-24", "-25",
		},
		Regions: []string{
			"", "-us-east-1", "-us-east-2", "-us-west-1", "-us-west-2",
			"-eu-west-1", "-eu-west-2", "-eu-central-1",
			"-ap-south-1", "-ap-northeast-1", "-ap-southeast-1",
		},
	}
}

// Generate creates all permutations for the given seed keyword.
func (e *Engine) Generate(seed string) []string {
	seed = strings.ToLower(strings.TrimSpace(seed))
	if seed == "" {
		return nil
	}

	seen := make(map[string]struct{})
	var results []string

	add := func(name string) {
		if _, ok := seen[name]; !ok && IsValidBucketName(name) {
			seen[name] = struct{}{}
			results = append(results, name)
		}
	}

	// Base seed
	add(seed)

	// Prefix + seed
	for _, prefix := range e.Prefixes {
		add(prefix + seed)
	}

	// Seed + suffix
	for _, suffix := range e.Suffixes {
		add(seed + suffix)
	}

	// Prefix + seed + suffix
	for _, prefix := range e.Prefixes {
		for _, suffix := range e.Suffixes {
			add(prefix + seed + suffix)
		}
	}

	// Seed + year
	for _, year := range e.Years {
		add(seed + year)
	}

	// Seed + suffix + year
	for _, suffix := range e.Suffixes {
		for _, year := range e.Years {
			add(seed + suffix + year)
		}
	}

	// Seed + region
	for _, region := range e.Regions {
		add(seed + region)
	}

	// Seed + suffix + region
	for _, suffix := range e.Suffixes {
		for _, region := range e.Regions {
			add(seed + suffix + region)
		}
	}

	// Separator variations (replace - with .)
	for _, sep := range e.Separators {
		if sep != "-" {
			variant := strings.ReplaceAll(seed, "-", sep)
			if variant != seed {
				add(variant)
				for _, suffix := range e.Suffixes {
					add(variant + suffix)
				}
			}
		}
	}

	return results
}

// GenerateFromWordlist applies permutations to each word in the list.
func (e *Engine) GenerateFromWordlist(words []string, seed string) []string {
	seed = strings.ToLower(strings.TrimSpace(seed))
	seen := make(map[string]struct{})
	var results []string

	add := func(name string) {
		if _, ok := seen[name]; !ok && IsValidBucketName(name) {
			seen[name] = struct{}{}
			results = append(results, name)
		}
	}

	for _, word := range words {
		word = strings.ToLower(strings.TrimSpace(word))
		if word == "" {
			continue
		}

		// Direct word
		add(word)

		// Seed + word combinations
		if seed != "" {
			add(seed + "-" + word)
			add(word + "-" + seed)
			add(seed + "." + word)
			add(word + "." + seed)
		}

		// Apply suffixes to word
		for _, suffix := range e.Suffixes {
			add(word + suffix)
			if seed != "" {
				add(seed + "-" + word + suffix)
			}
		}

		// Apply years
		for _, year := range e.Years {
			add(word + year)
			if seed != "" {
				add(seed + "-" + word + year)
			}
		}
	}

	return results
}

// generateCombinations recursively builds all separator-combination variants of tokens.
// e.g. ["acme","corp","dev"] → acme-corp-dev, acme-corpdev, acmecorp-dev, acmecorpdev
func generateCombinations(tokens []string, idx int, current string, seps []string, add func(string)) {
	if idx == len(tokens)-1 {
		add(current)
		return
	}
	for _, sep := range seps {
		generateCombinations(tokens, idx+1, current+sep+tokens[idx+1], seps, add)
	}
}
func IsValidBucketName(name string) bool {
	if len(name) < 3 || len(name) > 63 {
		return false
	}

	// Must not look like an IP address
	if isIPAddress(name) {
		return false
	}

	// No consecutive periods
	if strings.Contains(name, "..") {
		return false
	}

	// Must match pattern: lowercase alphanumeric, hyphens, dots
	return validBucketName.MatchString(name)
}

func isIPAddress(s string) bool {
	parts := strings.Split(s, ".")
	if len(parts) != 4 {
		return false
	}
	for _, part := range parts {
		if len(part) == 0 || len(part) > 3 {
			return false
		}
		for _, c := range part {
			if c < '0' || c > '9' {
				return false
			}
		}
	}
	return true
}
