# Bender

Local AI Workflow Automator for macOS.

Bender is a background daemon that automates daily tasks using local or API-based LLMs. It monitors configurable triggers (clipboard changes, file system events, git operations, screenshots) and automatically applies LLM-powered transformations through end-to-end pipelines.

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                      macOS System                       │
│                                                         │
│  ┌───────────────────────────────────────────────────┐  │
│  │              Bender Daemon (Go)                   │  │
│  │                                                   │  │
│  │  Clipboard Monitor │ FSEvents │ Git Hook Handler  │  │
│  │                    ↓                              │  │
│  │              LLM Router                           │  │
│  │        (Ollama / OpenAI / Anthropic)              │  │
│  │                    ↓                              │  │
│  │          Unix Socket API                          │  │
│  └───────────────────────────────────────────────────┘  │
│                         │                               │
│         ┌───────────────┼───────────────┐               │
│         ↓               ↓               ↓               │
│    CLI (TS)      Dashboard (Next.js)   Git Hooks        │
└─────────────────────────────────────────────────────────┘
```

## Features

- **Auto-file pipeline**: watch directory → settle → classify → move → rename → notify (with undo)
- **Screenshot pipeline**: watch directory → settle → tag via vision LLM → rename → move → notify (with undo)
- **Clipboard summarization**: auto-summarize long clipboard content via LLM
- **Git commit messages**: generate conventional commit messages from diffs
- **File classification**: categorize files by extension or LLM analysis
- **macOS integration**: notifications, Keychain for API key storage

## Components

| Component | Technology | Location |
|-----------|------------|----------|
| Daemon | Go 1.22+ | `daemon/` |
| CLI | TypeScript/Node.js | `cli/` |
| Dashboard | Next.js 14 | `dashboard/` |

## Requirements

- macOS
- Go 1.22+
- Node.js 20+
- Ollama (optional, for local LLM inference)

## Installation

```bash
# Install the daemon and CLI
./scripts/install.sh

# Uninstall
./scripts/uninstall.sh
```

## Development

### Daemon

```bash
cd daemon
go build -o benderd ./cmd/benderd
./benderd
```

### CLI

```bash
cd cli
npm install
npm run build
npm start
```

### Dashboard

```bash
cd dashboard
npm install
npm run dev
```

## CLI Usage

```bash
# Daemon management
bender start | stop | restart | status

# Ad-hoc tasks
bender summarize [text]        # Summarize clipboard or text
bender classify <file>         # Classify a file
bender rename <files...>       # Generate intelligent filenames
bender commit [--auto]         # Generate git commit message
bender screenshot <file>       # Tag a screenshot with vision AI
bender undo <task-id>          # Reverse file operations

# Pipelines
bender pipeline status                    # Show pipeline config/state
bender pipeline run auto-file <file>      # Run auto-file pipeline on a file
bender pipeline run screenshot <file>     # Run screenshot pipeline on a file

# Other
bender tasks [-l N] [-s status]  # View task queue history
bender logs [-f] [--level lvl]   # View daemon logs
bender config [get|set] [key]    # Manage configuration
bender keychain set|get|delete|list  # Manage API keys in macOS Keychain
```

## Configuration

Configuration files are stored in `~/.config/bender/` as YAML with JSON Schema validation.

Key pipeline settings in `~/.config/bender/config.yaml`:

```yaml
auto_file:
  auto_move: true          # Enable full auto-file pipeline
  auto_rename: true        # LLM-powered rename after classification
  settle_delay_ms: 3000    # Wait for file writes to complete

screenshots:
  settle_delay_ms: 2000    # Wait for screenshot to finish writing
```

## Testing

```bash
cd daemon && go test ./...      # 36 Go tests
cd cli && npx vitest run        # 31 CLI tests
```

## License

MIT
