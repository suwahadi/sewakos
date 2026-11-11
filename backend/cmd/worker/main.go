package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/suwahadi/sewakos/backend/internal/booking"
	"github.com/suwahadi/sewakos/backend/internal/config"
	"github.com/suwahadi/sewakos/backend/internal/email"
	"github.com/suwahadi/sewakos/backend/internal/midtrans"
	"github.com/suwahadi/sewakos/backend/internal/platform/postgres"
	"github.com/suwahadi/sewakos/backend/internal/platform/storage"
	"github.com/suwahadi/sewakos/backend/internal/worker"
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

	deps := worker.Deps{
		Booking: booking.Deps{
			DB:                pool,
			Midtrans:          midtrans.NewHTTPClient(cfg.MidtransServerKey, cfg.MidtransIsProduction, cfg.MidtransBaseURL),
			MidtransServerKey: cfg.MidtransServerKey,
			AppPublicURL:      cfg.AppPublicURL,
		},
		Email:   email.LogSender{Logger: logger},
		Storage: objectStore,

		OwnerReminderBeforeMinutes: cfg.OwnerReminderBeforeMinutes,
	}
	scheduler := &worker.Scheduler{}

	logger.Info("worker started")
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("worker shutting down")
			return
		case <-ticker.C:
			worker.RunJobs(ctx, deps, logger)
			scheduler.RunDue(ctx, deps, logger, time.Now())
		}
	}
}
