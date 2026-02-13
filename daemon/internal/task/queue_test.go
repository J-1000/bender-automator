package task

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"
)

func newTestQueue(t *testing.T) *Queue {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	q, err := NewQueue(Config{
		DBPath:      dbPath,
		MaxWorkers:  1,
		MaxRetries:  2,
		RetryDelay:  10 * time.Millisecond,
		TaskTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewQueue: %v", err)
	}
	return q
}

func TestEnqueue(t *testing.T) {
	q := newTestQueue(t)
	defer q.Stop()

	payload := json.RawMessage(`{"content":"test"}`)
	task, err := q.Enqueue(TaskClipboardSummarize, payload, 0)
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	if task.ID == "" {
		t.Error("task ID should not be empty")
	}
	if task.Type != TaskClipboardSummarize {
		t.Errorf("expected type %s, got %s", TaskClipboardSummarize, task.Type)
	}
	if task.Status != StatusPending {
		t.Errorf("expected status %s, got %s", StatusPending, task.Status)
	}
}

func TestGetTask(t *testing.T) {
	q := newTestQueue(t)
	defer q.Stop()

	payload := json.RawMessage(`{"path":"/test"}`)
	enqueued, _ := q.Enqueue(TaskFileClassify, payload, 0)

	fetched, err := q.GetTask(enqueued.ID)
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}
	if fetched == nil {
		t.Fatal("task not found")
	}
	if fetched.ID != enqueued.ID {
		t.Errorf("expected ID %s, got %s", enqueued.ID, fetched.ID)
	}
}

func TestGetTaskNotFound(t *testing.T) {
	q := newTestQueue(t)
	defer q.Stop()

	task, err := q.GetTask("nonexistent")
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}
	if task != nil {
		t.Error("expected nil for nonexistent task")
	}
}

func TestListTasks(t *testing.T) {
	q := newTestQueue(t)
	defer q.Stop()

	q.Enqueue(TaskClipboardSummarize, json.RawMessage(`{}`), 0)
	q.Enqueue(TaskFileClassify, json.RawMessage(`{}`), 0)
	q.Enqueue(TaskGitCommit, json.RawMessage(`{}`), 0)

	tasks, err := q.ListTasks(10)
	if err != nil {
		t.Fatalf("ListTasks: %v", err)
	}
	if len(tasks) != 3 {
		t.Errorf("expected 3 tasks, got %d", len(tasks))
	}
}

func TestListTasksLimit(t *testing.T) {
	q := newTestQueue(t)
	defer q.Stop()

	for i := 0; i < 5; i++ {
		q.Enqueue(TaskClipboardSummarize, json.RawMessage(`{}`), 0)
	}

	tasks, err := q.ListTasks(2)
	if err != nil {
		t.Fatalf("ListTasks: %v", err)
	}
	if len(tasks) != 2 {
		t.Errorf("expected 2 tasks, got %d", len(tasks))
	}
}

func TestCancelTask(t *testing.T) {
	q := newTestQueue(t)
	defer q.Stop()

	task, _ := q.Enqueue(TaskClipboardSummarize, json.RawMessage(`{}`), 0)

	err := q.CancelTask(task.ID)
	if err != nil {
		t.Fatalf("CancelTask: %v", err)
	}

	fetched, _ := q.GetTask(task.ID)
	if fetched.Status != StatusFailed {
		t.Errorf("expected status %s, got %s", StatusFailed, fetched.Status)
	}
}

func TestCancelTaskNotFound(t *testing.T) {
	q := newTestQueue(t)
	defer q.Stop()

	err := q.CancelTask("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent task")
	}
}

func TestPipelineTaskTypes(t *testing.T) {
	if TaskPipelineAutoFile != "pipeline.auto_file" {
		t.Errorf("expected pipeline.auto_file, got %s", TaskPipelineAutoFile)
	}
	if TaskPipelineScreenshot != "pipeline.screenshot" {
		t.Errorf("expected pipeline.screenshot, got %s", TaskPipelineScreenshot)
	}
}

func TestTaskIDInContext(t *testing.T) {
	q := newTestQueue(t)

	var capturedID string
	q.RegisterHandler(TaskClipboardSummarize, func(ctx context.Context, payload json.RawMessage) (json.RawMessage, error) {
		capturedID = TaskIDFromContext(ctx)
		return json.RawMessage(`{"ok":true}`), nil
	})

	q.Start()
	defer q.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := q.EnqueueAndWait(ctx, TaskClipboardSummarize, json.RawMessage(`{}`), 0)
	if err != nil {
		t.Fatalf("EnqueueAndWait: %v", err)
	}
	if result.Status != StatusCompleted {
		t.Fatalf("expected completed, got %s", result.Status)
	}
	if capturedID == "" {
		t.Error("expected task ID in context, got empty string")
	}
	if capturedID != result.ID {
		t.Errorf("context task ID %q != result task ID %q", capturedID, result.ID)
	}
}

func TestTaskIDFromContextEmpty(t *testing.T) {
	ctx := context.Background()
	id := TaskIDFromContext(ctx)
	if id != "" {
		t.Errorf("expected empty string, got %q", id)
	}
}

func TestEnqueueAndWait(t *testing.T) {
	q := newTestQueue(t)

	q.RegisterHandler(TaskClipboardSummarize, func(ctx context.Context, payload json.RawMessage) (json.RawMessage, error) {
		return json.RawMessage(`{"summary":"done"}`), nil
	})

	q.Start()
	defer q.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := q.EnqueueAndWait(ctx, TaskClipboardSummarize, json.RawMessage(`{"content":"test"}`), 0)
	if err != nil {
		t.Fatalf("EnqueueAndWait: %v", err)
	}
	if result.Status != StatusCompleted {
		t.Errorf("expected completed, got %s", result.Status)
	}
}
