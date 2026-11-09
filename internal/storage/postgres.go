// Package storage provides postgres-backed implementations of the
// repository interfaces defined by the domain packages (task.Repository,
// user.Repository), plus the shared connection-pool setup.
package storage

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"todoapp/internal/config"
	apperrors "todoapp/internal/errors"
	"todoapp/internal/task"
	"todoapp/internal/user"
)

// NewPool connects to the database at db.URL, applying the pool tuning
// from db.Pool. Zero values in the pool config keep the pgx defaults.
func NewPool(ctx context.Context, db config.Database) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(db.URL)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}
	if db.Pool.MaxConns > 0 {
		poolCfg.MaxConns = db.Pool.MaxConns
	}
	if db.Pool.MinConns > 0 {
		poolCfg.MinConns = db.Pool.MinConns
	}
	if db.Pool.MaxConnLifetime > 0 {
		poolCfg.MaxConnLifetime = db.Pool.MaxConnLifetime
	}
	if db.Pool.MaxConnIdleTime > 0 {
		poolCfg.MaxConnIdleTime = db.Pool.MaxConnIdleTime
	}
	return pgxpool.NewWithConfig(ctx, poolCfg)
}

// =========================================================================
// PostgresUserRepository
// =========================================================================

var _ user.Repository = (*PostgresUserRepository)(nil)

// PostgresUserRepository implements user.Repository on top of pgx.
type PostgresUserRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresUserRepository(pool *pgxpool.Pool) *PostgresUserRepository {
	return &PostgresUserRepository{pool: pool}
}

func (r *PostgresUserRepository) Init(ctx context.Context) error {
	const schema = `
	CREATE TABLE IF NOT EXISTS users (
		id            BIGSERIAL PRIMARY KEY,
		username      TEXT NOT NULL UNIQUE,
		password_hash TEXT NOT NULL,
		role          TEXT NOT NULL DEFAULT 'user',
		created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
	);`
	_, err := r.pool.Exec(ctx, schema)
	return err
}

func (r *PostgresUserRepository) Create(ctx context.Context, u *user.User) error {
	return r.pool.QueryRow(ctx,
		`INSERT INTO users (username, password_hash, role)
		 VALUES ($1, $2, $3)
		 RETURNING id, created_at`,
		u.Username, u.PasswordHash, u.Role,
	).Scan(&u.ID, &u.CreatedAt)
}

func (r *PostgresUserRepository) GetByUsername(ctx context.Context, username string) (*user.User, error) {
	u := &user.User{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, username, password_hash, role, created_at
		 FROM users WHERE username = $1`, username,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user %q: %w", username, err)
	}
	return u, nil
}

func (r *PostgresUserRepository) GetByID(ctx context.Context, id int64) (*user.User, error) {
	u := &user.User{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, username, password_hash, role, created_at
		 FROM users WHERE id = $1`, id,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user %d: %w", id, err)
	}
	return u, nil
}

// =========================================================================
// PostgresTaskRepository
// =========================================================================

// PostgresTaskRepository implements task.Repository on top of pgx.
type PostgresTaskRepository struct {
	pool *pgxpool.Pool
}

// Compile-time check: satisfies the domain persistence contract.
var _ task.Repository = (*PostgresTaskRepository)(nil)

func NewPostgresTaskRepository(pool *pgxpool.Pool) *PostgresTaskRepository {
	return &PostgresTaskRepository{pool: pool}
}

func (r *PostgresTaskRepository) Init(ctx context.Context) error {
	const schema = `
	CREATE TABLE IF NOT EXISTS users (
		id            BIGSERIAL PRIMARY KEY,
		username      TEXT NOT NULL UNIQUE,
		password_hash TEXT NOT NULL,
		role          TEXT NOT NULL DEFAULT 'user',
		created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
	);
	CREATE TABLE IF NOT EXISTS tasks (
		id          BIGSERIAL PRIMARY KEY,
		title       TEXT NOT NULL,
		description TEXT NOT NULL DEFAULT '',
		status      TEXT NOT NULL DEFAULT 'todo',
		created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
		updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
	);
	ALTER TABLE tasks ADD COLUMN IF NOT EXISTS user_id BIGINT NOT NULL DEFAULT 0;
	CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
	CREATE INDEX IF NOT EXISTS idx_tasks_user_id ON tasks(user_id);`
	_, err := r.pool.Exec(ctx, schema)
	return err
}

func (r *PostgresTaskRepository) Create(ctx context.Context, t *task.Task) error {
	if t.Status == "" {
		t.Status = task.StatusTodo
	}
	return r.pool.QueryRow(ctx,
		`INSERT INTO tasks (user_id, title, description, status)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, created_at, updated_at`,
		t.UserID, t.Title, t.Description, t.Status,
	).Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt)
}

func (r *PostgresTaskRepository) GetByID(ctx context.Context, id int64) (*task.Task, error) {
	t := &task.Task{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, title, description, status, created_at, updated_at
		 FROM tasks WHERE id = $1 AND status != $2`,
		id, task.StatusClose,
	).Scan(&t.ID, &t.UserID, &t.Title, &t.Description, &t.Status, &t.CreatedAt, &t.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperrors.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get task %d: %w", id, err)
	}
	return t, nil
}

func (r *PostgresTaskRepository) Update(ctx context.Context, t *task.Task) error {
	err := r.pool.QueryRow(ctx,
		`UPDATE tasks
		 SET title = $2, description = $3, status = $4, updated_at = now()
		 WHERE id = $1 AND status != $5
		 RETURNING created_at, updated_at`,
		t.ID, t.Title, t.Description, t.Status, task.StatusClose,
	).Scan(&t.CreatedAt, &t.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return apperrors.ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("update task %d: %w", t.ID, err)
	}
	return nil
}

func (r *PostgresTaskRepository) List(ctx context.Context, f task.TaskFilter) ([]task.Task, int, error) {
	where := "status != $1"
	args := []any{task.StatusClose}
	if f.Status != "" {
		args = append(args, f.Status)
		where = fmt.Sprintf("%s AND status = $%d", where, len(args))
	}
	if f.UserID != 0 {
		args = append(args, f.UserID)
		where = fmt.Sprintf("%s AND user_id = $%d", where, len(args))
	}

	var total int
	if err := r.pool.QueryRow(ctx,
		"SELECT count(*) FROM tasks WHERE "+where, args...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count tasks: %w", err)
	}

	offset := (f.Page - 1) * f.Limit
	args = append(args, f.Limit, offset)
	rows, err := r.pool.Query(ctx,
		fmt.Sprintf(
			"SELECT id, user_id, title, description, status, created_at, updated_at FROM tasks WHERE %s ORDER BY id LIMIT $%d OFFSET $%d",
			where, len(args)-1, len(args)),
		args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list tasks: %w", err)
	}
	defer rows.Close()

	tasks := make([]task.Task, 0, f.Limit)
	for rows.Next() {
		var t task.Task
		if err := rows.Scan(&t.ID, &t.UserID, &t.Title, &t.Description, &t.Status, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan task: %w", err)
		}
		tasks = append(tasks, t)
	}
	return tasks, total, rows.Err()
}
