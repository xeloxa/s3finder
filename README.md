```
     ____  _____  __ _           _
    / ___|___ / / _(_)_ __   __| | ___ _ __
    \___ \ |_ \| |_| | '_ \ / _` |/ _ \ '__|
     ___) |__) |  _| | | | | (_| |  __/ |
    |____/____/|_| |_|_| |_|\__,_|\___|_|
```

# s3finder

**AI-Powered S3 Bucket Enumeration Tool**

A high-performance CLI tool for discovering AWS S3 buckets using intelligent name generation. Combines traditional wordlist scanning with LLM-powered suggestions to find buckets that other tools miss.

---

## Features

- **High-Concurrency Scanning** — Worker pool architecture handles thousands of requests simultaneously
- **AI-Powered Generation** — OpenAI, Ollama, or Anthropic generate smart bucket name variations
- **Permutation Engine** — 780+ automatic variations per seed (suffixes, prefixes, years, regions)
- **Adaptive Rate Limiting** — AIMD algorithm auto-adjusts to avoid throttling and IP blocks
- **Deep Inspection** — AWS SDK integration reveals region, ACL status, and sample objects
- **Real-Time Output** — Colored terminal output shows discoveries as they happen
- **Multiple Formats** — Export results as JSON or TXT for post-processing

---

## Installation

### From Source

```bash
go install github.com/xeloxa/s3finder/cmd/s3finder@latest
```

### Build Locally

```bash
git clone https://github.com/xeloxa/s3finder.git
cd s3finder
go build -o s3finder ./cmd/s3finder
```

---

## Usage

### Basic Scan

```bash
# Scan with permutations of a seed keyword
s3finder -s acme-corp
```

### With Wordlist

```bash
# Combine wordlist with seed-based permutations
s3finder -s acme-corp -w wordlists/common.txt
```

### AI-Powered Scanning

```bash
# OpenAI (default)
export OPENAI_API_KEY=sk-xxxxx
s3finder -s acme-corp --ai

# Anthropic Claude
export ANTHROPIC_API_KEY=sk-ant-xxxxx
s3finder -s acme-corp --ai --ai-provider anthropic

# Local Ollama
s3finder -s acme-corp --ai --ai-provider ollama --ai-url http://localhost:11434 --ai-model llama3
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
| `--seed` | `-s` | *required* | Target keyword for bucket name generation |
| `--wordlist` | `-w` | | Path to wordlist file |
| `--threads` | `-t` | `100` | Number of concurrent workers |
| `--rps` | | `500` | Maximum requests per second |
| `--timeout` | | `10` | Request timeout in seconds |
| `--deep` | | `true` | Perform deep inspection on found buckets |
| `--ai` | | `false` | Enable AI-powered name generation |
| `--ai-provider` | | `openai` | AI provider: `openai`, `ollama`, `anthropic` |
| `--ai-model` | | `gpt-4o-mini` | AI model name |
| `--ai-key` | | | API key (or use environment variables) |
| `--ai-url` | | | Base URL for Ollama |
| `--ai-count` | | `50` | Number of AI-generated names |
| `--output` | `-o` | `results.json` | Output file path |
| `--format` | `-f` | `json` | Output format: `json`, `txt` |
| `--no-color` | | `false` | Disable colored output |
| `--verbose` | `-v` | `false` | Verbose output |

---

## Environment Variables

| Variable | Description |
|----------|-------------|
| `OPENAI_API_KEY` | OpenAI API key for AI generation |
| `ANTHROPIC_API_KEY` | Anthropic API key for Claude |

---

## Output Example

### Terminal Output

```
     ____  _____  __ _           _
    / ___|___ / / _(_)_ __   __| | ___ _ __
    \___ \ |_ \| |_| | '_ \ / _` |/ _ \ '__|
     ___) |__) |  _| | | | | (_| |  __/ |
    |____/____/|_| |_|_| |_|\__,_|\___|_|
                                        v1.0.0
    AI-Powered S3 Bucket Enumeration Tool
    ─────────────────────────────────────────

Permutation engine generated 780 names
AI (openai) generated 48 names
Generated 828 unique bucket names to scan

[PUBLIC] acme-corp-backup (objects: 1547, region: us-east-1)
[PRIVATE] acme-corp-internal (region: eu-west-1)
[PUBLIC] acme-corp-assets-2024 (objects: 100+, region: us-west-2)

────────────────────────────────────────
Scan completed in 2m34s
Scanned: 828 | Found: 3 | Public: 2 | Private: 1 | Errors: 0
Results saved to: results.json
```

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
│   ├── ai/                # LLM providers (OpenAI, Ollama, Anthropic)
│   ├── permutation/       # Name generation engine
│   ├── ratelimit/         # Adaptive AIMD rate limiter
│   └── output/            # Real-time + report writers
├── internal/config/       # Configuration management
└── wordlists/             # Default wordlists
```

---

## Disclaimer

This tool is intended for **authorized security testing** and **research purposes only**. Only scan buckets belonging to organizations you have explicit permission to test. Unauthorized access to AWS resources is illegal.

---

## License

MIT License - see [LICENSE](LICENSE) for details.
