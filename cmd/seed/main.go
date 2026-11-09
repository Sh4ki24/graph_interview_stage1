// Command seed populates the configured database with fake tasks belonging
// to the admin user (see config seeder.count). Run it via `make seed`.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"todoapp/internal/config"
	"todoapp/internal/storage"
	"todoapp/internal/task"
	"todoapp/internal/user"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load(config.Path())
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	if err != nil {
		logger.Warn("config load failed, using defaults", "error", err)
	}

	pool, err := storage.NewPool(ctx, cfg.Database)
	if err != nil {
		fmt.Fprintln(os.Stderr, "connect database:", err)
		os.Exit(1)
	}
	defer pool.Close()

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		fmt.Fprintln(os.Stderr, "ping database:", err)
		os.Exit(1)
	}

	taskRepo := storage.NewPostgresTaskRepository(pool)
	if err := taskRepo.Init(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "init schema:", err)
		os.Exit(1)
	}

	userRepo := storage.NewPostgresUserRepository(pool)
	if err := userRepo.Init(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "init users schema:", err)
		os.Exit(1)
	}

	// ensure the admin user exists so tasks have an owner
	adminID := ensureAdmin(ctx, userRepo, cfg, logger)

	n := cfg.Seeder.Count
	logger.Info("seeding tasks", "count", n, "user_id", adminID, "url", cfg.Database.URL)
	if err := task.Seed(ctx, taskRepo, adminID, n); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	logger.Info("seed complete", "count", n)
}

// ensureAdmin returns the ID of the admin user, creating it if needed.
func ensureAdmin(ctx context.Context, repo *storage.PostgresUserRepository, cfg config.Config, logger *slog.Logger) int64 {
	existing, err := repo.GetByUsername(ctx, cfg.Auth.AdminUsername)
	if err == nil && existing != nil {
		return existing.ID
	}
	hash, err := user.HashPassword(cfg.Auth.AdminPassword)
	if err != nil {
		logger.Error("hash admin password", "error", err)
		os.Exit(1)
	}
	admin := &user.User{Username: cfg.Auth.AdminUsername, PasswordHash: hash, Role: user.RoleAdmin}
	if err := repo.Create(ctx, admin); err != nil {
		logger.Error("create admin user", "error", err)
		os.Exit(1)
	}
	return admin.ID
}