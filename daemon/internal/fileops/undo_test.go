package fileops

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestUndoManagerRecordAndUndo(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	mgr, err := NewUndoManager(dbPath)
	if err != nil {
		t.Fatalf("NewUndoManager: %v", err)
	}
	defer mgr.Close()

	// Create a file at "new" location to undo
	origPath := filepath.Join(dir, "original.txt")
	newPath := filepath.Join(dir, "moved.txt")
	os.WriteFile(newPath, []byte("content"), 0644)

	// Record the operation
	err = mgr.Record(Operation{
		ID:           "op1",
		TaskID:       "task1",
		Type:         OpMove,
		OriginalPath: origPath,
		NewPath:      newPath,
		CreatedAt:    time.Now(),
	})
	if err != nil {
		t.Fatalf("Record: %v", err)
	}

	// Undo
	count, err := mgr.Undo("task1")
	if err != nil {
		t.Fatalf("Undo: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 undone, got %d", count)
	}

	// File should be back at original path
	if _, err := os.Stat(origPath); err != nil {
		t.Error("original file not restored")
	}
	if _, err := os.Stat(newPath); !os.IsNotExist(err) {
		t.Error("moved file still exists")
	}
}

func TestUndoManagerListByTask(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	mgr, err := NewUndoManager(dbPath)
	if err != nil {
		t.Fatalf("NewUndoManager: %v", err)
	}
	defer mgr.Close()

	mgr.Record(Operation{
		ID: "op1", TaskID: "task1", Type: OpMove,
		OriginalPath: "/a", NewPath: "/b", CreatedAt: time.Now(),
	})
	mgr.Record(Operation{
		ID: "op2", TaskID: "task1", Type: OpRename,
		OriginalPath: "/c", NewPath: "/d", CreatedAt: time.Now(),
	})
	mgr.Record(Operation{
		ID: "op3", TaskID: "task2", Type: OpMove,
		OriginalPath: "/e", NewPath: "/f", CreatedAt: time.Now(),
	})

	ops, err := mgr.ListByTask("task1")
	if err != nil {
		t.Fatalf("ListByTask: %v", err)
	}
	if len(ops) != 2 {
		t.Errorf("expected 2 operations, got %d", len(ops))
	}
}

func TestUndoManagerUndoNoOps(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	mgr, err := NewUndoManager(dbPath)
	if err != nil {
		t.Fatalf("NewUndoManager: %v", err)
	}
	defer mgr.Close()

	count, err := mgr.Undo("nonexistent")
	if err != nil {
		t.Fatalf("Undo: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 undone, got %d", count)
	}
}
