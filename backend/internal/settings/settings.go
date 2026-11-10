package settings

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
)

const adminFeeCacheTTL = 60 * time.Second

type Store interface {
	GetSetting(ctx context.Context, key string) ([]byte, error)
}

// A missing admin_fee row is treated as 0, not an error.
type AdminFeeCache struct {
	store Store

	mu    sync.Mutex
	value int64
	until time.Time
}

func NewAdminFeeCache(store Store) *AdminFeeCache {
	return &AdminFeeCache{store: store}
}

func (c *AdminFeeCache) Get(ctx context.Context) (int64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if time.Now().Before(c.until) {
		return c.value, nil
	}

	raw, err := c.store.GetSetting(ctx, "admin_fee")
	if errors.Is(err, pgx.ErrNoRows) {
		c.value, c.until = 0, time.Now().Add(adminFeeCacheTTL)
		return c.value, nil
	}
	if err != nil {
		return 0, fmt.Errorf("get admin_fee setting: %w", err)
	}

	var fee int64
	if err := json.Unmarshal(raw, &fee); err != nil {
		return 0, fmt.Errorf("decode admin_fee setting: %w", err)
	}

	c.value, c.until = fee, time.Now().Add(adminFeeCacheTTL)
	return c.value, nil
}

func (c *AdminFeeCache) Invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.until = time.Time{}
}
