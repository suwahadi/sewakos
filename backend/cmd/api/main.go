// Command api serves the HTTP API (SPEC-GO.md §3).
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/suwahadi/sewakos/backend/internal/config"
	"github.com/suwahadi/sewakos/backend/internal/media"
	"github.com/suwahadi/sewakos/backend/internal/platform/postgres"
	"github.com/suwahadi/sewakos/backend/internal/platform/storage"
	"github.com/suwahadi/sewakos/backend/internal/server"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("load config", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	pool, err := postgres.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("connect postgres", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	redisOptions, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		logger.Error("parse REDIS_URL", "error", err)
		os.Exit(1)
	}
	redisClient := redis.NewClient(redisOptions)
	defer redisClient.Close()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		logger.Error("connect redis", "error", err)
		os.Exit(1)
	}

	objectStore, err := storage.New(storage.Config{
		Endpoint: cfg.StorageEndpoint, Region: cfg.StorageRegion, Bucket: cfg.StorageBucket,
		AccessKey: cfg.StorageAccessKey, SecretKey: cfg.StorageSecretKey, UseSSL: cfg.StorageUseSSL,
	})
	if err != nil {
		logger.Error("create storage client", "error", err)
		os.Exit(1)
	}
	images := media.NewProcessor(media.Config{
		MaxPixels: cfg.ImageMaxPixels, LargePx: cfg.ImageLargePx, ThumbPx: cfg.ImageThumbPx,
		LargeQuality: cfg.ImageLargeQuality, ThumbQuality: cfg.ImageThumbQuality, Concurrency: cfg.ImageMaxConcurrency,
	})

	router := server.NewRouter(server.Deps{
		Config: cfg, DB: pool, Redis: redisClient, Logger: logger, Storage: objectStore, Images: images,
	})
	httpServer := &http.Server{Addr: cfg.HTTPAddr, Handler: router}

	go func() {
		logger.Info("api listening", "addr", cfg.HTTPAddr)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("serve", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	logger.Info("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown", "error", err)
	}
}
