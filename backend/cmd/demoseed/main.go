package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/suwahadi/sewakos/backend/internal/config"
	"github.com/suwahadi/sewakos/backend/internal/demoseed"
	"github.com/suwahadi/sewakos/backend/internal/media"
	"github.com/suwahadi/sewakos/backend/internal/platform/postgres"
	"github.com/suwahadi/sewakos/backend/internal/platform/storage"
)

func main() {
	count := flag.Int("count", 100, "number of demo properties to ensure exist")
	flag.Parse()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("load config", "error", err)
		os.Exit(1)
	}
	ctx := context.Background()
	pool, err := postgres.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("connect postgres", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

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

	created, err := demoseed.Run(ctx, demoseed.Deps{
		DB: pool, Storage: objectStore, Images: images, Logger: logger,
		HTTP: &http.Client{Timeout: time.Minute}, PhotoBaseURL: "https://images.unsplash.com",
	}, demoseed.Options{Count: *count, SlugPrefix: "demo-kost"})
	if err != nil {
		logger.Error("demo seed", "created", created, "error", err)
		os.Exit(1)
	}
	logger.Info("demo seed complete", "created", created)
}
