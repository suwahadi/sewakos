// Package ratelimit implements the one fixed-window counter both rate
// limiters in SPEC-GO.md §5.1 need: "api" (60/minute per IP, wired as HTTP
// middleware in internal/server) and "otp" (2/minute per destination phone
// number, keyed and called directly by the OTP service once it exists, B2.1).
package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Result is one Allow check's outcome.
type Result struct {
	Allowed    bool
	Count      int64
	Remaining  int64
	RetryAfter time.Duration
}

// Allow increments the counter for key within the current fixed window and
// reports whether it's still under limit. Window should match the limiter's
// stated period (e.g. time.Minute); the window boundary is the Unix time
// divided by the window length, so every caller using the same key during the
// same window shares one counter.
func Allow(ctx context.Context, rdb *redis.Client, scope, key string, limit int64, window time.Duration) (Result, error) {
	windowStart := time.Now().Unix() / int64(window.Seconds())
	redisKey := fmt.Sprintf("ratelimit:%s:%s:%d", scope, key, windowStart)

	count, err := rdb.Incr(ctx, redisKey).Result()
	if err != nil {
		return Result{}, fmt.Errorf("increment rate limit counter: %w", err)
	}
	if count == 1 {
		rdb.Expire(ctx, redisKey, window)
	}

	remaining := limit - count
	if remaining < 0 {
		remaining = 0
	}
	retryAfter := window - time.Duration(time.Now().Unix()%int64(window.Seconds()))*time.Second

	return Result{
		Allowed:    count <= limit,
		Count:      count,
		Remaining:  remaining,
		RetryAfter: retryAfter,
	}, nil
}
