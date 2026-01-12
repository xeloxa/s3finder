package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/xeloxa/s3finder/internal/config"
	"github.com/xeloxa/s3finder/pkg/ai"
	"github.com/xeloxa/s3finder/pkg/output"
	"github.com/xeloxa/s3finder/pkg/permutation"
	"github.com/xeloxa/s3finder/pkg/scanner"
)

var (
	version   = "dev"
	buildTime = "unknown"
	cfg     = config.Default()
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

	// Scanner flags
	rootCmd.Flags().IntVarP(&cfg.Workers, "threads", "t", cfg.Workers, "Number of concurrent workers")
	rootCmd.Flags().Float64Var(&cfg.MaxRPS, "rps", cfg.MaxRPS, "Maximum requests per second")
	rootCmd.Flags().IntVar(&cfg.Timeout, "timeout", cfg.Timeout, "Request timeout in seconds")
	rootCmd.Flags().BoolVar(&cfg.DeepInspect, "deep", cfg.DeepInspect, "Perform deep inspection on found buckets")

	// Input flags
	rootCmd.Flags().StringVarP(&cfg.Seed, "seed", "s", "", "Target keyword for bucket name generation (required)")
	rootCmd.Flags().StringVarP(&cfg.Wordlist, "wordlist", "w", "", "Path to wordlist file")

	// AI flags
	rootCmd.Flags().BoolVar(&cfg.AIEnabled, "ai", cfg.AIEnabled, "Enable AI-powered name generation")
	rootCmd.Flags().StringVar(&cfg.AIProvider, "ai-provider", cfg.AIProvider, "AI provider (openai, ollama, anthropic)")
	rootCmd.Flags().StringVar(&cfg.AIModel, "ai-model", cfg.AIModel, "AI model name")
	rootCmd.Flags().StringVar(&cfg.AIKey, "ai-key", "", "AI provider API key (or use env: OPENAI_API_KEY, ANTHROPIC_API_KEY)")
	rootCmd.Flags().StringVar(&cfg.AIBaseURL, "ai-url", "", "AI provider base URL (for Ollama)")
	rootCmd.Flags().IntVar(&cfg.AICount, "ai-count", cfg.AICount, "Number of AI-generated names")

	// Output flags
	rootCmd.Flags().StringVarP(&cfg.OutputFile, "output", "o", cfg.OutputFile, "Output file path")
	rootCmd.Flags().StringVarP(&cfg.OutputFormat, "format", "f", cfg.OutputFormat, "Output format (json, txt)")
	rootCmd.Flags().BoolVar(&cfg.NoColor, "no-color", cfg.NoColor, "Disable colored output")
	rootCmd.Flags().BoolVarP(&cfg.Verbose, "verbose", "v", cfg.Verbose, "Verbose output")

	rootCmd.MarkFlagRequired("seed")

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
		}
	}

	// Banner
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

	// Setup output writers
	realtimeWriter := output.NewRealtime(&output.RealtimeConfig{
		Output:    os.Stdout,
		UseColors: !cfg.NoColor,
		Verbose:   cfg.Verbose,
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

	// Process results
	for result := range results {
		if err := multiWriter.WriteResult(result); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing result: %v\n", err)
		}
	}

	// Print summary
	stats := s.Stats()
	duration := time.Since(startTime).Round(time.Second)

	fmt.Printf("\n%s\n", "────────────────────────────────────────")
	fmt.Printf("Scan completed in %s\n", duration)
	fmt.Printf("Scanned: %d | Found: %d | Public: %d | Private: %d | Errors: %d\n",
		stats.Scanned, stats.Found, stats.Public, stats.Private, stats.Errors)
	fmt.Printf("Results saved to: %s\n", cfg.OutputFile)

	return nil
}

func generateNames(ctx context.Context) ([]string, error) {
	seen := make(map[string]struct{})
	var allNames []string

	add := func(names []string) {
		for _, name := range names {
			if _, ok := seen[name]; !ok {
				seen[name] = struct{}{}
				allNames = append(allNames, name)
			}
		}
	}

	// 1. Permutation engine on seed
	engine := permutation.Default()
	permNames := engine.Generate(cfg.Seed)
	add(permNames)
	fmt.Printf("Permutation engine generated %d names\n", len(permNames))

	// 2. Wordlist + permutations
	if cfg.Wordlist != "" {
		words, err := config.LoadWordlist(cfg.Wordlist)
		if err != nil {
			return nil, fmt.Errorf("failed to load wordlist: %w", err)
		}
		wordlistNames := engine.GenerateFromWordlist(words, cfg.Seed)
		add(wordlistNames)
		fmt.Printf("Wordlist generated %d names\n", len(wordlistNames))
	}

	// 3. AI generation
	if cfg.AIEnabled {
		fmt.Printf("Generating AI suggestions using %s...\n", cfg.AIProvider)

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
			aiNames, err := generator.Generate(ctx, cfg.Seed, cfg.AICount)
			if err != nil {
				fmt.Printf("Warning: AI generation failed: %v\n", err)
			} else {
				add(aiNames)
				fmt.Printf("AI (%s) generated %d names\n", generator.Name(), len(aiNames))
			}
		}
	}

	return allNames, nil
}

func printBanner() {
	banner := `
     ____  _____  __ _           _
    / ___|___ / / _(_)_ __   __| | ___ _ __
    \___ \ |_ \| |_| | '_ \ / _` + "`" + ` |/ _ \ '__|
     ___) |__) |  _| | | | | (_| |  __/ |
    |____/____/|_| |_|_| |_|\__,_|\___|_|
                                        %s
    AI-Powered S3 Bucket Enumeration Tool
    ─────────────────────────────────────────
`
	fmt.Printf(banner, version)
}
