# Bender

Local AI Workflow Automator for macOS.

Bender is a background daemon that automates daily tasks using local or API-based LLMs. It monitors configurable triggers (clipboard changes, file system events, git operations) and automatically applies LLM-powered transformations.

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

## Configuration

Configuration files are stored in `~/.config/bender/` as YAML with JSON Schema validation.

## License

MIT
