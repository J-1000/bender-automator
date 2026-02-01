package git

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// Repo represents a git repository
type Repo struct {
	path string
}

// Open opens a git repository at the given path
func Open(path string) (*Repo, error) {
	// Find the git root
	cmd := exec.Command("git", "-C", path, "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("not a git repository: %w", err)
	}

	return &Repo{
		path: strings.TrimSpace(string(output)),
	}, nil
}

// Path returns the repository root path
func (r *Repo) Path() string {
	return r.path
}

// StagedDiff returns the diff of staged changes
func (r *Repo) StagedDiff(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", r.path, "diff", "--cached")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git diff: %w", err)
	}
	return string(output), nil
}

// StagedFiles returns the list of staged files
func (r *Repo) StagedFiles(ctx context.Context) ([]string, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", r.path, "diff", "--cached", "--name-only")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git diff --name-only: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var files []string
	for _, line := range lines {
		if line != "" {
			files = append(files, line)
		}
	}
	return files, nil
}

// StagedStats returns stats about staged changes
func (r *Repo) StagedStats(ctx context.Context) (*DiffStats, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", r.path, "diff", "--cached", "--stat")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git diff --stat: %w", err)
	}

	return parseDiffStats(string(output)), nil
}

// DiffStats contains statistics about a diff
type DiffStats struct {
	FilesChanged int
	Insertions   int
	Deletions    int
}

func parseDiffStats(stat string) *DiffStats {
	stats := &DiffStats{}
	lines := strings.Split(stat, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "file") && strings.Contains(line, "changed") {
			// Parse the summary line
			parts := strings.Fields(line)
			for i, part := range parts {
				if part == "file" || part == "files" {
					if i > 0 {
						fmt.Sscanf(parts[i-1], "%d", &stats.FilesChanged)
					}
				}
				if part == "insertions(+)" || part == "insertion(+)" {
					if i > 0 {
						fmt.Sscanf(parts[i-1], "%d", &stats.Insertions)
					}
				}
				if part == "deletions(-)" || part == "deletion(-)" {
					if i > 0 {
						fmt.Sscanf(parts[i-1], "%d", &stats.Deletions)
					}
				}
			}
		}
	}
	return stats
}

// Commit creates a commit with the given message
func (r *Repo) Commit(ctx context.Context, message string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", r.path, "commit", "-m", message)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git commit: %w", err)
	}

	// Extract commit hash from output
	lines := strings.Split(string(output), "\n")
	if len(lines) > 0 {
		parts := strings.Fields(lines[0])
		for _, part := range parts {
			if len(part) >= 7 && isHex(part) {
				return part, nil
			}
		}
	}
	return "", nil
}

func isHex(s string) bool {
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

// InstallHook installs a git hook
func (r *Repo) InstallHook(name string, content string) error {
	hookPath := filepath.Join(r.path, ".git", "hooks", name)
	cmd := exec.Command("bash", "-c", fmt.Sprintf("cat > %s && chmod +x %s", hookPath, hookPath))
	cmd.Stdin = strings.NewReader(content)
	return cmd.Run()
}

// RemoveHook removes a git hook
func (r *Repo) RemoveHook(name string) error {
	hookPath := filepath.Join(r.path, ".git", "hooks", name)
	cmd := exec.Command("rm", "-f", hookPath)
	return cmd.Run()
}

// HookExists checks if a hook exists
func (r *Repo) HookExists(name string) bool {
	hookPath := filepath.Join(r.path, ".git", "hooks", name)
	cmd := exec.Command("test", "-f", hookPath)
	return cmd.Run() == nil
}

// PrepareCommitMsgHook returns the content for prepare-commit-msg hook
func PrepareCommitMsgHook(socketPath string) string {
	return fmt.Sprintf(`#!/bin/bash
# Bender git hook - generate commit message
COMMIT_MSG_FILE=$1
COMMIT_SOURCE=$2

# Only generate for new commits (not merges, amends, etc.)
if [ -z "$COMMIT_SOURCE" ]; then
    GENERATED=$(echo '{"jsonrpc":"2.0","method":"git.generate_commit","id":1}' | nc -U %s 2>/dev/null | jq -r '.result.message // empty')
    if [ -n "$GENERATED" ]; then
        echo "$GENERATED" > "$COMMIT_MSG_FILE"
    fi
fi
`, socketPath)
}
