# Auto-File & Screenshot Pipelines

## Summary

Transform the existing "analyze-only" task handlers into full end-to-end automated pipelines:
- **Auto-file**: watch dir → settle → classify → move → rename → notify (with undo)
- **Screenshot**: watch dir → settle → tag (vision) → rename → move → notify (with undo)

Existing task handlers remain pure (input→output). A new `pipelines.go` orchestrates multi-step workflows.

---

## Phase 1: Foundation

### 1. Config changes — `daemon/internal/config/config.go`
Add to `AutoFileConfig`:
- `AutoMove bool` (yaml: `auto_move`) — whether to auto-move after classification
- `AutoRename bool` (yaml: `auto_rename`) — whether to auto-rename via LLM
- `SettleDelayMs int` (yaml: `settle_delay_ms`) — wait time before processing (default: 3000)

Add to `ScreenshotsConfig`:
- `SettleDelayMs int` (yaml: `settle_delay_ms`) — default: 2000

Add defaults in `setDefaults()`.

### 2. Config files — `configs/default.yaml`, `configs/schema.json`
Add the new fields with sensible defaults.

### 3. Task types & context — `daemon/internal/task/queue.go`
- Add constants: `TaskPipelineAutoFile = "pipeline.auto_file"`, `TaskPipelineScreenshot = "pipeline.screenshot"`
- Inject task ID into handler context: in `processTask()`, set `ctx = context.WithValue(ctx, taskIDKey, task.ID)` so pipeline handlers can read their own task ID for undo recording

---

## Phase 2: Core Pipelines

### 4. New file — `daemon/cmd/benderd/pipelines.go`

**`PipelineRunner` struct** holding: `router`, `cfg`, `undoMgr`, `notifier`

**`waitForSettle(ctx, path, delayMs)`** — waits for file size to stabilize (two consecutive size checks with `delayMs` gap). Returns error if file disappears or context cancels.

**`RunAutoFilePipeline(ctx, payload) (json.RawMessage, error)`**
1. Parse payload (`{path}`)
2. Extract task ID from context (for undo recording)
3. `waitForSettle` — skip in-progress downloads
4. Call `handleFileClassify()` directly (as Go function, not via queue)
5. If `auto_move` enabled & destination differs: `fileops.MoveFile()`, record undo
6. If `auto_rename` enabled: call `handleFileRename()`, then `fileops.RenameFile()`, record undo
7. Send notification with result summary
8. Return `{original_path, final_path, category, new_name, steps[]}`

**`RunScreenshotPipeline(ctx, payload) (json.RawMessage, error)`**
1. Parse payload (`{path}`)
2. Extract task ID from context
3. `waitForSettle`
4. If `use_vision`: call `handleScreenshotTag()` directly
5. If `rename` & suggested name available: `fileops.RenameFile()`, record undo
6. If `destination` set: `fileops.MoveFile()`, record undo
7. Send notification
8. Return `{original_path, final_path, app, description, tags[], steps[]}`

**Key design**: Pipeline handlers call existing handlers as Go functions (not re-queuing). Each pipeline IS a task handler itself, getting retry/timeout from the queue. Rename steps are non-fatal (best-effort).

### 5. Wire up — `daemon/cmd/benderd/main.go`

**Create PipelineRunner** after notifier init, pass to `registerTaskHandlers`.

**Register pipeline handlers:**
```go
queue.RegisterHandler(task.TaskPipelineAutoFile, pipelines.RunAutoFilePipeline)
queue.RegisterHandler(task.TaskPipelineScreenshot, pipelines.RunScreenshotPipeline)
```

**Update file watcher handler** — if `auto_move` is true, enqueue `pipeline.auto_file` instead of `file.classify`. Otherwise keep current classify-only behavior.

**Add screenshot watcher** — second `fswatch.Watcher` instance watching `cfg.Screenshots.WatchDir`. Filters by image extensions (`.png`, `.jpg`, `.jpeg`, `.gif`, `.webp`). On EventCreate, enqueues `pipeline.screenshot`.

**Add API handlers:**
- `pipeline.status` — returns enabled/disabled state, config for both pipelines
- `pipeline.auto_file` — sync execution via `EnqueueAndWait` (for CLI)
- `pipeline.screenshot` — sync execution via `EnqueueAndWait` (for CLI)

**Edge case**: If screenshot `watch_dir` overlaps with auto-file `watch_dirs`, skip image files in the auto-file handler to avoid double-processing.

**Timeout**: Pipeline tasks involve LLM calls + file I/O. Increase timeout for pipeline tasks or document that `queue.default_timeout_seconds` should be >= 60 when pipelines are active.

---

## Phase 3: CLI

### 6. New file — `cli/src/commands/pipeline.ts`
- `pipelineStatus()` — calls `pipeline.status`, displays both pipeline states
- `pipelineRun(type, file)` — calls `pipeline.auto_file` or `pipeline.screenshot` with spinner

### 7. Register commands — `cli/src/index.ts`
```
bender pipeline status        — show pipeline status
bender pipeline run auto-file <file>  — manually trigger auto-file pipeline
bender pipeline run screenshot <file> — manually trigger screenshot pipeline
```

---

## Phase 4: Dashboard

### 8. API route — `dashboard/app/api/pipelines/route.ts` (new)
GET handler proxying to `pipeline.status`.

### 9. Pipelines page — `dashboard/app/pipelines/page.tsx` (new)
- Two status cards (auto-file, screenshot) showing enabled/config
- Recent pipeline activity list (filtered from task queue by `pipeline.*` types)
- Undo buttons for completed pipeline tasks
- Auto-refresh every 5s (matching existing tasks page pattern)

### 10. Nav link — `dashboard/app/layout.tsx`
Add "Pipelines" link between Tasks and Config.

### 11. Config page — `dashboard/app/config/page.tsx`
Add toggles for `auto_move`, `auto_rename`, number input for `settle_delay_ms` in Auto File section. Add `settle_delay_ms` in Screenshots section.

### 12. Tasks page — `dashboard/app/tasks/page.tsx`
Update `isFileOperation` check to include `pipeline.auto_file` and `pipeline.screenshot` so undo button appears for pipeline tasks.

---

## Phase 5: Tests

### 13. `daemon/cmd/benderd/pipelines_test.go` (new)
- `TestWaitForSettle` — happy path with stable file
- `TestWaitForSettleMissingFile` — file doesn't exist
- `TestWaitForSettleContextCancel` — cancelled context

### 14. `daemon/internal/task/queue_test.go`
- Test new task type constants exist
- Test context value injection in processTask

### 15. `cli/src/__tests__/commands.test.ts`
- Test `pipeline.status` RPC call and response shape
- Test `pipeline.auto_file` RPC call with path param
- Test `pipeline.screenshot` RPC call with path param

---

## Files Changed

| File | Action | What |
|------|--------|------|
| `daemon/internal/config/config.go` | modify | Add AutoMove, AutoRename, SettleDelayMs fields + defaults |
| `daemon/internal/task/queue.go` | modify | Add 2 task type constants, inject task ID into context |
| `daemon/cmd/benderd/pipelines.go` | **create** | PipelineRunner, both pipeline handlers, waitForSettle |
| `daemon/cmd/benderd/main.go` | modify | Create PipelineRunner, register handlers, update watcher, add screenshot watcher, add 3 API handlers |
| `configs/default.yaml` | modify | Add new config fields |
| `configs/schema.json` | modify | Add schema for new fields |
| `cli/src/commands/pipeline.ts` | **create** | pipelineStatus, pipelineRun |
| `cli/src/index.ts` | modify | Register pipeline subcommand |
| `dashboard/app/api/pipelines/route.ts` | **create** | GET proxy to pipeline.status |
| `dashboard/app/pipelines/page.tsx` | **create** | Pipeline status + activity page |
| `dashboard/app/layout.tsx` | modify | Add Pipelines nav link |
| `dashboard/app/config/page.tsx` | modify | Add new config toggles |
| `dashboard/app/tasks/page.tsx` | modify | Undo support for pipeline types |
| `daemon/cmd/benderd/pipelines_test.go` | **create** | waitForSettle tests |
| `cli/src/__tests__/commands.test.ts` | modify | Pipeline RPC tests |

**No changes**: `handlers.go` (stays pure), `fswatch/watcher.go` (reused as-is), `fileops/` (reused as-is), `notify/` (reused as-is)

---

## Verification

1. **Build**: `cd daemon && go build ./...`
2. **Go tests**: `cd daemon && go test ./...`
3. **CLI tests**: `cd cli && npx vitest run`
4. **Manual e2e auto-file**: Start daemon, drop a `.pdf` into `~/Downloads`, verify it gets classified, moved to `~/Documents/Downloads/`, and renamed
5. **Manual e2e screenshot**: Take a macOS screenshot to `~/Desktop`, verify it gets tagged, renamed, and moved to `~/Pictures/Screenshots/`
6. **Undo**: Use `bender undo <task-id>` or dashboard to reverse pipeline operations
7. **Pipeline status**: `bender pipeline status` shows both pipelines active
8. **Dashboard**: Visit `/pipelines` page, verify status cards and activity list
