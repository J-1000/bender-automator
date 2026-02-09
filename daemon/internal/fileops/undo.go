package fileops

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// OperationType represents the kind of file operation performed.
type OperationType string

const (
	OpMove   OperationType = "move"
	OpRename OperationType = "rename"
)

// Operation records a file operation for undo purposes.
type Operation struct {
	ID           string        `json:"id"`
	TaskID       string        `json:"task_id"`
	Type         OperationType `json:"type"`
	OriginalPath string        `json:"original_path"`
	NewPath      string        `json:"new_path"`
	CreatedAt    time.Time     `json:"created_at"`
}

// UndoManager tracks file operations and supports undoing them.
type UndoManager struct {
	db        *sql.DB
	retention time.Duration
}

// NewUndoManager creates a new undo manager backed by SQLite.
func NewUndoManager(dbPath string) (*UndoManager, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS file_operations (
		id TEXT PRIMARY KEY,
		task_id TEXT NOT NULL,
		type TEXT NOT NULL,
		original_path TEXT NOT NULL,
		new_path TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("create table: %w", err)
	}

	return &UndoManager{
		db:        db,
		retention: 24 * time.Hour,
	}, nil
}

// Record logs a file operation for later undo.
func (u *UndoManager) Record(op Operation) error {
	_, err := u.db.Exec(
		`INSERT INTO file_operations (id, task_id, type, original_path, new_path, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		op.ID, op.TaskID, op.Type, op.OriginalPath, op.NewPath, op.CreatedAt,
	)
	return err
}

// Undo reverses all operations associated with a task ID.
// Returns the number of operations undone.
func (u *UndoManager) Undo(taskID string) (int, error) {
	rows, err := u.db.Query(
		`SELECT id, original_path, new_path FROM file_operations WHERE task_id = ? ORDER BY created_at DESC`,
		taskID,
	)
	if err != nil {
		return 0, fmt.Errorf("query operations: %w", err)
	}
	defer rows.Close()

	undone := 0
	var ids []string
	for rows.Next() {
		var id, origPath, newPath string
		if err := rows.Scan(&id, &origPath, &newPath); err != nil {
			return undone, fmt.Errorf("scan row: %w", err)
		}

		// Move the file back to its original location
		if _, err := os.Stat(newPath); err == nil {
			if err := os.MkdirAll(filepath.Dir(origPath), 0755); err != nil {
				return undone, fmt.Errorf("create directory for undo: %w", err)
			}
			if err := os.Rename(newPath, origPath); err != nil {
				return undone, fmt.Errorf("undo move %s -> %s: %w", newPath, origPath, err)
			}
			undone++
		}
		ids = append(ids, id)
	}

	// Remove the undo records
	for _, id := range ids {
		u.db.Exec(`DELETE FROM file_operations WHERE id = ?`, id)
	}

	return undone, nil
}

// ListByTask returns all operations for a given task ID.
func (u *UndoManager) ListByTask(taskID string) ([]Operation, error) {
	rows, err := u.db.Query(
		`SELECT id, task_id, type, original_path, new_path, created_at FROM file_operations WHERE task_id = ? ORDER BY created_at DESC`,
		taskID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ops []Operation
	for rows.Next() {
		var op Operation
		if err := rows.Scan(&op.ID, &op.TaskID, &op.Type, &op.OriginalPath, &op.NewPath, &op.CreatedAt); err != nil {
			return nil, err
		}
		ops = append(ops, op)
	}
	return ops, nil
}

// Cleanup removes operations older than the retention window.
func (u *UndoManager) Cleanup() error {
	cutoff := time.Now().Add(-u.retention)
	_, err := u.db.Exec(`DELETE FROM file_operations WHERE created_at < ?`, cutoff)
	return err
}

// Close closes the underlying database.
func (u *UndoManager) Close() error {
	return u.db.Close()
}
