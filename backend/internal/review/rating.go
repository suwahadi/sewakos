package review

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/suwahadi/sewakos/backend/internal/platform/postgres/dbgen"
)

const ratingCacheTTL = 5 * time.Minute

type Rating struct {
	Average float64 `json:"average"`
	Count   int64   `json:"count"`
}

type RatingStore interface {
	GetPropertyRating(ctx context.Context, propertyID int64) (dbgen.GetPropertyRatingRow, error)
}

type RatingCache struct {
	rdb *redis.Client
}

func NewRatingCache(rdb *redis.Client) *RatingCache {
	return &RatingCache{rdb: rdb}
}

func ratingKey(propertyID int64) string {
	return fmt.Sprintf("rating:property:%d", propertyID)
}

func (c *RatingCache) Get(ctx context.Context, store RatingStore, propertyID int64) (Rating, error) {
	if cached, err := c.rdb.Get(ctx, ratingKey(propertyID)).Bytes(); err == nil {
		var rating Rating
		if json.Unmarshal(cached, &rating) == nil {
			return rating, nil
		}
	}
	row, err := store.GetPropertyRating(ctx, propertyID)
	if err != nil {
		return Rating{}, fmt.Errorf("get property rating %d: %w", propertyID, err)
	}
	rating := Rating{Average: row.Average, Count: row.Count}
	if encoded, err := json.Marshal(rating); err == nil {
		_ = c.rdb.Set(ctx, ratingKey(propertyID), encoded, ratingCacheTTL).Err()
	}
	return rating, nil
}

func (c *RatingCache) Invalidate(ctx context.Context, propertyID int64) {
	_ = c.rdb.Del(ctx, ratingKey(propertyID)).Err()
}

func (r Rating) Body() map[string]any {
	var average any
	if r.Count > 0 {
		average = r.Average
	}
	return map[string]any{"average": average, "count": r.Count}
}
