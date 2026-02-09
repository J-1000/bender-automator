package task

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/user/bender/internal/logging"
)

// TaskType represents the type of task
type TaskType string

const (
	TaskClipboardSummarize TaskType = "clipboard.summarize"
	TaskFileClassify       TaskType = "file.classify"
	TaskFileRename         TaskType = "file.rename"
	TaskGitCommit          TaskType = "git.commit"
	TaskScreenshotTag      TaskType = "screenshot.tag"
)

// TaskStatus represents the status of a task
type TaskStatus string

const (
	StatusPending   TaskStatus = "pending"
	StatusRunning   TaskStatus = "running"
	StatusCompleted TaskStatus = "completed"
	StatusFailed    TaskStatus = "failed"
)

// Task represents a queued task
type Task struct {
	ID         string          `json:"id"`
	Type       TaskType        `json:"type"`
	Priority   int             `json:"priority"`
	Payload    json.RawMessage `json:"payload"`
	Status     TaskStatus      `json:"status"`
	Result     json.RawMessage `json:"result,omitempty"`
	Error      string          `json:"error,omitempty"`
	CreatedAt  time.Time       `json:"created_at"`
	StartedAt  *time.Time      `json:"started_at,omitempty"`
	FinishedAt *time.Time      `json:"finished_at,omitempty"`
	RetryCount int             `json:"retry_count"`
	MaxRetries int             `json:"max_retries"`
}

// Handler processes a task and returns a result
type Handler func(ctx context.Context, payload json.RawMessage) (json.RawMessage, error)

// Queue manages task execution
type Queue struct {
	db           *sql.DB
	handlers     map[TaskType]Handler
	tasks        chan *Task
	maxWorkers   int
	maxRetries   int
	retryDelay   time.Duration
	taskTimeout  time.Duration
	mu           sync.RWMutex
	wg           sync.WaitGroup
	ctx          context.Context
	cancel       context.CancelFunc
}

// Config for the task queue
type Config struct {
	DBPath        string
	MaxWorkers    int
	MaxRetries    int
	RetryDelay    time.Duration
	TaskTimeout   time.Duration
}

// NewQueue creates a new task queue
func NewQueue(cfg Config) (*Queue, error) {
	if cfg.MaxWorkers == 0 {
		cfg.MaxWorkers = 2
	}
	if cfg.MaxRetries == 0 {
		cfg.MaxRetries = 3
	}
	if cfg.RetryDelay == 0 {
		cfg.RetryDelay = 5 * time.Second
	}
	if cfg.TaskTimeout == 0 {
		cfg.TaskTimeout = 30 * time.Second
	}

	db, err := sql.Open("sqlite3", cfg.DBPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := initDB(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("init database: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	q := &Queue{
		db:          db,
		handlers:    make(map[TaskType]Handler),
		tasks:       make(chan *Task, 100),
		maxWorkers:  cfg.MaxWorkers,
		maxRetries:  cfg.MaxRetries,
		retryDelay:  cfg.RetryDelay,
		taskTimeout: cfg.TaskTimeout,
		ctx:         ctx,
		cancel:      cancel,
	}

	return q, nil
}

func initDB(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS tasks (
			id TEXT PRIMARY KEY,
			type TEXT NOT NULL,
			priority INTEGER DEFAULT 0,
			payload TEXT,
			status TEXT DEFAULT 'pending',
			result TEXT,
			error TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			started_at DATETIME,
			finished_at DATETIME,
			retry_count INTEGER DEFAULT 0,
			max_retries INTEGER DEFAULT 3
		);
		CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
		CREATE INDEX IF NOT EXISTS idx_tasks_created ON tasks(created_at);
	`)
	return err
}

// RegisterHandler registers a handler for a task type
func (q *Queue) RegisterHandler(taskType TaskType, handler Handler) {
	q.mu.Lock()
	q.handlers[taskType] = handler
	q.mu.Unlock()
}

// Start begins processing tasks
func (q *Queue) Start() error {
	// Restart any interrupted tasks
	if err := q.restartPending(); err != nil {
		logging.Warn("failed to restart pending tasks: %v", err)
	}

	// Start workers
	for i := 0; i < q.maxWorkers; i++ {
		q.wg.Add(1)
		go q.worker(i)
	}

	logging.Info("task queue started with %d workers", q.maxWorkers)
	return nil
}

// Stop halts task processing
func (q *Queue) Stop() error {
	q.cancel()
	close(q.tasks)
	q.wg.Wait()
	q.db.Close()
	logging.Info("task queue stopped")
	return nil
}

func (q *Queue) restartPending() error {
	rows, err := q.db.Query(`
		SELECT id, type, priority, payload, retry_count, max_retries
		FROM tasks
		WHERE status IN ('pending', 'running')
		ORDER BY priority DESC, created_at ASC
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var task Task
		if err := rows.Scan(&task.ID, &task.Type, &task.Priority, &task.Payload, &task.RetryCount, &task.MaxRetries); err != nil {
			continue
		}
		select {
		case q.tasks <- &task:
		default:
			logging.Warn("task queue full, dropping task %s", task.ID)
		}
	}
	return nil
}

func (q *Queue) worker(id int) {
	defer q.wg.Done()

	for task := range q.tasks {
		select {
		case <-q.ctx.Done():
			return
		default:
		}

		q.processTask(task)
	}
}

func (q *Queue) processTask(task *Task) {
	q.mu.RLock()
	handler, ok := q.handlers[task.Type]
	q.mu.RUnlock()

	if !ok {
		q.failTask(task, fmt.Errorf("no handler for task type: %s", task.Type))
		return
	}

	// Update status to running
	now := time.Now()
	task.Status = StatusRunning
	task.StartedAt = &now
	q.updateTask(task)

	// Execute with timeout
	ctx, cancel := context.WithTimeout(q.ctx, q.taskTimeout)
	defer cancel()

	result, err := handler(ctx, task.Payload)
	finishedAt := time.Now()
	task.FinishedAt = &finishedAt

	if err != nil {
		if task.RetryCount < task.MaxRetries {
			task.RetryCount++
			task.Status = StatusPending
			q.updateTask(task)

			// Re-queue after delay
			go func() {
				time.Sleep(q.retryDelay)
				select {
				case q.tasks <- task:
				case <-q.ctx.Done():
				}
			}()
			return
		}
		q.failTask(task, err)
		return
	}

	task.Status = StatusCompleted
	task.Result = result
	q.updateTask(task)
	logging.Debug("task %s completed", task.ID)
}

func (q *Queue) failTask(task *Task, err error) {
	task.Status = StatusFailed
	task.Error = err.Error()
	now := time.Now()
	task.FinishedAt = &now
	q.updateTask(task)
	logging.Error("task %s failed: %v", task.ID, err)
}

func (q *Queue) updateTask(task *Task) {
	_, err := q.db.Exec(`
		UPDATE tasks SET
			status = ?,
			result = ?,
			error = ?,
			started_at = ?,
			finished_at = ?,
			retry_count = ?
		WHERE id = ?
	`, task.Status, task.Result, task.Error, task.StartedAt, task.FinishedAt, task.RetryCount, task.ID)
	if err != nil {
		logging.Error("failed to update task %s: %v", task.ID, err)
	}
}

// Enqueue adds a new task to the queue
func (q *Queue) Enqueue(taskType TaskType, payload json.RawMessage, priority int) (*Task, error) {
	id := generateID()
	task := &Task{
		ID:         id,
		Type:       taskType,
		Priority:   priority,
		Payload:    payload,
		Status:     StatusPending,
		MaxRetries: q.maxRetries,
		CreatedAt:  time.Now(),
	}

	_, err := q.db.Exec(`
		INSERT INTO tasks (id, type, priority, payload, status, max_retries, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, task.ID, task.Type, task.Priority, task.Payload, task.Status, task.MaxRetries, task.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert task: %w", err)
	}

	select {
	case q.tasks <- task:
	default:
		return nil, fmt.Errorf("task queue full")
	}

	logging.Debug("task %s enqueued: %s", task.ID, task.Type)
	return task, nil
}

// GetTask retrieves a task by ID
func (q *Queue) GetTask(id string) (*Task, error) {
	var task Task
	err := q.db.QueryRow(`
		SELECT id, type, priority, payload, status, result, error, created_at, started_at, finished_at, retry_count, max_retries
		FROM tasks WHERE id = ?
	`, id).Scan(&task.ID, &task.Type, &task.Priority, &task.Payload, &task.Status, &task.Result, &task.Error, &task.CreatedAt, &task.StartedAt, &task.FinishedAt, &task.RetryCount, &task.MaxRetries)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &task, nil
}

// ListTasks returns recent tasks
func (q *Queue) ListTasks(limit int) ([]*Task, error) {
	if limit == 0 {
		limit = 100
	}

	rows, err := q.db.Query(`
		SELECT id, type, priority, payload, status, result, error, created_at, started_at, finished_at, retry_count, max_retries
		FROM tasks
		ORDER BY created_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*Task
	for rows.Next() {
		var task Task
		if err := rows.Scan(&task.ID, &task.Type, &task.Priority, &task.Payload, &task.Status, &task.Result, &task.Error, &task.CreatedAt, &task.StartedAt, &task.FinishedAt, &task.RetryCount, &task.MaxRetries); err != nil {
			continue
		}
		tasks = append(tasks, &task)
	}
	return tasks, nil
}

// EnqueueAndWait adds a task and blocks until it completes or the context is cancelled.
func (q *Queue) EnqueueAndWait(ctx context.Context, taskType TaskType, payload json.RawMessage, priority int) (*Task, error) {
	t, err := q.Enqueue(taskType, payload, priority)
	if err != nil {
		return nil, err
	}

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			current, err := q.GetTask(t.ID)
			if err != nil {
				return nil, err
			}
			if current == nil {
				return nil, fmt.Errorf("task %s not found", t.ID)
			}
			switch current.Status {
			case StatusCompleted:
				return current, nil
			case StatusFailed:
				return current, fmt.Errorf("task failed: %s", current.Error)
			}
		}
	}
}

// CancelTask cancels a pending task.
func (q *Queue) CancelTask(id string) error {
	result, err := q.db.Exec(`
		UPDATE tasks SET status = 'failed', error = 'cancelled by user', finished_at = ?
		WHERE id = ? AND status IN ('pending', 'running')
	`, time.Now(), id)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("task %s not found or already completed", id)
	}
	return nil
}

func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
