package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWaitForSettle(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	os.WriteFile(path, []byte("stable content"), 0644)

	ctx := context.Background()
	err := waitForSettle(ctx, path, 100)
	if err != nil {
		t.Fatalf("waitForSettle: %v", err)
	}
}

func TestWaitForSettleMissingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nonexistent.txt")

	ctx := context.Background()
	err := waitForSettle(ctx, path, 100)
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestWaitForSettleContextCancel(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	os.WriteFile(path, []byte("content"), 0644)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := waitForSettle(ctx, path, 5000)
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestWaitForSettleFileDisappears(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "vanish.txt")
	os.WriteFile(path, []byte("temp"), 0644)

	ctx := context.Background()

	// Remove the file after a brief delay (before second stat)
	go func() {
		time.Sleep(50 * time.Millisecond)
		os.Remove(path)
	}()

	err := waitForSettle(ctx, path, 100)
	if err == nil {
		t.Fatal("expected error when file disappears")
	}
}

func TestIsImageExtension(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"photo.png", true},
		{"photo.PNG", true},
		{"image.jpg", true},
		{"image.jpeg", true},
		{"anim.gif", true},
		{"modern.webp", true},
		{"doc.pdf", false},
		{"code.go", false},
		{"archive.zip", false},
		{"noext", false},
	}

	for _, tt := range tests {
		got := isImageExtension(tt.path)
		if got != tt.want {
			t.Errorf("isImageExtension(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}
