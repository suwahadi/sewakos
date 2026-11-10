package ratelimit

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/suwahadi/sewakos/backend/internal/response"
)

// EnforceHTTP checks Allow for scope/key and, if the limit is exceeded,
// writes the standard 429 envelope with the documented rate-limit headers
// (API.md §1.4-§1.5) and returns false — callers must return immediately in
// that case. Redis being unreachable fails open (true), matching every other
// limiter in this codebase (SPEC-GO.md availability over strict enforcement).
func EnforceHTTP(ctx context.Context, w http.ResponseWriter, rdb *redis.Client, scope, key string, limit int64, window time.Duration) bool {
	result, err := Allow(ctx, rdb, scope, key, limit, window)
	if err != nil {
		return true
	}
	if result.Allowed {
		return true
	}

	w.Header().Set("Retry-After", strconv.Itoa(int(result.RetryAfter.Seconds())))
	w.Header().Set("X-RateLimit-Limit", strconv.FormatInt(limit, 10))
	w.Header().Set("X-RateLimit-Remaining", strconv.FormatInt(result.Remaining, 10))
	response.Fail(w, http.StatusTooManyRequests, response.MsgRateLimited, nil)
	return false
}
