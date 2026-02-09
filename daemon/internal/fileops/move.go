package fileops

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// MoveFile moves a file to a destination, handling conflicts by appending a numeric suffix.
// It creates destination directories as needed. Returns the actual destination path used.
func MoveFile(src, dst string) (string, error) {
	if _, err := os.Stat(src); err != nil {
		return "", fmt.Errorf("source not found: %w", err)
	}

	// Create destination directory
	dstDir := filepath.Dir(dst)
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return "", fmt.Errorf("create destination directory: %w", err)
	}

	// Resolve conflicts
	dst = resolveConflict(dst)

	if err := os.Rename(src, dst); err != nil {
		return "", fmt.Errorf("move file: %w", err)
	}

	return dst, nil
}

// RenameFile renames a file in place, preserving its directory.
// Handles conflicts by appending a numeric suffix.
func RenameFile(src, newName string) (string, error) {
	if _, err := os.Stat(src); err != nil {
		return "", fmt.Errorf("source not found: %w", err)
	}

	dir := filepath.Dir(src)
	dst := filepath.Join(dir, newName)

	dst = resolveConflict(dst)

	if err := os.Rename(src, dst); err != nil {
		return "", fmt.Errorf("rename file: %w", err)
	}

	return dst, nil
}

// resolveConflict appends -1, -2, etc. if the destination already exists.
func resolveConflict(dst string) string {
	if _, err := os.Stat(dst); os.IsNotExist(err) {
		return dst
	}

	ext := filepath.Ext(dst)
	base := strings.TrimSuffix(dst, ext)

	for i := 1; i < 1000; i++ {
		candidate := fmt.Sprintf("%s-%d%s", base, i, ext)
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}

	return dst
}
