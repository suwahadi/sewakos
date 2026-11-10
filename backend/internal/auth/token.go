// Package auth implements access token creation and verification
// (SPEC-GO.md §6.1). Issuing a token via register/login OTP flows is wired up
// in a later task; this package is the shared primitive both that and the
// auth middleware (server.RequireAuth) depend on.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"hash/crc32"
	"strconv"
	"strings"

	"github.com/suwahadi/sewakos/backend/internal/platform/postgres/dbgen"
)

// ErrInvalidToken is returned by Verify for any malformed or unknown token;
// callers must not distinguish the reason in the response (SPEC-GO.md §5.3).
var ErrInvalidToken = errors.New("auth: invalid token")

const randomPartLength = 40

var randomAlphabet = []byte("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789")

// VerifiedUser is what Verify returns on success.
type VerifiedUser struct {
	ID   int64
	Role string
}

// Store is the subset of dbgen.Querier this package needs.
type Store interface {
	CreateAccessToken(ctx context.Context, arg dbgen.CreateAccessTokenParams) (int64, error)
	GetAccessTokenForVerification(ctx context.Context, id int64) (dbgen.GetAccessTokenForVerificationRow, error)
	TouchAccessToken(ctx context.Context, id int64) error
}

// Service issues and verifies access tokens.
type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

// Issue creates a new access token for userID and returns it in the
// "{id}|{plain}" form every contract example shows (SPEC-GO.md §6.1).
func (s *Service) Issue(ctx context.Context, userID int64) (string, error) {
	random, err := randomString(randomPartLength)
	if err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	checksum := crc32.ChecksumIEEE([]byte(random))
	plain := random + fmt.Sprintf("%08x", checksum)

	id, err := s.store.CreateAccessToken(ctx, dbgen.CreateAccessTokenParams{
		UserID:    userID,
		TokenHash: hashToken(plain),
	})
	if err != nil {
		return "", fmt.Errorf("store access token: %w", err)
	}
	return fmt.Sprintf("%d|%s", id, plain), nil
}

// Verify checks token's format and hash against storage, and touches
// last_used_at on success. Tokens never expire (SPEC-GO.md §6.1).
func (s *Service) Verify(ctx context.Context, token string) (VerifiedUser, error) {
	id, plain, ok := splitToken(token)
	if !ok {
		return VerifiedUser{}, ErrInvalidToken
	}

	row, err := s.store.GetAccessTokenForVerification(ctx, id)
	if err != nil {
		return VerifiedUser{}, ErrInvalidToken
	}

	if subtle.ConstantTimeCompare([]byte(hashToken(plain)), []byte(row.TokenHash)) != 1 {
		return VerifiedUser{}, ErrInvalidToken
	}

	_ = s.store.TouchAccessToken(ctx, id)
	return VerifiedUser{ID: row.UserID, Role: row.Role}, nil
}

// splitToken parses "{id}|{plain}", id being a positive integer.
func splitToken(token string) (id int64, plain string, ok bool) {
	i := strings.IndexByte(token, '|')
	if i <= 0 || i == len(token)-1 {
		return 0, "", false
	}
	id, err := strconv.ParseInt(token[:i], 10, 64)
	if err != nil || id <= 0 {
		return 0, "", false
	}
	return id, token[i+1:], true
}

func hashToken(plain string) string {
	sum := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(sum[:])
}

func randomString(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	out := make([]byte, n)
	for i, b := range buf {
		out[i] = randomAlphabet[int(b)%len(randomAlphabet)]
	}
	return string(out), nil
}
