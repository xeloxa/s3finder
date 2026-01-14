package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/xeloxa/s3finder/internal/config"
	"github.com/xeloxa/s3finder/pkg/ai"
	"github.com/xeloxa/s3finder/pkg/output"
	"github.com/xeloxa/s3finder/pkg/permutation"
	"github.com/xeloxa/s3finder/pkg/recon"
	"github.com/xeloxa/s3finder/pkg/scanner"
)

var (
	version   = "dev"
	buildTime = "unknown"
	cfg       = config.Default()
	outputMu  sync.Mutex // Global mutex for synchronized output
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "s3finder",
		Short: "AI-powered S3 bucket enumeration tool",
		Long: `s3finder discovers AWS S3 buckets using wordlists and AI-driven name generation.

Examples:
  s3finder -s acme                    # Scan with permutations of "acme"
  s3finder -s acme -w wordlist.txt    # Scan with wordlist + permutations
  s3finder -s acme --ai               # Enable AI name generation
  s3finder -s acme -t 200 --rps 1000  # High-speed scan`,
		RunE: run,
	}

	// ... flags ...
	rootCmd.Flags().IntVarP(&cfg.Workers, "threads", "t", cfg.Workers, "Number of concurrent workers")
	rootCmd.Flags().Float64Var(&cfg.MaxRPS, "rps", cfg.MaxRPS, "Maximum requests per second")
	rootCmd.Flags().IntVar(&cfg.Timeout, "timeout", cfg.Timeout, "Request timeout in seconds")
	rootCmd.Flags().BoolVar(&cfg.DeepInspect, "deep", cfg.DeepInspect, "Perform deep inspection on found buckets")

	// Input flags
	rootCmd.Flags().StringVarP(&cfg.Seed, "seed", "s", "", "Target keyword for bucket name generation")
	rootCmd.Flags().StringVarP(&cfg.Wordlist, "wordlist", "w", "", "Path to wordlist file")
	rootCmd.Flags().StringVarP(&cfg.Domain, "domain", "d", "", "Target domain for CT log subdomain discovery")
	rootCmd.Flags().IntVar(&cfg.CTLimit, "ct-limit", cfg.CTLimit, "Maximum subdomains to fetch from CT logs")

	// AI flags
	rootCmd.Flags().BoolVar(&cfg.AIEnabled, "ai", cfg.AIEnabled, "Enable AI-powered name generation")
	rootCmd.Flags().StringVar(&cfg.AIProvider, "ai-provider", cfg.AIProvider, "AI provider (openai, ollama, anthropic, gemini)")
	rootCmd.Flags().StringVar(&cfg.AIModel, "ai-model", cfg.AIModel, "AI model name")
	rootCmd.Flags().StringVar(&cfg.AIKey, "ai-key", "", "AI provider API key (or use env: OPENAI_API_KEY, ANTHROPIC_API_KEY, GEMINI_API_KEY)")
	rootCmd.Flags().StringVar(&cfg.AIBaseURL, "ai-url", "", "AI provider base URL (for custom endpoints or proxies)")
	rootCmd.Flags().IntVar(&cfg.AICount, "ai-count", cfg.AICount, "Number of AI-generated names")

	// Output flags
	rootCmd.Flags().StringVarP(&cfg.OutputFile, "output", "o", cfg.OutputFile, "Output file path")
	rootCmd.Flags().StringVarP(&cfg.OutputFormat, "format", "f", cfg.OutputFormat, "Output format (json, txt)")
	rootCmd.Flags().BoolVar(&cfg.NoColor, "no-color", cfg.NoColor, "Disable colored output")
	rootCmd.Flags().BoolVarP(&cfg.Verbose, "verbose", "v", cfg.Verbose, "Verbose output")

	// Version command
	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("s3finder version %s (built %s)\n", version, buildTime)
		},
	})

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) error {
	// Validate input sources
	if cfg.Seed == "" && cfg.Wordlist == "" && cfg.Domain == "" && !cfg.AIEnabled {
		return fmt.Errorf("at least one input source is required: --seed, --wordlist, --domain, or --ai")
	}

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\n\nInterrupted. Shutting down...")
		cancel()
	}()

	// Resolve API key from environment if not provided
	if cfg.AIEnabled && cfg.AIKey == "" {
		switch cfg.AIProvider {
		case "openai":
			cfg.AIKey = os.Getenv("OPENAI_API_KEY")
		case "anthropic":
			cfg.AIKey = os.Getenv("ANTHROPIC_API_KEY")
		case "gemini":
			cfg.AIKey = os.Getenv("GEMINI_API_KEY")
		}
	}

	// Banner (Static)
	printBanner()

	// Generate bucket names
	names, err := generateNames(ctx)
	if err != nil {
		return fmt.Errorf("failed to generate names: %w", err)
	}

	if len(names) == 0 {
		return fmt.Errorf("no bucket names generated")
	}

	fmt.Printf("Generated %d unique bucket names to scan\n\n", len(names))

	// Setup progress bar
	progress := output.NewProgress(&output.ProgressConfig{
		Output:      os.Stderr,
		Total:       int64(len(names)),
		RefreshRate: 100 * time.Millisecond,
		ShowRPS:     true,
		UseColors:   !cfg.NoColor,
		BarWidth:    25,
		ExternalMu:  &outputMu,
	})

	// Setup output writers (pass progress for coordinated output)
	realtimeWriter := output.NewRealtime(&output.RealtimeConfig{
		Output:    os.Stdout,
		UseColors: !cfg.NoColor,
		UseLinks:  !cfg.NoColor, // Enable clickable links when colors are enabled
		Verbose:   cfg.Verbose,
		Progress:  progress, // Coordinate output with progress bar
	})

	reportWriter, err := output.NewReport(&output.ReportConfig{
		FilePath:  cfg.OutputFile,
		Format:    cfg.OutputFormat,
		StartTime: time.Now(),
	})
	if err != nil {
		return fmt.Errorf("failed to create report writer: %w", err)
	}
	defer reportWriter.Close()

	multiWriter := output.NewMultiWriter(realtimeWriter, reportWriter)

	// Create scanner
	s := scanner.New(&scanner.Config{
		Workers:     cfg.Workers,
		MaxRPS:      cfg.MaxRPS,
		Timeout:     time.Duration(cfg.Timeout) * time.Second,
		DeepInspect: cfg.DeepInspect,
	})

	// Start scan
	startTime := time.Now()
	results := s.Scan(ctx, names)

	// Start progress display with stats provider
	go func() {
		ticker := time.NewTicker(50 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				stats := s.Stats()
				progress.Update(stats.Scanned, stats.Found, stats.Public, stats.Private, stats.Errors, s.CurrentRPS())
			}
		}
	}()
	progress.Start()

	// Process results
	for result := range results {
		if err := multiWriter.WriteResult(result); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing result: %v\n", err)
		}
	}

	// Stop progress display
	progress.Stop()

	// Print summary
	stats := s.Stats()
	duration := time.Since(startTime).Round(time.Second)

	fmt.Printf("\n%s\n", "────────────────────────────────────────")
	fmt.Printf("Scan completed in %s\n", duration)
	fmt.Printf("Scanned: %d | Found: %d | Public: %d | Private: %d | Errors: %d | Not Found: %d\n",
		stats.Scanned, stats.Found, stats.Public, stats.Private, stats.Errors, stats.NotFound)
	fmt.Printf("Results saved to: %s\n", cfg.OutputFile)

	return nil
}

func generateNames(ctx context.Context) ([]string, error) {
	seen := make(map[string]struct{})
	var allNames []string
	var contextWords []string

	add := func(names []string) {
		for _, name := range names {
			if _, ok := seen[name]; !ok {
				seen[name] = struct{}{}
				allNames = append(allNames, name)
			}
		}
	}

	engine := permutation.Default()

	// 1. CT Log subdomain discovery (if domain provided)
	if cfg.Domain != "" {
		fmt.Printf("Fetching subdomains from CT logs for %s...\n", cfg.Domain)
		ctClient := recon.NewCTClient(30*time.Second, cfg.CTLimit)
		subdomains, err := ctClient.FetchSubdomains(ctx, cfg.Domain)
		if err != nil {
			fmt.Printf("Warning: CT log fetch failed: %v\n", err)
		} else {
			add(subdomains)
			// Extract words from subdomains and add them as seeds for permutations
			wordMap := make(map[string]struct{})
			for _, sub := range subdomains {
				// Remove the base domain if present to focus on subparts
				cleanSub := strings.TrimSuffix(sub, "."+cfg.Domain)
				// Split by dots and dashes
				parts := strings.FieldsFunc(cleanSub, func(r rune) bool {
					return r == '.' || r == '-'
				})
				for _, part := range parts {
					if len(part) > 2 { // Ignore very short parts like 'm', 'v1'
						wordMap[part] = struct{}{}
					}
				}
			}

			if len(wordMap) > 0 {
				fmt.Printf("Extracted %d unique words from CT logs for deeper scanning\n", len(wordMap))
				for word := range wordMap {
					contextWords = append(contextWords, word)
					// Add permutations of each extracted word
					add(engine.Generate(word))
				}
			}
			fmt.Printf("CT logs processing completed\n")
		}
	}

	// 2. Permutation engine on seed
	if cfg.Seed != "" {
		permNames := engine.Generate(cfg.Seed)
		add(permNames)
		fmt.Printf("Permutation engine generated %d names from seed: %s\n", len(permNames), cfg.Seed)
	}

	// 3. Wordlist (Raw)
	if cfg.Wordlist != "" {
		words, err := config.LoadWordlist(cfg.Wordlist)
		if err != nil {
			return nil, fmt.Errorf("failed to load wordlist: %w", err)
		}
		add(words)
		fmt.Printf("Wordlist loaded %d names\n", len(words))
	}

	// 4. AI generation
	if cfg.AIEnabled {
		if cfg.Seed == "" && len(contextWords) == 0 {
			fmt.Println("Warning: AI generation requires a seed keyword or discovered context. Skipping AI generation.")
		} else {
			fmt.Printf("Generating AI suggestions using %s (with context-aware discovery)...\n", cfg.AIProvider)

			aiCfg := &ai.Config{
				Provider:    cfg.AIProvider,
				Model:       cfg.AIModel,
				APIKey:      cfg.AIKey,
				BaseURL:     cfg.AIBaseURL,
				Temperature: 0.7,
			}

			generator, err := ai.NewGenerator(aiCfg)
			if err != nil {
				fmt.Printf("Warning: AI generation failed: %v\n", err)
			} else {
				// Use both seed and discovered context words
				aiNames, err := generator.Generate(ctx, cfg.Seed, contextWords, cfg.AICount)
				if err != nil {
					fmt.Printf("Warning: AI generation failed: %v\n", err)
				} else {
					add(aiNames)
					fmt.Printf("AI (%s) discovered patterns and generated %d names\n", generator.Name(), len(aiNames))
				}
			}
		}
	}

	return allNames, nil
}

func printBanner() {
	if cfg.NoColor {
		banner := `
     ____  _____  __ _           _
    / ___|___ / / _(_)_ __   __| | ___ _ __
    \___ \ |_ \| |_| | '_ \ / _` + "`" + ` |/ _ \ '__|
     ___) |__) |  _| | | | | (_| |  __/ |
    |____/____/|_| |_|_| |_|\__,_|\___|_|
                                        %s
    AI-Powered S3 Bucket Enumeration Tool
    Author: Ali Sünbül (xeloxa)
    Email:  alisunbul@proton.me
    Repo:   https://github.com/xeloxa/s3finder
    ─────────────────────────────────────────
`
		fmt.Printf(banner, version)
		return
	}

	lines := []string{
		"     ____  _____  __ _           _",
		"    / ___|___ / / _(_)_ __   __| | ___ _ __",
		"    \\___ \\ |_ \\| |_| | '_ \\ / _` |/ _ \\ '__|",
		"     ___) |__) |  _| | | | | (_| |  __/ |",
		"    |____/____/|_| |_|_| |_|\\__,_|\\___|_|",
		"                                        " + version,
		"    AI-Powered S3 Bucket Enumeration Tool",
		"    Author: Ali Sünbül (xeloxa)",
		"    Email:  alisunbul@proton.me",
		"    Repo:   https://github.com/xeloxa/s3finder",
		"    ─────────────────────────────────────────",
	}

	type rangeColors struct {
		startR, startG, startB int
		endR, endG, endB       int
	}

	logoGradient := rangeColors{76, 175, 80, 118, 221, 118}
	infoGradient := rangeColors{0, 188, 212, 33, 150, 243}
	authorGradient := rangeColors{255, 235, 59, 255, 152, 0}
	sepGradient := rangeColors{158, 158, 158, 66, 66, 66}

	fmt.Println() // Add top margin
	for i, line := range lines {
		var grad rangeColors
		switch {
		case i <= 4:
			grad = logoGradient
		case i <= 6:
			grad = infoGradient
		case i <= 9:
			grad = authorGradient
		default:
			grad = sepGradient
		}

		for j, char := range line {
			p := float64(j) / float64(len(line))
			r := int(float64(grad.startR) + p*float64(grad.endR-grad.startR))
			g := int(float64(grad.startG) + p*float64(grad.endG-grad.startG))
			b := int(float64(grad.startB) + p*float64(grad.endB-grad.startB))

			fmt.Printf("\033[38;2;%d;%d;%dm%c", r, g, b, char)
		}
		fmt.Println("\033[0m")
	}
	fmt.Println() // Add bottom margin
}
