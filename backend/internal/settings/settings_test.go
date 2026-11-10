package settings

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
)

type stubStore struct {
	calls int
	value []byte
	err   error
}

func (s *stubStore) GetSetting(ctx context.Context, key string) ([]byte, error) {
	s.calls++
	return s.value, s.err
}

func TestAdminFeeCacheReadsOnceWithinTTL(t *testing.T) {
	store := &stubStore{value: []byte("2500")}
	cache := NewAdminFeeCache(store)

	for i := 0; i < 3; i++ {
		fee, err := cache.Get(context.Background())
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		if fee != 2500 {
			t.Errorf("Get() = %d, want 2500", fee)
		}
	}
	if store.calls != 1 {
		t.Errorf("store called %d times, want 1", store.calls)
	}
}

func TestAdminFeeCacheInvalidateForcesReread(t *testing.T) {
	store := &stubStore{value: []byte("0")}
	cache := NewAdminFeeCache(store)

	if _, err := cache.Get(context.Background()); err != nil {
		t.Fatalf("Get: %v", err)
	}
	cache.Invalidate()
	store.value = []byte("1000")
	fee, err := cache.Get(context.Background())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if fee != 1000 {
		t.Errorf("Get() after Invalidate = %d, want 1000", fee)
	}
	if store.calls != 2 {
		t.Errorf("store called %d times, want 2", store.calls)
	}
}

func TestAdminFeeCacheMissingSettingDefaultsToZero(t *testing.T) {
	store := &stubStore{err: pgx.ErrNoRows}
	cache := NewAdminFeeCache(store)

	fee, err := cache.Get(context.Background())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if fee != 0 {
		t.Errorf("Get() = %d, want 0", fee)
	}
}

func TestAdminFeeCachePropagatesOtherErrors(t *testing.T) {
	store := &stubStore{err: errors.New("boom")}
	cache := NewAdminFeeCache(store)

	if _, err := cache.Get(context.Background()); err == nil {
		t.Fatal("Get() err = nil, want error")
	}
}
