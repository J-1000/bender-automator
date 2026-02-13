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
