// @title           Todo API
// @version         1.0
// @description     A todo list API with JWT authentication.
// @host            localhost:8080
// @BasePath        /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization

package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"todoapp/internal/auth"
	"todoapp/internal/config"
	"todoapp/internal/events"
	"todoapp/internal/routes"
	"todoapp/internal/storage"
	"todoapp/internal/task"
	"todoapp/internal/user"
)

// newLogger builds a leveled slog logger from the log config.
func newLogger(level string) *slog.Logger {
	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: lvl}))
}

// jwtSecret returns the signing key from JWT_SECRET. If none is set it falls
// back to a random ephemeral secret so the app still starts (tokens will not
// survive a restart). The secret is never stored in a config file.
func jwtSecret(logger *slog.Logger) string {
	if s := os.Getenv("JWT_SECRET"); s != "" {
		return s
	}
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		logger.Warn("could not generate random jwt secret", "error", err)
		return "ephemeral-dev-secret"
	}
	logger.Warn("JWT_SECRET not set; using an ephemeral random secret")
	return hex.EncodeToString(buf)
}

// seedAdmin ensures the single admin account (from config) exists.
func seedAdmin(ctx context.Context, repo *storage.PostgresUserRepository, cfg config.Config, logger *slog.Logger) {
	existing, err := repo.GetByUsername(ctx, cfg.Auth.AdminUsername)
	if err != nil {
		logger.Error("lookup admin user", "error", err)
		return
	}
	if existing != nil {
		return // admin already seeded
	}
	hash, err := user.HashPassword(cfg.Auth.AdminPassword)
	if err != nil {
		logger.Error("hash admin password", "error", err)
		return
	}
	admin := &user.User{
		Username:     cfg.Auth.AdminUsername,
		PasswordHash: hash,
		Role:         user.RoleAdmin,
	}
	if err := repo.Create(ctx, admin); err != nil {
		logger.Error("create admin user", "error", err)
		return
	}
	logger.Info("seeded admin user", "username", admin.Username)
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load(config.Path())
	logger := newLogger(config.Default().Log.Level)
	if err != nil {
		logger.Warn("config load failed, using defaults", "error", err)
	} else {
		logger = newLogger(cfg.Log.Level)
	}

	auth.JWTSecret = jwtSecret(logger)

	pool, err := storage.NewPool(ctx, cfg.Database)
	if err != nil {
		logger.Error("connect database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	connCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(connCtx); err != nil {
		logger.Error("ping database", "error", err)
		os.Exit(1)
	}

	repo := storage.NewPostgresTaskRepository(pool)
	if err := repo.Init(ctx); err != nil {
		logger.Error("init schema", "error", err)
		os.Exit(1)
	}

	userRepo := storage.NewPostgresUserRepository(pool)
	if err := userRepo.Init(ctx); err != nil {
		logger.Error("init users schema", "error", err)
		os.Exit(1)
	}
	seedAdmin(ctx, userRepo, cfg, logger)

	// async task events: service publishes, worker goroutine drains
	eventCh := make(chan events.Event, 64)
	events.StartWorker(ctx, eventCh, logger)

	svc := task.NewService(repo, eventCh, task.Options{
		DefaultLimit: cfg.Pagination.DefaultLimit,
		MaxLimit:     cfg.Pagination.MaxLimit,
	})
	srv := routes.New(svc, userRepo, cfg.Pagination)
	if err := routes.Run(ctx, cfg, logger, srv); err != nil {
		logger.Error("server", "error", err)
		os.Exit(1)
	}
}