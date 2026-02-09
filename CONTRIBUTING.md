# Contributing to Bender

## Development Setup

### Requirements

- macOS (Keychain integration, pbpaste, osascript)
- Go 1.22+
- Node.js 20+
- Ollama (optional, for local LLM testing)

### Getting Started

```bash
# Clone the repo
git clone <repo-url>
cd bender-automator

# Build the daemon
cd daemon
go build ./cmd/benderd/

# Install CLI dependencies
cd ../cli
npm install

# Install dashboard dependencies
cd ../dashboard
npm install
```

## Project Structure

```
daemon/                 # Go daemon
  cmd/benderd/          # Entry point and handler wiring
  internal/
    api/                # JSON-RPC server over Unix socket
    clipboard/          # Clipboard monitoring (pbpaste)
    config/             # YAML config loading and validation
    fileops/            # File move/rename with undo tracking
    fswatch/            # Directory watcher
    git/                # Git operations
    keychain/           # macOS Keychain integration
    llm/                # LLM provider router (Ollama, OpenAI, Anthropic)
    logging/            # Logger with ring buffer
    notify/             # macOS notifications
    task/               # SQLite-backed task queue

cli/                    # TypeScript CLI
  src/
    commands/           # Command implementations
    lib/                # JSON-RPC client

dashboard/              # Next.js 14 dashboard
  app/                  # Pages (tasks, config, logs)
  lib/                  # Shared daemon client helper

configs/                # Default config and JSON Schema
scripts/                # Install/uninstall scripts
```

## Running Tests

### Daemon (Go)

```bash
cd daemon
go test ./...
```

### CLI (TypeScript)

```bash
cd cli
npm test
```

## Code Style

### Go

- Follow standard Go conventions (`gofmt`, `go vet`)
- Task handlers: `func(ctx context.Context, payload json.RawMessage) (json.RawMessage, error)`
- API handlers: `func(ctx context.Context, params json.RawMessage) (any, error)`
- Use `internal/logging` for all log output

### TypeScript

- ESM modules (`"type": "module"`)
- Commands export a single async function matching the command name
- Use `chalk` for colored output, `ora` for spinners

## Architecture Notes

- The daemon communicates via JSON-RPC 2.0 over a Unix socket at `/tmp/bender.sock`
- Config lives at `~/.config/bender/config.yaml`
- SQLite database at `~/.local/share/bender/bender.db`
- API keys can be stored in macOS Keychain (prefix with `keychain:` in config)
- Dashboard API routes proxy to the daemon via `lib/daemon.ts`

## Making Changes

1. Create a feature branch
2. Make your changes with tests
3. Ensure `go test ./...` and `npm test` pass
4. Submit a pull request

## Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: add new feature
fix: resolve bug
test: add tests
docs: update documentation
chore: maintenance tasks
```
