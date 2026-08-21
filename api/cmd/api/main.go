// Task Tracker API server: config → logging → database → auth → storage →
// HTTP server with graceful shutdown. Modules are wired here via manual
// constructor injection — no DI framework.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/genkovich/task-tracker/api/internal/modules/auth"
	"github.com/genkovich/task-tracker/api/internal/modules/board"
	"github.com/genkovich/task-tracker/api/internal/modules/user"
	"github.com/genkovich/task-tracker/api/internal/platform/authmw"
	"github.com/genkovich/task-tracker/api/internal/platform/config"
	"github.com/genkovich/task-tracker/api/internal/platform/database"
	"github.com/genkovich/task-tracker/api/internal/platform/logging"
	"github.com/genkovich/task-tracker/api/internal/platform/storage"
	"github.com/genkovich/task-tracker/api/internal/server"
)

func main() {
	cfg := config.Load()

	logger := logging.New(cfg.LogLevel)
	slog.SetDefault(logger)

	var db *database.DB
	if cfg.DatabaseURL != "" {
		var err error
		db, err = database.New(context.Background(), cfg.DatabaseURL)
		if err != nil {
			slog.Error("failed to connect to database", "error", err)
			os.Exit(1)
		}
		defer db.Close()
		slog.Info("database connected")
	} else {
		slog.Warn("DATABASE_URL not set, running without database")
	}

	if cfg.JWTSecret == "" || cfg.GoogleClientID == "" || cfg.GoogleClientSecret == "" {
		slog.Error("missing required auth config: JWT_SECRET, GOOGLE_CLIENT_ID, GOOGLE_CLIENT_SECRET must be set")
		os.Exit(1)
	}

	authHandler := auth.New(db, auth.Config{
		JWTSecret:          cfg.JWTSecret,
		GoogleClientID:     cfg.GoogleClientID,
		GoogleClientSecret: cfg.GoogleClientSecret,
		GoogleRedirectURL:  cfg.GoogleRedirectURL,
		FrontendURL:        cfg.FrontendURL,
		AppEnv:             cfg.AppEnv,
	})

	tokenValidator := auth.NewTokenValidator(cfg.JWTSecret)
	authMW := authmw.Middleware(tokenValidator)

	var avatarStorage storage.ObjectStorage
	if cfg.S3AvatarBucket != "" {
		s3Storage, err := storage.NewS3(storage.S3Config{
			Bucket:        cfg.S3AvatarBucket,
			Region:        cfg.S3AvatarRegion,
			Endpoint:      cfg.S3AvatarEndpoint,
			PublicBaseURL: cfg.S3AvatarPublicBaseURL,
			AccessKeyID:   cfg.S3AvatarAccessKeyID,
			SecretKey:     cfg.S3AvatarSecretKey,
		})
		if err != nil {
			slog.Error("failed to init S3 avatar storage", "error", err)
			os.Exit(1)
		}
		avatarStorage = s3Storage
		slog.Info("avatar storage", "type", "s3", "endpoint", cfg.S3AvatarEndpoint, "bucket", cfg.S3AvatarBucket)
	} else {
		avatarStorage = storage.NewLocal("/var/www/files/avatars", "/files/avatars")
		slog.Info("avatar storage", "type", "local", "dir", "/var/www/files/avatars")
	}

	s := server.New(db, cfg.CORSAllowedOrigins, authMW,
		server.WithAppEnv(cfg.AppEnv),
		user.New(db, avatarStorage),
		authHandler,
		// board is deliberately unauthenticated (ADR-0001, no accounts): it
		// registers only public routes, plus its SSE streams via the
		// streaming group (no per-request timeout).
		board.New(db),
	)

	// No WriteTimeout: it is connection-wide and would kill long-lived SSE
	// streams (ADR-0002) at the mark no matter what the handler does.
	// Non-streaming routes get their per-route deadline from
	// middleware.Timeout inside server.New; slow-client reads are bounded by
	// ReadTimeout/ReadHeaderTimeout below.
	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           s.Handler(),
		ReadTimeout:       15 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		slog.Info("server starting", "port", cfg.Port, "env", cfg.AppEnv)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown error", "error", err)
	}
}
