package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/user/bender/internal/config"
	"github.com/user/bender/internal/fileops"
	"github.com/user/bender/internal/llm"
	"github.com/user/bender/internal/logging"
	"github.com/user/bender/internal/notify"
	"github.com/user/bender/internal/task"
)

// PipelineRunner orchestrates multi-step file processing pipelines.
type PipelineRunner struct {
	router   *llm.Router
	cfg      *config.Config
	undoMgr  *fileops.UndoManager
	notifier *notify.Notifier
}

// NewPipelineRunner creates a new PipelineRunner.
func NewPipelineRunner(router *llm.Router, cfg *config.Config, undoMgr *fileops.UndoManager, notifier *notify.Notifier) *PipelineRunner {
	return &PipelineRunner{
		router:   router,
		cfg:      cfg,
		undoMgr:  undoMgr,
		notifier: notifier,
	}
}

// pipelineStep records a step executed during a pipeline.
type pipelineStep struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
}

// waitForSettle waits for a file's size to stabilize across two consecutive checks.
func waitForSettle(ctx context.Context, path string, delayMs int) error {
	if delayMs <= 0 {
		delayMs = 2000
	}
	delay := time.Duration(delayMs) * time.Millisecond

	info1, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("file not found: %w", err)
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(delay):
	}

	info2, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("file disappeared: %w", err)
	}

	if info1.Size() != info2.Size() {
		// Size changed — wait one more round
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}

		info3, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("file disappeared: %w", err)
		}
		if info2.Size() != info3.Size() {
			return fmt.Errorf("file still changing size")
		}
	}

	return nil
}

// autoFileResult is the JSON output of RunAutoFilePipeline.
type autoFileResult struct {
	OriginalPath string         `json:"original_path"`
	FinalPath    string         `json:"final_path"`
	Category     string         `json:"category"`
	NewName      string         `json:"new_name,omitempty"`
	Steps        []pipelineStep `json:"steps"`
}

// RunAutoFilePipeline classifies, moves, and renames a file end-to-end.
func (p *PipelineRunner) RunAutoFilePipeline(ctx context.Context, payload json.RawMessage) (json.RawMessage, error) {
	var params struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(payload, &params); err != nil {
		return nil, fmt.Errorf("parse payload: %w", err)
	}
	if params.Path == "" {
		return nil, fmt.Errorf("empty path")
	}

	taskID := task.TaskIDFromContext(ctx)
	currentPath := params.Path
	var steps []pipelineStep

	logging.Info("pipeline.auto_file: starting for %s", filepath.Base(currentPath))

	// 1. Settle
	if err := waitForSettle(ctx, currentPath, p.cfg.AutoFile.SettleDelayMs); err != nil {
		return nil, fmt.Errorf("settle: %w", err)
	}
	steps = append(steps, pipelineStep{Name: "settle", Status: "ok"})

	// 2. Classify
	classifyPayload, _ := json.Marshal(map[string]string{"path": currentPath})
	classifyRaw, err := handleFileClassify(ctx, classifyPayload, p.router, p.cfg)
	if err != nil {
		return nil, fmt.Errorf("classify: %w", err)
	}
	var cr classifyResult
	json.Unmarshal(classifyRaw, &cr)
	steps = append(steps, pipelineStep{Name: "classify", Status: "ok", Detail: cr.Category})

	// 3. Move (if enabled and destination differs)
	if p.cfg.AutoFile.AutoMove && cr.Destination != "" && cr.Destination != currentPath {
		actualDst, err := fileops.MoveFile(currentPath, cr.Destination)
		if err != nil {
			steps = append(steps, pipelineStep{Name: "move", Status: "error", Detail: err.Error()})
		} else {
			if taskID != "" {
				p.undoMgr.Record(fileops.Operation{
					ID:           fmt.Sprintf("%d", time.Now().UnixNano()),
					TaskID:       taskID,
					Type:         fileops.OpMove,
					OriginalPath: currentPath,
					NewPath:      actualDst,
					CreatedAt:    time.Now(),
				})
			}
			steps = append(steps, pipelineStep{Name: "move", Status: "ok", Detail: actualDst})
			currentPath = actualDst
		}
	}

	// 4. Rename (if enabled, best-effort)
	var newName string
	if p.cfg.AutoFile.AutoRename {
		renamePayload, _ := json.Marshal(map[string]string{"path": currentPath})
		renameRaw, err := handleFileRename(ctx, renamePayload, p.router, p.cfg)
		if err != nil {
			steps = append(steps, pipelineStep{Name: "rename", Status: "error", Detail: err.Error()})
		} else {
			var rr renameResult
			json.Unmarshal(renameRaw, &rr)
			if rr.NewName != "" && rr.NewName != filepath.Base(currentPath) {
				actualDst, err := fileops.RenameFile(currentPath, rr.NewName)
				if err != nil {
					steps = append(steps, pipelineStep{Name: "rename", Status: "error", Detail: err.Error()})
				} else {
					if taskID != "" {
						p.undoMgr.Record(fileops.Operation{
							ID:           fmt.Sprintf("%d", time.Now().UnixNano()),
							TaskID:       taskID,
							Type:         fileops.OpRename,
							OriginalPath: currentPath,
							NewPath:      actualDst,
							CreatedAt:    time.Now(),
						})
					}
					newName = rr.NewName
					steps = append(steps, pipelineStep{Name: "rename", Status: "ok", Detail: rr.NewName})
					currentPath = actualDst
				}
			}
		}
	}

	// 5. Notify
	p.notifier.SendWithSubtitle("Bender", "Auto-filed", fmt.Sprintf("%s → %s", filepath.Base(params.Path), cr.Category))

	result := autoFileResult{
		OriginalPath: params.Path,
		FinalPath:    currentPath,
		Category:     cr.Category,
		NewName:      newName,
		Steps:        steps,
	}

	logging.Info("pipeline.auto_file: completed %s → %s (%s)", filepath.Base(params.Path), cr.Category, currentPath)
	return json.Marshal(result)
}

// screenshotPipelineResult is the JSON output of RunScreenshotPipeline.
type screenshotPipelineResult struct {
	OriginalPath string         `json:"original_path"`
	FinalPath    string         `json:"final_path"`
	App          string         `json:"app"`
	Description  string         `json:"description"`
	Tags         []string       `json:"tags"`
	Steps        []pipelineStep `json:"steps"`
}

// RunScreenshotPipeline tags, renames, and moves a screenshot end-to-end.
func (p *PipelineRunner) RunScreenshotPipeline(ctx context.Context, payload json.RawMessage) (json.RawMessage, error) {
	var params struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(payload, &params); err != nil {
		return nil, fmt.Errorf("parse payload: %w", err)
	}
	if params.Path == "" {
		return nil, fmt.Errorf("empty path")
	}

	taskID := task.TaskIDFromContext(ctx)
	currentPath := params.Path
	var steps []pipelineStep

	logging.Info("pipeline.screenshot: starting for %s", filepath.Base(currentPath))

	// 1. Settle
	if err := waitForSettle(ctx, currentPath, p.cfg.Screenshots.SettleDelayMs); err != nil {
		return nil, fmt.Errorf("settle: %w", err)
	}
	steps = append(steps, pipelineStep{Name: "settle", Status: "ok"})

	// 2. Tag via vision
	var sr screenshotResult
	if p.cfg.Screenshots.UseVision {
		tagPayload, _ := json.Marshal(map[string]string{"path": currentPath})
		tagRaw, err := handleScreenshotTag(ctx, tagPayload, p.router, p.cfg)
		if err != nil {
			return nil, fmt.Errorf("tag: %w", err)
		}
		json.Unmarshal(tagRaw, &sr)
		steps = append(steps, pipelineStep{Name: "tag", Status: "ok", Detail: sr.Description})
	}

	// 3. Rename (if enabled and suggested name available, best-effort)
	if p.cfg.Screenshots.Rename && sr.SuggestedName != "" && sr.SuggestedName != filepath.Base(currentPath) {
		actualDst, err := fileops.RenameFile(currentPath, sr.SuggestedName)
		if err != nil {
			steps = append(steps, pipelineStep{Name: "rename", Status: "error", Detail: err.Error()})
		} else {
			if taskID != "" {
				p.undoMgr.Record(fileops.Operation{
					ID:           fmt.Sprintf("%d", time.Now().UnixNano()),
					TaskID:       taskID,
					Type:         fileops.OpRename,
					OriginalPath: currentPath,
					NewPath:      actualDst,
					CreatedAt:    time.Now(),
				})
			}
			steps = append(steps, pipelineStep{Name: "rename", Status: "ok", Detail: sr.SuggestedName})
			currentPath = actualDst
		}
	}

	// 4. Move to destination (if configured)
	if p.cfg.Screenshots.Destination != "" {
		dest := filepath.Join(p.cfg.Screenshots.Destination, filepath.Base(currentPath))
		if dest != currentPath {
			actualDst, err := fileops.MoveFile(currentPath, dest)
			if err != nil {
				steps = append(steps, pipelineStep{Name: "move", Status: "error", Detail: err.Error()})
			} else {
				if taskID != "" {
					p.undoMgr.Record(fileops.Operation{
						ID:           fmt.Sprintf("%d", time.Now().UnixNano()),
						TaskID:       taskID,
						Type:         fileops.OpMove,
						OriginalPath: currentPath,
						NewPath:      actualDst,
						CreatedAt:    time.Now(),
					})
				}
				steps = append(steps, pipelineStep{Name: "move", Status: "ok", Detail: actualDst})
				currentPath = actualDst
			}
		}
	}

	// 5. Notify
	desc := sr.Description
	if desc == "" {
		desc = filepath.Base(currentPath)
	}
	p.notifier.SendWithSubtitle("Bender", "Screenshot processed", desc)

	result := screenshotPipelineResult{
		OriginalPath: params.Path,
		FinalPath:    currentPath,
		App:          sr.App,
		Description:  sr.Description,
		Tags:         sr.Tags,
		Steps:        steps,
	}

	logging.Info("pipeline.screenshot: completed %s → %s", filepath.Base(params.Path), currentPath)
	return json.Marshal(result)
}

// isImageExtension returns true if the file has a common image extension.
func isImageExtension(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp":
		return true
	}
	return false
}
