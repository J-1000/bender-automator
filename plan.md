# Auto-File & Screenshot Pipelines

## Summary

Transform the existing "analyze-only" task handlers into full end-to-end automated pipelines:
- **Auto-file**: watch dir → settle → classify → move → rename → notify (with undo)
- **Screenshot**: watch dir → settle → tag (vision) → rename → move → notify (with undo)

Existing task handlers remain pure (input→output). A new `pipelines.go` orchestrates multi-step workflows.
