# Product Requirements Document: Bender

## Local AI Workflow Automator for macOS

**Version:** 1.0  
**Status:** Draft  
**Last Updated:** February 2026

---

## 1. Overview

### 1.1 Product Summary

Bender is a macOS background daemon that automates daily tasks using local or API-based LLMs. It consists of three components: a Go daemon running as a LaunchAgent, a TypeScript CLI for interaction, and an optional Next.js dashboard for configuration and monitoring.

### 1.2 Problem Statement

Power users and developers perform repetitive tasks daily that could benefit from AI assistance: organizing files, generating commit messages, summarizing content, and tagging assets. Currently, these tasks require manual context-switching to AI tools or are left undone entirely.

### 1.3 Solution

Bender runs silently in the background, monitoring configurable triggers (clipboard changes, file system events, git operations) and automatically applying LLM-powered transformations. Users interact through a CLI for ad-hoc commands and configuration, with an optional web dashboard for visual management.

### 1.4 Target Users

- Developers seeking automated git workflows
- Knowledge workers managing large file collections
- Power users who want AI assistance without manual prompting
- Privacy-conscious users preferring local LLM inference

---

## 2. Architecture

### 2.1 System Components

```
┌─────────────────────────────────────────────────────────────────┐
│                         macOS System                            │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │                    LaunchAgent (plist)                     │ │
│  │                 ~/Library/LaunchAgents/                    │ │
│  └────────────────────────────────────────────────────────────┘ │
│                              │                                  │
│                              ▼                                  │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │                  Bender Daemon (Go)                        │ │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌────────────┐  │ │
│  │  │ Clipboard│  │   FSEvents  │  │  Git Hook │  │  Task Queue │ │
│  │  │ Monitor  │  │  Watcher │  │  Handler │  │            │  │ │
│  │  └────┬─────┘  └────┬─────┘  └────┬─────┘  └──────┬─────┘  │ │
│  │       └─────────────┴─────────────┴───────────────┘        │ │
│  │                           │                                │ │
│  │                           ▼                                │ │
│  │  ┌──────────────────────────────────────────────────────┐  │ │
│  │  │                  LLM Router                          │  │ │
│  │  │   ┌─────────┐    ┌─────────┐    ┌─────────────────┐  │  │ │
│  │  │   │ Ollama  │    │ OpenAI  │    │ Anthropic API   │  │  │ │
│  │  │   │ (local) │    │   API   │    │                 │  │  │ │
│  │  │   └─────────┘    └─────────┘    └─────────────────┘  │  │ │
│  │  └──────────────────────────────────────────────────────┘  │ │
│  │                           │                                │ │
│  │                           ▼                                │ │
│  │  ┌──────────────────────────────────────────────────────┐  │ │
│  │  │           Unix Socket / HTTP API                     │  │ │
│  │  │              /tmp/bender.sock                        │  │ │
│  │  └──────────────────────────────────────────────────────┘  │ │
│  └────────────────────────────────────────────────────────────┘ │
│                              │                                  │
│            ┌─────────────────┼─────────────────┐                │
│            ▼                 ▼                 ▼                │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐   │
│  │   CLI (TS)   │  │  Dashboard   │  │   Git Hooks          │   │
│  │   bender     │  │  (Next.js)   │  │   (installed by CLI) │   │
│  └──────────────┘  └──────────────┘  └──────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

### 2.2 Technology Stack

| Component | Technology | Rationale |
|-----------|------------|-----------|
| Daemon | Go 1.22+ | Native macOS APIs via cgo, excellent concurrency, single binary distribution |
| CLI | TypeScript + Node.js | Rapid development, good DX for configuration files, npm distribution |
| Dashboard | Next.js 14 | React ecosystem, API routes for daemon communication, optional deployment |
| IPC | Unix Domain Socket | Fast, secure, no network exposure |
| Config | YAML + JSON Schema | Human-readable, IDE support with schema validation |
| Database | SQLite | Embedded, zero-config, stores task history and logs |

### 2.3 Directory Structure

```
bender/
├── daemon/                    # Go daemon
│   ├── cmd/
│   │   └── benderd/
│   │       └── main.go
│   ├── internal/
│   │   ├── clipboard/         # Clipboard monitoring
│   │   ├── fswatch/           # File system events
│   │   ├── git/               # Git integration
│   │   ├── llm/               # LLM provider abstraction
│   │   │   ├── provider.go    # Interface
│   │   │   ├── ollama.go
│   │   │   ├── openai.go
│   │   │   └── anthropic.go
│   │   ├── task/              # Task queue and execution
│   │   ├── api/               # Unix socket API
│   │   └── config/            # Configuration loading
│   ├── go.mod
│   └── go.sum
├── cli/                       # TypeScript CLI
│   ├── src/
│   │   ├── index.ts
│   │   ├── commands/
│   │   │   ├── start.ts
│   │   │   ├── stop.ts
│   │   │   ├── status.ts
│   │   │   ├── config.ts
│   │   │   ├── logs.ts
│   │   │   ├── run.ts         # Ad-hoc task execution
│   │   │   └── install.ts     # Git hooks, LaunchAgent
│   │   └── lib/
│   │       ├── client.ts      # Daemon IPC client
│   │       └── config.ts      # Config file handling
│   ├── package.json
│   └── tsconfig.json
├── dashboard/                 # Optional Next.js dashboard
│   ├── app/
│   │   ├── page.tsx           # Dashboard home
│   │   ├── tasks/
│   │   ├── config/
│   │   └── logs/
│   ├── components/
│   └── package.json
├── configs/
│   ├── schema.json            # JSON Schema for config validation
│   └── default.yaml           # Default configuration
└── scripts/
    ├── install.sh
    └── uninstall.sh
```

---

## 3. Core Features

### 3.1 Clipboard Summarization

**Trigger:** Clipboard content changes and exceeds configurable length threshold

**Behavior:**
1. Detect clipboard change via macOS Pasteboard API
2. If content length > threshold (default: 500 chars), queue summarization task
3. Send content to configured LLM with summarization prompt
4. Store summary in Bender's internal clipboard buffer
5. Optionally show macOS notification with summary preview
6. User can paste summary via `bender paste-summary` or keyboard shortcut

**Configuration:**
```yaml
clipboard:
  enabled: true
  min_length: 500
  debounce_ms: 1000
  auto_summarize: true
  notification: true
  prompt: "Summarize the following text concisely in 2-3 sentences:"
```

### 3.2 Auto-File Downloads

**Trigger:** New file appears in watched directory (default: ~/Downloads)

**Behavior:**
1. Monitor directory via FSEvents
2. On new file, extract metadata (filename, extension, size, creation date)
3. For documents/text files, optionally extract content preview
4. Send metadata to LLM with classification prompt
5. LLM returns suggested destination folder
6. Move file to destination (with conflict resolution)
7. Log action for undo capability

**Configuration:**
```yaml
auto_file:
  enabled: true
  watch_dirs:
    - ~/Downloads
  destination_root: ~/Documents/Sorted
  categories:
    - name: receipts
      path: ~/Documents/Finances/Receipts
      description: "Purchase receipts, invoices, bills"
    - name: screenshots
      path: ~/Pictures/Screenshots
      description: "Screen captures and app screenshots"
    - name: documents
      path: ~/Documents/General
      description: "PDFs, Word docs, text files"
  exclude_patterns:
    - "*.crdownload"
    - "*.part"
    - ".DS_Store"
  prompt_template: |
    Given a file with the following properties:
    - Filename: {{filename}}
    - Extension: {{extension}}
    - Size: {{size}}
    - Content preview: {{preview}}
    
    Available categories:
    {{#categories}}
    - {{name}}: {{description}} (path: {{path}})
    {{/categories}}
    
    Return ONLY the category name that best fits this file.
```

### 3.3 Intelligent File Renaming

**Trigger:** Manual CLI command or configurable file system event

**Behavior:**
1. Accept file path(s) as input
2. For images: extract EXIF data, run OCR if text-heavy
3. For documents: extract text preview
4. Send to LLM with naming convention prompt
5. Generate descriptive filename following user's conventions
6. Rename file (with conflict resolution: append -1, -2, etc.)

**Configuration:**
```yaml
rename:
  naming_convention: "kebab-case"  # kebab-case, snake_case, camelCase, PascalCase
  include_date: true
  date_format: "YYYY-MM-DD"
  max_length: 60
  prompt: |
    Generate a descriptive filename for this {{file_type}}.
    Content/Description: {{content}}
    Use {{naming_convention}} format.
    Include key identifiers but keep under {{max_length}} characters.
    Return ONLY the filename without extension.
```

### 3.4 Git Commit Message Generation

**Trigger:** Git pre-commit hook or manual CLI command

**Behavior:**
1. Capture staged diff via `git diff --cached`
2. Parse diff to extract: files changed, insertions, deletions
3. Send to LLM with commit message prompt
4. Generate conventional commit message
5. Present to user for approval/editing
6. Optionally auto-commit with generated message

**Configuration:**
```yaml
git:
  enabled: true
  auto_install_hooks: false
  commit_format: "conventional"  # conventional, simple, detailed
  include_scope: true
  include_body: true
  max_subject_length: 72
  prompt: |
    Generate a git commit message for the following changes:
    
    Files changed: {{files}}
    Diff:
    {{diff}}
    
    Use conventional commit format: type(scope): description
    Types: feat, fix, docs, style, refactor, test, chore
    Keep subject line under {{max_subject_length}} characters.
    Add a body explaining WHY if the change is non-trivial.
```

### 3.5 Screenshot Auto-Tagging

**Trigger:** New screenshot detected in screenshot directory

**Behavior:**
1. Monitor screenshot directory via FSEvents
2. On new image, send to vision-capable LLM
3. Extract: app name, content description, text visible
4. Generate tags and descriptive filename
5. Optionally write tags to EXIF/XMP metadata
6. Rename file with generated name

**Configuration:**
```yaml
screenshots:
  enabled: true
  watch_dir: ~/Desktop  # macOS default screenshot location
  destination: ~/Pictures/Screenshots
  rename: true
  add_metadata_tags: true
  vision_model: "gpt-4o"  # Must be vision-capable
  prompt: |
    Analyze this screenshot and provide:
    1. App or website shown (if identifiable)
    2. Brief description of content (under 10 words)
    3. Up to 5 relevant tags
    
    Return as JSON: {"app": "", "description": "", "tags": []}
```

---

## 4. Daemon Specification

### 4.1 Process Management

**LaunchAgent Configuration:**
```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.bender.daemon</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/benderd</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/usr/local/var/log/bender/stdout.log</string>
    <key>StandardErrorPath</key>
    <string>/usr/local/var/log/bender/stderr.log</string>
    <key>EnvironmentVariables</key>
    <dict>
        <key>BENDER_CONFIG</key>
        <string>~/.config/bender/config.yaml</string>
    </dict>
</dict>
</plist>
```

**Lifecycle:**
- Start: `launchctl load ~/Library/LaunchAgents/com.bender.daemon.plist`
- Stop: `launchctl unload ~/Library/LaunchAgents/com.bender.daemon.plist`
- Restart: Handled by KeepAlive, or manual unload/load

### 4.2 IPC Protocol

**Socket:** `/tmp/bender.sock` (Unix Domain Socket)

**Protocol:** JSON-RPC 2.0 over newline-delimited JSON

**Methods:**

```typescript
interface BenderAPI {
  // Status
  "status.get": () => DaemonStatus;
  "status.health": () => HealthCheck;
  
  // Configuration
  "config.get": () => Config;
  "config.set": (partial: Partial<Config>) => void;
  "config.reload": () => void;
  
  // Tasks
  "task.run": (task: TaskRequest) => TaskResult;
  "task.queue": () => QueuedTask[];
  "task.cancel": (taskId: string) => void;
  "task.history": (limit?: number) => TaskHistory[];
  
  // Features
  "clipboard.summarize": (text?: string) => Summary;
  "clipboard.get_summary": () => Summary | null;
  "file.classify": (path: string) => Classification;
  "file.rename": (path: string) => RenameResult;
  "git.generate_commit": (repoPath?: string) => CommitMessage;
  
  // Logs
  "logs.stream": () => AsyncIterable<LogEntry>;
  "logs.get": (filter?: LogFilter) => LogEntry[];
}
```

### 4.3 LLM Provider Interface

```go
// provider.go
package llm

type Provider interface {
    Name() string
    Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)
    CompleteWithVision(ctx context.Context, req VisionRequest) (*CompletionResponse, error)
    SupportsVision() bool
    MaxTokens() int
}

type CompletionRequest struct {
    Model       string
    Messages    []Message
    Temperature float64
    MaxTokens   int
}

type Message struct {
    Role    string // "system", "user", "assistant"
    Content string
}

type VisionRequest struct {
    CompletionRequest
    Images []Image
}

type Image struct {
    Data     []byte
    MimeType string
}

type CompletionResponse struct {
    Content      string
    FinishReason string
    Usage        Usage
}
```

**Supported Providers:**

| Provider | Local | Vision | Notes |
|----------|-------|--------|-------|
| Ollama | Yes | llava, bakllava | Default for privacy |
| OpenAI | No | gpt-4o, gpt-4o-mini | Best vision quality |
| Anthropic | No | claude-3+ | Alternative API |

### 4.4 Task Queue

**Requirements:**
- FIFO queue with priority support
- Concurrent execution limit (configurable, default: 2)
- Retry with exponential backoff
- Task timeout (configurable, default: 30s)
- Persistent queue survives daemon restart

**Implementation:**
```go
type TaskQueue struct {
    db          *sql.DB
    workers     int
    tasks       chan Task
    results     chan TaskResult
}

type Task struct {
    ID          string
    Type        TaskType
    Priority    int
    Payload     json.RawMessage
    CreatedAt   time.Time
    RetryCount  int
    MaxRetries  int
}

type TaskType string

const (
    TaskClipboardSummarize TaskType = "clipboard.summarize"
    TaskFileClassify       TaskType = "file.classify"
    TaskFileRename         TaskType = "file.rename"
    TaskGitCommit          TaskType = "git.commit"
    TaskScreenshotTag      TaskType = "screenshot.tag"
)
```

---

## 5. CLI Specification

### 5.1 Commands

```bash
# Daemon management
bender start              # Start daemon (installs LaunchAgent if needed)
bender stop               # Stop daemon
bender restart            # Restart daemon
bender status             # Show daemon status and stats

# Configuration
bender config             # Open config file in $EDITOR
bender config get <key>   # Get config value
bender config set <key> <value>  # Set config value
bender config validate    # Validate config file

# Ad-hoc tasks
bender summarize [text]   # Summarize clipboard or provided text
bender classify <file>    # Classify file and suggest location
bender rename <file...>   # Generate names for file(s)
bender commit [--auto]    # Generate commit message (--auto to commit)
bender tag <image...>     # Generate tags for image(s)

# Installation
bender install hooks      # Install git hooks in current repo
bender install agent      # Install LaunchAgent
bender uninstall hooks    # Remove git hooks
bender uninstall agent    # Remove LaunchAgent

# Logs and history
bender logs [-f] [--level=<level>]  # View logs (-f to follow)
bender history [--limit=N]          # View task history
bender undo <task-id>               # Undo a file operation

# Dashboard
bender dashboard          # Start local dashboard server
```

### 5.2 Output Formatting

- Default: Human-readable colored output
- `--json`: JSON output for scripting
- `--quiet`: Suppress non-essential output

### 5.3 Interactive Modes

**Commit message workflow:**
```
$ bender commit
Analyzing staged changes...

Generated commit message:
────────────────────────────────────────
feat(auth): add OAuth2 support for GitHub login

- Implement OAuth2 flow with PKCE
- Add GitHub provider configuration
- Store tokens securely in keychain
────────────────────────────────────────

[e]dit / [a]ccept / [r]egenerate / [c]ancel: a
Committed: abc1234
```

---

## 6. Configuration

### 6.1 File Locations

- Config: `~/.config/bender/config.yaml`
- Database: `~/.local/share/bender/bender.db`
- Logs: `/usr/local/var/log/bender/`
- Socket: `/tmp/bender.sock`

### 6.2 Full Configuration Schema

```yaml
# ~/.config/bender/config.yaml

# LLM Provider Configuration
llm:
  # Default provider for non-vision tasks
  default_provider: ollama
  
  # Provider configurations
  providers:
    ollama:
      enabled: true
      base_url: http://localhost:11434
      model: llama3.2
      timeout_seconds: 30
    
    openai:
      enabled: false
      api_key: ${OPENAI_API_KEY}  # Environment variable reference
      model: gpt-4o-mini
      vision_model: gpt-4o
      timeout_seconds: 60
    
    anthropic:
      enabled: false
      api_key: ${ANTHROPIC_API_KEY}
      model: claude-3-haiku-20240307
      timeout_seconds: 60

# Clipboard monitoring
clipboard:
  enabled: true
  min_length: 500
  debounce_ms: 1000
  auto_summarize: true
  notification: true
  notification_sound: false

# File auto-organization
auto_file:
  enabled: true
  watch_dirs:
    - ~/Downloads
  destination_root: ~/Documents/Sorted
  ignore_hidden: true
  exclude_patterns:
    - "*.crdownload"
    - "*.part"
    - "*.download"
    - ".DS_Store"
  categories:
    - name: images
      path: ~/Pictures/Downloads
      extensions: [jpg, jpeg, png, gif, webp, svg]
    - name: documents
      path: ~/Documents/Downloads
      extensions: [pdf, doc, docx, txt, md, rtf]
    - name: code
      path: ~/Code/Downloads
      extensions: [py, js, ts, go, rs, java, cpp, h]
    - name: archives
      path: ~/Downloads/Archives
      extensions: [zip, tar, gz, rar, 7z]
  use_llm_classification: true  # Use LLM for ambiguous files

# File renaming
rename:
  naming_convention: kebab-case
  include_date: true
  date_format: "YYYY-MM-DD"
  date_position: prefix  # prefix or suffix
  max_length: 60
  preserve_extension: true

# Git integration
git:
  enabled: true
  auto_install_hooks: false
  commit_format: conventional
  include_scope: true
  include_body: true
  max_subject_length: 72
  max_body_width: 80
  include_diff_in_body: false

# Screenshot handling
screenshots:
  enabled: true
  watch_dir: ~/Desktop
  destination: ~/Pictures/Screenshots
  rename: true
  add_metadata_tags: true
  use_vision: true
  vision_provider: openai  # Override default for vision tasks

# Task queue
queue:
  max_concurrent: 2
  default_timeout_seconds: 30
  max_retries: 3
  retry_delay_seconds: 5

# Logging
logging:
  level: info  # debug, info, warn, error
  max_size_mb: 10
  max_files: 5
  include_timestamps: true

# Notifications
notifications:
  enabled: true
  sound: false
  show_previews: true
```

---

## 7. Dashboard Specification

### 7.1 Pages

**Home (/)**
- Daemon status (running/stopped, uptime)
- Quick stats: tasks today, files organized, commits generated
- Recent activity feed
- Quick action buttons

**Tasks (/tasks)**
- Task queue visualization
- Active tasks with progress
- History with filtering
- Undo capability for file operations

**Configuration (/config)**
- Visual config editor
- Provider status indicators
- Test connection buttons
- Category management for auto-file

**Logs (/logs)**
- Real-time log streaming
- Level filtering
- Search functionality
- Export capability

### 7.2 API Routes

```typescript
// app/api/daemon/route.ts
GET  /api/daemon/status
POST /api/daemon/start
POST /api/daemon/stop

// app/api/tasks/route.ts
GET  /api/tasks
POST /api/tasks
GET  /api/tasks/[id]
DELETE /api/tasks/[id]

// app/api/config/route.ts
GET  /api/config
PUT  /api/config
POST /api/config/validate

// app/api/logs/route.ts
GET  /api/logs
GET  /api/logs/stream  // SSE endpoint
```

---

## 8. Security Considerations

### 8.1 Data Privacy

- Local-first: Ollama as default provider keeps data on-device
- API keys stored in macOS Keychain (not plaintext config)
- File contents not logged, only metadata
- Clipboard content cleared from memory after processing

### 8.2 File System Safety

- Operations are logged and reversible (24-hour undo window)
- Dry-run mode available for all file operations
- Exclude system directories by default
- Confirmation required for bulk operations (>10 files)

### 8.3 Network Security

- Unix socket not accessible over network
- Dashboard binds to localhost by default
- API keys read from environment variables or Keychain
- TLS verification enabled for all API calls

---

## 9. Installation

### 9.1 Prerequisites

- macOS 13.0+ (Ventura or later)
- Node.js 20+ (for CLI)
- Go 1.22+ (for building daemon, optional if using binary)
- Ollama (optional, for local LLM)

### 9.2 Installation Script

```bash
#!/bin/bash
# install.sh

set -e

INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="$HOME/.config/bender"
DATA_DIR="$HOME/.local/share/bender"
LOG_DIR="/usr/local/var/log/bender"

# Create directories
mkdir -p "$CONFIG_DIR" "$DATA_DIR" "$LOG_DIR"

# Download and install daemon
curl -L "https://github.com/user/bender/releases/latest/download/benderd-darwin-arm64" \
  -o "$INSTALL_DIR/benderd"
chmod +x "$INSTALL_DIR/benderd"

# Install CLI
npm install -g @bender/cli

# Copy default config
if [ ! -f "$CONFIG_DIR/config.yaml" ]; then
  curl -L "https://raw.githubusercontent.com/user/bender/main/configs/default.yaml" \
    -o "$CONFIG_DIR/config.yaml"
fi

# Install LaunchAgent
bender install agent

echo "Bender installed successfully!"
echo "Start with: bender start"
echo "Configure: bender config"
```

---

## 10. Development Phases

### Phase 1: Foundation (Week 1-2)

- [ ] Go daemon scaffold with graceful shutdown
- [ ] Unix socket IPC with JSON-RPC
- [ ] Configuration loading and validation
- [ ] Basic CLI with start/stop/status
- [ ] LaunchAgent installation

### Phase 2: Core Features (Week 3-4)

- [ ] Ollama provider integration
- [ ] Clipboard monitoring (macOS Pasteboard)
- [ ] FSEvents file watcher
- [ ] Task queue with SQLite persistence
- [ ] Summarization feature end-to-end

### Phase 3: File Operations (Week 5-6)

- [ ] Auto-file classification and moving
- [ ] Intelligent file renaming
- [ ] Screenshot detection and tagging
- [ ] Undo/redo system

### Phase 4: Git Integration (Week 7)

- [ ] Git hook installation
- [ ] Diff parsing and analysis
- [ ] Commit message generation
- [ ] Interactive commit workflow

### Phase 5: Additional Providers (Week 8)

- [ ] OpenAI provider (with vision)
- [ ] Anthropic provider
- [ ] Provider fallback logic
- [ ] Rate limiting and cost tracking

### Phase 6: Dashboard (Week 9-10)

- [ ] Next.js app scaffold
- [ ] Real-time status page
- [ ] Task management UI
- [ ] Visual configuration editor
- [ ] Log viewer with streaming

### Phase 7: Polish (Week 11-12)

- [ ] Error handling improvements
- [ ] Performance optimization
- [ ] Documentation
- [ ] Integration tests
- [ ] Release automation

---

## 11. Success Metrics

| Metric | Target |
|--------|--------|
| Daemon memory usage | < 50MB idle |
| Task latency (local LLM) | < 5s for summarization |
| Task latency (API) | < 10s for summarization |
| File classification accuracy | > 90% correct |
| Commit message acceptance rate | > 70% without edits |
| Crash-free uptime | > 99.9% |

---

## 12. Future Considerations

- **Plugin system:** Allow custom task types via JavaScript plugins
- **Multi-device sync:** Sync configuration and history via iCloud
- **iOS companion:** Shortcuts integration for mobile
- **Linux support:** systemd daemon for Linux desktop
- **Team features:** Shared configurations and custom prompts

---

## Appendix A: Go Module Dependencies

```go
// go.mod
module github.com/user/bender

go 1.22

require (
    github.com/fsnotify/fsevents v0.2.0
    github.com/mattn/go-sqlite3 v1.14.22
    github.com/spf13/viper v1.18.2
    golang.org/x/sys v0.17.0  // macOS clipboard
)
```

## Appendix B: CLI Package Dependencies

```json
{
  "name": "@bender/cli",
  "dependencies": {
    "commander": "^12.0.0",
    "chalk": "^5.3.0",
    "ora": "^8.0.0",
    "inquirer": "^9.2.0",
    "yaml": "^2.4.0",
    "ajv": "^8.12.0"
  }
}
```
