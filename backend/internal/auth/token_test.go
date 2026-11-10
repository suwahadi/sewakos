package auth

import (
	"context"
	"testing"

	"github.com/suwahadi/sewakos/backend/internal/platform/postgres/dbgen"
)

type fakeStore struct {
	nextID  int64
	tokens  map[int64]dbgen.GetAccessTokenForVerificationRow
	touched map[int64]int
}

func newFakeStore() *fakeStore {
	return &fakeStore{tokens: map[int64]dbgen.GetAccessTokenForVerificationRow{}, touched: map[int64]int{}}
}

func (s *fakeStore) CreateAccessToken(_ context.Context, arg dbgen.CreateAccessTokenParams) (int64, error) {
	s.nextID++
	s.tokens[s.nextID] = dbgen.GetAccessTokenForVerificationRow{
		ID: s.nextID, TokenHash: arg.TokenHash, UserID: arg.UserID, Role: "tenant",
	}
	return s.nextID, nil
}

func (s *fakeStore) GetAccessTokenForVerification(_ context.Context, id int64) (dbgen.GetAccessTokenForVerificationRow, error) {
	row, ok := s.tokens[id]
	if !ok {
		return dbgen.GetAccessTokenForVerificationRow{}, errNotFound
	}
	return row, nil
}

func (s *fakeStore) TouchAccessToken(_ context.Context, id int64) error {
	s.touched[id]++
	return nil
}

type sentinelErr string

func (e sentinelErr) Error() string { return string(e) }

const errNotFound = sentinelErr("not found")

func TestIssueThenVerifyRoundTrip(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)
	ctx := context.Background()

	token, err := svc.Issue(ctx, 42)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	// Length check: 40 random chars + 8 hex crc32 chars = 48, per SPEC-GO.md §6.1.
	i := 0
	for ; i < len(token); i++ {
		if token[i] == '|' {
			break
		}
	}
	plain := token[i+1:]
	if len(plain) != 48 {
		t.Errorf("plain token length = %d, want 48 (%q)", len(plain), plain)
	}

	user, err := svc.Verify(ctx, token)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if user.ID != 42 || user.Role != "tenant" {
		t.Errorf("Verify() = %+v", user)
	}
	if store.touched[1] != 1 {
		t.Errorf("last_used_at should be touched once, got %d", store.touched[1])
	}
}

func TestVerifyRejectsMalformedToken(t *testing.T) {
	svc := NewService(newFakeStore())
	ctx := context.Background()

	for _, bad := range []string{"", "no-pipe-here", "1|", "|onlyplain", "abc|plain", "0|plain", "-1|plain"} {
		if _, err := svc.Verify(ctx, bad); err != ErrInvalidToken {
			t.Errorf("Verify(%q) err = %v, want ErrInvalidToken", bad, err)
		}
	}
}

func TestVerifyRejectsWrongSecret(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)
	ctx := context.Background()

	token, err := svc.Issue(ctx, 1)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	i := 0
	for ; i < len(token); i++ {
		if token[i] == '|' {
			break
		}
	}
	tampered := token[:i] + "|wrongsecretwrongsecretwrongsecretwrongsecretXX"

	if _, err := svc.Verify(ctx, tampered); err != ErrInvalidToken {
		t.Errorf("Verify(tampered) err = %v, want ErrInvalidToken", err)
	}
}

func TestVerifyRejectsUnknownID(t *testing.T) {
	svc := NewService(newFakeStore())
	if _, err := svc.Verify(context.Background(), "999|whatever"); err != ErrInvalidToken {
		t.Errorf("err = %v, want ErrInvalidToken", err)
	}
}
