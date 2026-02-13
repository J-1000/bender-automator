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
