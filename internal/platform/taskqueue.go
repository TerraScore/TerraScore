package platform

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TaskHandler processes a claimed task. Return nil on success, error to mark as failed.
type TaskHandler func(ctx context.Context, taskType string, payload json.RawMessage) error

// TaskQueue polls PostgreSQL for durable background tasks.
type TaskQueue struct {
	db       *pgxpool.Pool
	logger   *slog.Logger
	handlers map[string]TaskHandler
	pollFreq time.Duration
}

// NewTaskQueue creates a task queue worker.
func NewTaskQueue(db *pgxpool.Pool, logger *slog.Logger) *TaskQueue {
	return &TaskQueue{
		db:       db,
		logger:   logger,
		handlers: make(map[string]TaskHandler),
		pollFreq: 5 * time.Second,
	}
}

// Register adds a handler for a task type.
func (tq *TaskQueue) Register(taskType string, handler TaskHandler) {
	tq.handlers[taskType] = handler
}

// Enqueue inserts a task into the queue.
func (tq *TaskQueue) Enqueue(ctx context.Context, taskType string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshalling task payload: %w", err)
	}

	_, err = tq.db.Exec(ctx,
		`INSERT INTO task_queue (task_type, payload, status, created_at)
		 VALUES ($1, $2, 'pending', NOW())`,
		taskType, data,
	)
	if err != nil {
		return fmt.Errorf("enqueueing task: %w", err)
	}

	return nil
}

// Start begins polling for tasks. Call in a goroutine.
func (tq *TaskQueue) Start(ctx context.Context) {
	ticker := time.NewTicker(tq.pollFreq)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			tq.poll(ctx)
		}
	}
}

func (tq *TaskQueue) poll(ctx context.Context) {
	// Claim one pending task atomically
	row := tq.db.QueryRow(ctx,
		`UPDATE task_queue
		 SET status = 'processing', started_at = NOW()
		 WHERE id = (
		   SELECT id FROM task_queue
		   WHERE status = 'pending'
		   ORDER BY created_at ASC
		   LIMIT 1
		   FOR UPDATE SKIP LOCKED
		 )
		 RETURNING id, task_type, payload`,
	)

	var id int64
	var taskType string
	var payload json.RawMessage
	if err := row.Scan(&id, &taskType, &payload); err != nil {
		return // no tasks available
	}

	handler, ok := tq.handlers[taskType]
	if !ok {
		tq.logger.Warn("no handler for task type", "type", taskType, "id", id)
		return
	}

	tq.logger.Info("processing task", "type", taskType, "id", id)

	if err := handler(ctx, taskType, payload); err != nil {
		tq.logger.Error("task failed", "type", taskType, "id", id, "error", err)
		tq.db.Exec(ctx,
			`UPDATE task_queue SET status = 'failed', error_message = $2, completed_at = NOW() WHERE id = $1`,
			id, err.Error(),
		)
		return
	}

	tq.db.Exec(ctx,
		`UPDATE task_queue SET status = 'completed', completed_at = NOW() WHERE id = $1`,
		id,
	)
	tq.logger.Info("task completed", "type", taskType, "id", id)
}
