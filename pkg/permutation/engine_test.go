package permutation

import (
	"strings"
	"testing"
)

func TestIsValidBucketName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		// Valid bucket names
		{"simple lowercase", "mybucket", true},
		{"with hyphens", "my-bucket-name", true},
		{"with dots", "my.bucket.name", true},
		{"with numbers", "bucket123", true},
		{"mixed valid chars", "my-bucket.123", true},
		{"minimum length 3", "abc", true},
		{"maximum length 63", strings.Repeat("a", 63), true},
		{"starts with number", "123bucket", true},
		{"ends with number", "bucket123", true},

		// Invalid: length constraints
		{"too short 1 char", "a", false},
		{"too short 2 chars", "ab", false},
		{"too long 64 chars", strings.Repeat("a", 64), false},
		{"too long 100 chars", strings.Repeat("a", 100), false},
		{"empty string", "", false},

		// Invalid: character constraints
		{"uppercase letters", "MyBucket", false},
		{"mixed case", "myBucket", false},
		{"underscore", "my_bucket", false},
		{"space in name", "my bucket", false},
		{"special char @", "my@bucket", false},
		{"special char !", "mybucket!", false},
		{"special char #", "my#bucket", false},
		{"special char $", "my$bucket", false},
		{"unicode chars", "mybücket", false},

		// Invalid: structural constraints
		{"consecutive dots", "my..bucket", false},
		{"starts with hyphen", "-mybucket", false},
		{"ends with hyphen", "mybucket-", false},
		{"starts with dot", ".mybucket", false},
		{"ends with dot", "mybucket.", false},

		// Invalid: IP address format
		{"ip address format", "192.168.1.1", false},
		{"ip-like with zeros", "10.0.0.1", false},
		{"ip-like edge case", "1.2.3.4", false},

		// Valid: IP-like but not IP
		{"ip-like with letters", "192.168.1.a", true},
		{"ip-like too many octets", "1.2.3.4.5", true},
		{"ip-like with long numbers", "1234.5678.9012.3456", true},

		// Edge cases
		{"single dot in middle", "my.bucket", true},
		{"hyphen and dot combo", "my-bucket.test", true},
		{"all numbers", "1234567890", true},
		{"number-hyphen-number", "123-456-789", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidBucketName(tt.input)
			if result != tt.expected {
				t.Errorf("IsValidBucketName(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsValidBucketName_BoundaryLengths(t *testing.T) {
	// Test exact boundary conditions
	tests := []struct {
		length   int
		expected bool
	}{
		{1, false},
		{2, false},
		{3, true},  // minimum valid
		{4, true},
		{62, true},
		{63, true}, // maximum valid
		{64, false},
		{65, false},
	}

	for _, tt := range tests {
		name := strings.Repeat("a", tt.length)
		t.Run(string(rune('0'+tt.length)), func(t *testing.T) {
			result := IsValidBucketName(name)
			if result != tt.expected {
				t.Errorf("IsValidBucketName(len=%d) = %v, want %v", tt.length, result, tt.expected)
			}
		})
	}
}

func TestGenerate(t *testing.T) {
	engine := Default()

	t.Run("generates permutations for valid seed", func(t *testing.T) {
		names := engine.Generate("acme")

		if len(names) == 0 {
			t.Fatal("expected at least one permutation, got 0")
		}

		// Should generate many variations
		if len(names) < 100 {
			t.Errorf("expected at least 100 permutations, got %d", len(names))
		}
	})

	t.Run("returns empty for empty seed", func(t *testing.T) {
		names := engine.Generate("")

		if len(names) != 0 {
			t.Errorf("expected 0 permutations for empty seed, got %d", len(names))
		}
	})

	t.Run("returns empty for whitespace seed", func(t *testing.T) {
		names := engine.Generate("   ")

		if len(names) != 0 {
			t.Errorf("expected 0 permutations for whitespace seed, got %d", len(names))
		}
	})

	t.Run("lowercases input", func(t *testing.T) {
		names := engine.Generate("ACME")

		for _, name := range names {
			if strings.ToLower(name) != name {
				t.Errorf("expected lowercase name, got %q", name)
			}
		}
	})

	t.Run("all generated names are valid", func(t *testing.T) {
		names := engine.Generate("testcompany")

		for _, name := range names {
			if !IsValidBucketName(name) {
				t.Errorf("generated invalid bucket name: %q", name)
			}
		}
	})

	t.Run("no duplicate names", func(t *testing.T) {
		names := engine.Generate("acme")
		seen := make(map[string]bool)

		for _, name := range names {
			if seen[name] {
				t.Errorf("duplicate name generated: %q", name)
			}
			seen[name] = true
		}
	})

	t.Run("includes expected patterns", func(t *testing.T) {
		names := engine.Generate("acme")
		nameSet := make(map[string]bool)
		for _, n := range names {
			nameSet[n] = true
		}

		expected := []string{
			"acme",           // base
			"acme-dev",       // suffix
			"acme-prod",      // suffix
			"dev-acme",       // prefix
			"acme-backup",    // suffix
			"acme-2024",      // year
			"acme-us-east-1", // region
		}

		for _, exp := range expected {
			if !nameSet[exp] {
				t.Errorf("expected pattern %q not found in generated names", exp)
			}
		}
	})
}

func TestGenerate_WithHyphenatedSeed(t *testing.T) {
	engine := Default()
	names := engine.Generate("acme-corp")

	t.Run("handles hyphenated seeds", func(t *testing.T) {
		if len(names) == 0 {
			t.Fatal("expected permutations for hyphenated seed")
		}

		// Should include dot-separated variant
		hasDotVariant := false
		for _, name := range names {
			if strings.Contains(name, "acme.corp") {
				hasDotVariant = true
				break
			}
		}

		if !hasDotVariant {
			t.Error("expected dot-separated variant for hyphenated seed")
		}
	})

	t.Run("all names are valid", func(t *testing.T) {
		for _, name := range names {
			if !IsValidBucketName(name) {
				t.Errorf("generated invalid bucket name: %q", name)
			}
		}
	})
}

func TestGenerateFromWordlist(t *testing.T) {
	engine := Default()

	t.Run("combines wordlist with seed", func(t *testing.T) {
		words := []string{"backup", "logs", "data"}
		names := engine.GenerateFromWordlist(words, "acme")

		if len(names) == 0 {
			t.Fatal("expected permutations, got 0")
		}

		nameSet := make(map[string]bool)
		for _, n := range names {
			nameSet[n] = true
		}

		// Should include seed-word combinations
		expected := []string{
			"backup",
			"acme-backup",
			"backup-acme",
			"logs",
			"acme-logs",
			"data",
			"acme-data",
		}

		for _, exp := range expected {
			if !nameSet[exp] {
				t.Errorf("expected %q in generated names", exp)
			}
		}
	})

	t.Run("works with empty seed", func(t *testing.T) {
		words := []string{"backup", "logs"}
		names := engine.GenerateFromWordlist(words, "")

		if len(names) == 0 {
			t.Fatal("expected permutations even with empty seed")
		}

		// Should still process words with suffixes
		hasBackup := false
		for _, name := range names {
			if strings.HasPrefix(name, "backup") {
				hasBackup = true
				break
			}
		}

		if !hasBackup {
			t.Error("expected wordlist words to be processed")
		}
	})

	t.Run("skips empty words", func(t *testing.T) {
		words := []string{"", "  ", "backup", ""}
		names := engine.GenerateFromWordlist(words, "acme")

		for _, name := range names {
			if name == "" || strings.TrimSpace(name) == "" {
				t.Error("generated empty name from wordlist")
			}
		}
	})

	t.Run("all names are valid", func(t *testing.T) {
		words := []string{"backup", "logs", "internal-data", "test.files"}
		names := engine.GenerateFromWordlist(words, "acme")

		for _, name := range names {
			if !IsValidBucketName(name) {
				t.Errorf("generated invalid bucket name: %q", name)
			}
		}
	})

	t.Run("no duplicates", func(t *testing.T) {
		words := []string{"backup", "backup", "logs"} // intentional duplicate
		names := engine.GenerateFromWordlist(words, "acme")

		seen := make(map[string]bool)
		for _, name := range names {
			if seen[name] {
				t.Errorf("duplicate name: %q", name)
			}
			seen[name] = true
		}
	})
}

func TestDefaultEngine(t *testing.T) {
	engine := Default()

	t.Run("has suffixes", func(t *testing.T) {
		if len(engine.Suffixes) == 0 {
			t.Error("default engine should have suffixes")
		}
	})

	t.Run("has prefixes", func(t *testing.T) {
		if len(engine.Prefixes) == 0 {
			t.Error("default engine should have prefixes")
		}
	})

	t.Run("has years", func(t *testing.T) {
		if len(engine.Years) == 0 {
			t.Error("default engine should have years")
		}
	})

	t.Run("has regions", func(t *testing.T) {
		if len(engine.Regions) == 0 {
			t.Error("default engine should have regions")
		}
	})

	t.Run("includes common suffixes", func(t *testing.T) {
		suffixSet := make(map[string]bool)
		for _, s := range engine.Suffixes {
			suffixSet[s] = true
		}

		required := []string{"-dev", "-prod", "-backup", "-logs"}
		for _, r := range required {
			if !suffixSet[r] {
				t.Errorf("missing common suffix: %s", r)
			}
		}
	})
}

func BenchmarkGenerate(b *testing.B) {
	engine := Default()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.Generate("benchmark-company")
	}
}

func BenchmarkIsValidBucketName(b *testing.B) {
	names := []string{
		"valid-bucket-name",
		"another.valid.bucket",
		"Invalid_Name",
		"toolongbucketnamethatexceedsthemaximumlengthallowedbyawss3",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, name := range names {
			IsValidBucketName(name)
		}
	}
}
