package fileops

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMoveFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "test.txt")
	os.WriteFile(src, []byte("hello"), 0644)

	dstDir := filepath.Join(dir, "dest")
	dst := filepath.Join(dstDir, "test.txt")

	actual, err := MoveFile(src, dst)
	if err != nil {
		t.Fatalf("MoveFile: %v", err)
	}
	if actual != dst {
		t.Errorf("expected %s, got %s", dst, actual)
	}

	// Source should not exist
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Error("source file still exists")
	}

	// Destination should exist
	data, err := os.ReadFile(actual)
	if err != nil {
		t.Fatalf("read dest: %v", err)
	}
	if string(data) != "hello" {
		t.Errorf("expected 'hello', got %q", data)
	}
}

func TestMoveFileConflict(t *testing.T) {
	dir := t.TempDir()

	// Create source
	src := filepath.Join(dir, "file.txt")
	os.WriteFile(src, []byte("new"), 0644)

	// Create existing destination
	dst := filepath.Join(dir, "dest", "file.txt")
	os.MkdirAll(filepath.Dir(dst), 0755)
	os.WriteFile(dst, []byte("existing"), 0644)

	actual, err := MoveFile(src, dst)
	if err != nil {
		t.Fatalf("MoveFile: %v", err)
	}

	// Should have appended -1
	expected := filepath.Join(dir, "dest", "file-1.txt")
	if actual != expected {
		t.Errorf("expected %s, got %s", expected, actual)
	}
}

func TestRenameFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "old-name.txt")
	os.WriteFile(src, []byte("content"), 0644)

	actual, err := RenameFile(src, "new-name.txt")
	if err != nil {
		t.Fatalf("RenameFile: %v", err)
	}

	expected := filepath.Join(dir, "new-name.txt")
	if actual != expected {
		t.Errorf("expected %s, got %s", expected, actual)
	}

	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Error("old file still exists")
	}
}

func TestRenameFileConflict(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "a.txt")
	os.WriteFile(src, []byte("a"), 0644)
	os.WriteFile(filepath.Join(dir, "b.txt"), []byte("b"), 0644)

	actual, err := RenameFile(src, "b.txt")
	if err != nil {
		t.Fatalf("RenameFile: %v", err)
	}

	expected := filepath.Join(dir, "b-1.txt")
	if actual != expected {
		t.Errorf("expected %s, got %s", expected, actual)
	}
}

func TestResolveConflictNoConflict(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nonexistent.txt")
	result := resolveConflict(path)
	if result != path {
		t.Errorf("expected %s, got %s", path, result)
	}
}

func TestMoveFileSourceNotFound(t *testing.T) {
	dir := t.TempDir()
	_, err := MoveFile(filepath.Join(dir, "nonexistent"), filepath.Join(dir, "dest"))
	if err == nil {
		t.Error("expected error for missing source")
	}
}
