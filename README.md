<p align="center">
  <img src="logo.png" alt="s3finder" width="400">
</p>

<p align="center">
  <strong>AI-Powered S3 Bucket Enumeration Tool</strong>
</p>

<p align="center">
  <a href="https://xeloxa.github.io/s3finder/">Documentation</a> •
  <a href="#installation">Installation</a> •
  <a href="#quick-start">Quick Start</a> •
  <a href="#features">Features</a>
</p>

<p align="center">
  <a href="https://xeloxa.github.io/s3finder/"><img src="https://img.shields.io/badge/Docs-GitHub%20Pages-blue?style=flat" alt="Documentation"></a>
  <a href="https://go.dev/"><img src="https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go" alt="Go Version"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-MIT-blue.svg" alt="License"></a>
  <a href="https://github.com/xeloxa/s3finder/releases"><img src="https://img.shields.io/badge/Platform-Linux%20%7C%20macOS%20%7C%20Windows-lightgrey" alt="Platform"></a>
</p>

---

A high-performance CLI tool for discovering AWS S3 buckets using intelligent name generation. Decouples input sources for precise control: permutations only apply to the provided seed, while wordlists and CT logs are processed as raw inputs.

<p align="center">
  <img src="banner.png" alt="S3Finder Banner" width="100%">
</p>

## Features

- **Decoupled Input Sources** — Independent handling of seeds, wordlists, and domains (no cross-contamination)
- **Optional Seed** — Scan using only a wordlist or domain without requiring a seed keyword
- **High-Concurrency Scanning** — Worker pool architecture handles thousands of requests simultaneously
- **CT Log Reconnaissance** — Discover subdomains via Certificate Transparency logs (crt.sh) with automatic word extraction
- **AI-Powered Generation** — OpenAI, Ollama, Anthropic, or Gemini generate context-aware bucket name variations
- **Permutation Engine** — 780+ automatic variations per seed (suffixes, prefixes, years, regions)
- **Adaptive Rate Limiting** — AIMD algorithm auto-adjusts to avoid throttling and IP blocks
- **Deep Inspection** — AWS SDK integration reveals region, ACL status, and sample objects
- **Live Progress Bar** — Real-time TUI showing scanned count, RPS, ETA, and discovery stats
- **HTTP/2 & Connection Pooling** — Optimized networking with keep-alives and connection reuse
- **Smart Retry Logic** — Automatic retries with exponential backoff for transient failures
- **Custom DNS Resolver** — Uses Google/Cloudflare DNS to prevent local resolver saturation
- **Multiple Formats** — Export results as JSON or TXT for post-processing
- **Cross-Platform** — Native binaries for Linux, macOS, and Windows (amd64 & arm64)

---

<p align="center">
  <img src="demo.gif" alt="s3finder demo" >
</p>

---

## Installation

### Download Binary (Recommended)

Download the latest release for your platform:

| Platform | Architecture | Download |
|----------|--------------|----------|
| Linux | amd64 | [s3finder-linux-amd64.tar.gz](https://github.com/xeloxa/s3finder/releases/latest) |
| Linux | arm64 | [s3finder-linux-arm64.tar.gz](https://github.com/xeloxa/s3finder/releases/latest) |
| macOS | Intel | [s3finder-darwin-amd64.tar.gz](https://github.com/xeloxa/s3finder/releases/latest) |
| macOS | Apple Silicon | [s3finder-darwin-arm64.tar.gz](https://github.com/xeloxa/s3finder/releases/latest) |
| Windows | amd64 | [s3finder-windows-amd64.zip](https://github.com/xeloxa/s3finder/releases/latest) |
| Windows | arm64 | [s3finder-windows-arm64.zip](https://github.com/xeloxa/s3finder/releases/latest) |

### Homebrew (macOS/Linux)

```bash
brew install xeloxa/tap/s3finder
```

### Go Install

```bash
go install github.com/xeloxa/s3finder/cmd/s3finder@latest
```

### Build from Source

```bash
git clone https://github.com/xeloxa/s3finder.git
cd s3finder

# Build for current platform
make build

# Build for all platforms
make build-all

# Or use go directly
go build -o s3finder ./cmd/s3finder
```

---

## Quick Start

```bash
# Basic scan with permutations of a seed
s3finder -s acme-corp

# Scan using ONLY a wordlist (no permutations)
s3finder -w wordlist.txt

# Scan using ONLY a domain (CT log discovery)
s3finder -d acme.com

# Combined independent sources
s3finder -s acme -w custom.txt -d acme.com

# High-speed scan
s3finder -s acme-corp -t 200 --rps 1000
```

---

## Usage

### Seed-Based Permutations

```bash
# Scan with 780+ permutations of a seed keyword
s3finder -s acme-corp
```

### Wordlist Scanning (Raw Mode)

Wordlists are now processed as raw inputs. They are **not** combined with the seed or permuted, giving you exact control over what is scanned.

```bash
# Scan exactly what is in the wordlist
s3finder -w wordlists/common.txt
```

### CT Log Reconnaissance (As-Is Mode)

Discovered subdomains are scanned exactly as they appear in Certificate Transparency logs. Unique words are extracted from subdomains and used to generate additional permutations for deeper scanning.

```bash
# Fetch and scan subdomains from CT logs
s3finder -d acme.com

# Limit CT results (default: 100)
s3finder -d acme.com --ct-limit 50
```

> [!NOTE]
> Bucket names containing dots (e.g., `dev.acme.com`) may trigger SSL/TLS certificate warnings due to virtual-hosted style access limitations.

### AI-Powered Scanning

AI generation analyzes CT log patterns and generates bucket names matching organizational naming conventions.

```bash
# OpenAI (default: gpt-4o-mini)
export OPENAI_API_KEY=sk-xxxxx
s3finder -s acme-corp --ai

# Anthropic Claude (default: claude-3-5-haiku-20241022)
export ANTHROPIC_API_KEY=sk-ant-xxxxx
s3finder -s acme-corp --ai --ai-provider anthropic

# Google Gemini (default: gemini-3-flash-preview)
export GEMINI_API_KEY=xxxxx
s3finder -s acme-corp --ai --ai-provider gemini

# Ollama local (default: llama3.2)
s3finder -s acme-corp --ai --ai-provider ollama

# Context-aware: combine with CT logs for pattern discovery
s3finder -s acme -d acme.com --ai
```

### High-Speed Scanning

```bash
# Aggressive scan with 200 workers and 1000 RPS
s3finder -s acme-corp -t 200 --rps 1000
```

### Output Options

```bash
# JSON report (default)
s3finder -s acme-corp -o results.json

# Plain text report
s3finder -s acme-corp -o results.txt -f txt

# Disable colors (for piping)
s3finder -s acme-corp --no-color
```

---

## Flags Reference

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--seed` | `-s` | | Target keyword for bucket name generation |
| `--domain` | `-d` | | Target domain for CT log subdomain discovery |
| `--ct-limit` | | `100` | Maximum subdomains to fetch from CT logs |
| `--wordlist` | `-w` | | Path to wordlist file |
| `--threads` | `-t` | `50` | Number of concurrent workers |
| `--rps` | | `150` | Maximum requests per second |
| `--timeout` | | `15` | Request timeout in seconds |
| `--deep` | | `true` | Perform deep inspection on found buckets |
| `--ai` | | `false` | Enable AI-powered name generation |
| `--ai-provider` | | `openai` | AI provider: `openai`, `ollama`, `anthropic`, `gemini` |
| `--ai-model` | | *provider default* | AI model name |
| `--ai-key` | | | API key (or use environment variables) |
| `--ai-url` | | | Base URL for custom endpoints or proxies |
| `--ai-count` | | `50` | Number of AI-generated names |
| `--output` | `-o` | `results.json` | Output file path |
| `--format` | `-f` | `json` | Output format: `json`, `txt` |
| `--no-color` | | `false` | Disable colored output |
| `--verbose` | `-v` | `false` | Verbose output |

> [!NOTE]
> At least one input source (`--seed`, `--wordlist`, `--domain`, or `--ai`) must be provided.

---

## Environment Variables

| Variable | Description |
|----------|-------------|
| `OPENAI_API_KEY` | OpenAI API key for AI generation |
| `ANTHROPIC_API_KEY` | Anthropic API key for Claude |
| `GEMINI_API_KEY` | Google Gemini API key |

---

## Build Commands

```bash
# Build for current platform
make build

# Build for all platforms (Linux, macOS, Windows × amd64, arm64)
make build-all

# Build for specific platform
make build-linux
make build-darwin
make build-windows

# Run tests
make test

# Run tests with coverage
make test-cover

# Create release archives
make release

# Clean build artifacts
make clean

# Show all available commands
make help
```

---

## Output Example

### Terminal Output

```
     ____  _____  __ _           _
    / ___|___ / / _(_)_ __   __| | ___ _ __
    \___ \ |_ \| |_| | '_ \ / _` |/ _ \ '__|
     ___) |__) |  _| | | | | (_| |  __/ |
    |____/____/|_| |_|_| |_|\__,_|\___|_|
                                        v1.2.4
    AI-Powered S3 Bucket Enumeration Tool
    ─────────────────────────────────────────

Permutation engine generated 780 names
AI (openai) generated 48 names
Generated 828 unique bucket names to scan

[PUBLIC] acme-corp-backup (objects: 1547, region: us-east-1)
         https://acme-corp-backup.s3.amazonaws.com
[PRIVATE] acme-corp-internal (region: eu-west-1)
[PUBLIC] acme-corp-assets-2024 (objects: 100+, region: us-west-2)
         https://acme-corp-assets-2024.s3.amazonaws.com

[████████████████████████████████] 100.0% [828/828] Public:2 Private:1 Err:0 145 r/s ETA:0s [2m34s]

────────────────────────────────────────
Scan completed in 2m34s
Scanned: 828 | Found: 3 | Public: 2 | Private: 1 | Errors: 0
Results saved to: results.json
```

### Progress Bar

During scanning, a live TUI progress bar displays real-time statistics:
- **Visual progress** - Fill bar showing scan completion percentage
- **Scanned count** - Current/total buckets scanned
- **Public/Private/Errors** - Real-time discovery counts
- **RPS** - Current requests per second
- **ETA** - Estimated time remaining
- **Elapsed time** - Total time since scan started

### JSON Report

```json
{
  "generated_at": "2025-01-12T15:30:00Z",
  "scan_duration": "2m34s",
  "total_found": 3,
  "public_buckets": 2,
  "private_buckets": 1,
  "results": [
    {
      "bucket": "acme-corp-backup",
      "probe_result": "public",
      "inspect": {
        "bucket": "acme-corp-backup",
        "exists": true,
        "is_public": true,
        "acl": "public-read",
        "region": "us-east-1",
        "object_count": 1547,
        "sample_keys": ["db-dump.sql", "config.yml", "backup-2024.tar.gz"]
      }
    }
  ]
}
```

---

## Supported Platforms

| Platform | Architecture | Status |
|----------|--------------|--------|
| Linux | amd64 | ✅ Supported |
| Linux | arm64 | ✅ Supported |
| macOS | amd64 (Intel) | ✅ Supported |
| macOS | arm64 (Apple Silicon) | ✅ Supported |
| Windows | amd64 | ✅ Supported |
| Windows | arm64 | ✅ Supported |

### Platform-Specific Notes

**Windows:**
- ANSI colors are enabled automatically on Windows 10+
- Use PowerShell or Windows Terminal for best experience
- Legacy cmd.exe may not display colors correctly

**macOS:**
- Both Intel and Apple Silicon are natively supported
- No Rosetta required for M1/M2/M3 Macs

**Linux:**
- Works on all major distributions
- ARM64 builds for Raspberry Pi and AWS Graviton

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         SCANNER ORCHESTRATOR                     │
├─────────────────────────────────────────────────────────────────┤
│  Wordlist → AI Generator → Permutation Engine                   │
│                             │                                    │
│                             ▼                                    │
│                   ┌──────────────────┐                          │
│                   │   names channel   │                          │
│                   └────────┬─────────┘                          │
│         ┌──────────────────┼──────────────────┐                 │
│         ▼                  ▼                  ▼                 │
│   ┌──────────┐       ┌──────────┐       ┌──────────┐           │
│   │ Worker 1 │       │ Worker 2 │       │ Worker N │           │
│   └────┬─────┘       └────┬─────┘       └────┬─────┘           │
│        └──────────────────┼──────────────────┘                  │
│                           ▼                                     │
│              ┌───────────────────────────┐                      │
│              ▼                           ▼                      │
│     ┌─────────────┐              ┌─────────────┐                │
│     │  Inspector  │              │   Output    │                │
│     │ (AWS SDK)   │              │   Writer    │                │
│     └─────────────┘              └─────────────┘                │
└─────────────────────────────────────────────────────────────────┘
```

---

## Project Structure

```
s3finder/
├── cmd/s3finder/          # CLI entrypoint
├── pkg/
│   ├── scanner/           # Worker pool, prober, inspector
│   ├── ai/                # LLM providers (OpenAI, Ollama, Anthropic, Gemini)
│   ├── recon/             # CT log reconnaissance (crt.sh)
│   ├── permutation/       # Name generation engine
│   ├── ratelimit/         # Adaptive AIMD rate limiter
│   └── output/            # Real-time + report writers
├── internal/config/       # Configuration management
├── wordlists/             # Default wordlists
├── Makefile               # Build automation
└── .goreleaser.yaml       # Release automation
```

---

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Run tests (`make test`)
4. Commit your changes (`git commit -m 'Add amazing feature'`)
5. Push to the branch (`git push origin feature/amazing-feature`)
6. Open a Pull Request

---

## Disclaimer

This tool is intended for **authorized security testing** and **research purposes only**. Only scan buckets belonging to organizations you have explicit permission to test. Unauthorized access to AWS resources is illegal.

---

## License

MIT License - see [LICENSE](LICENSE) for details.
