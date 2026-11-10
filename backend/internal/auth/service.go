package auth

import (
	"context"
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"

	"github.com/suwahadi/sewakos/backend/internal/platform/postgres/dbgen"
)

// ErrLoginFailed covers every case SPEC-GO.md §5.3/§6.5 requires to be
// indistinguishable to the client: unknown phone, no password set, or a
// wrong password.
var ErrLoginFailed = errors.New("auth: login failed")

// ErrAccountSuspended is checked only after the password matches (SPEC-GO.md §6.5).
var ErrAccountSuspended = errors.New("auth: account suspended")

// LoginResult is what a successful Login returns.
type LoginResult struct {
	Token      string
	UserID     int64
	Name       string
	Role       string
	IsVerified bool
}

// UserStore is the subset of dbgen.Querier LoginService needs.
type UserStore interface {
	GetUserByPhone(ctx context.Context, phone string) (dbgen.GetUserByPhoneRow, error)
}

// LoginService implements API.md §2.4 / SPEC-GO.md §6.5.
type LoginService struct {
	users  UserStore
	tokens *Service
}

func NewLoginService(users UserStore, tokens *Service) *LoginService {
	return &LoginService{users: users, tokens: tokens}
}

func (s *LoginService) Login(ctx context.Context, phone, password string) (LoginResult, error) {
	row, err := s.users.GetUserByPhone(ctx, phone)
	if err != nil {
		return LoginResult{}, ErrLoginFailed
	}
	if !row.Password.Valid {
		return LoginResult{}, ErrLoginFailed
	}
	if bcrypt.CompareHashAndPassword([]byte(row.Password.String), []byte(password)) != nil {
		return LoginResult{}, ErrLoginFailed
	}
	if row.SuspendedAt.Valid {
		return LoginResult{}, ErrAccountSuspended
	}

	token, err := s.tokens.Issue(ctx, row.ID)
	if err != nil {
		return LoginResult{}, fmt.Errorf("issue token: %w", err)
	}
	return LoginResult{
		Token:      token,
		UserID:     row.ID,
		Name:       row.Name,
		Role:       row.Role,
		IsVerified: row.PhoneVerifiedAt.Valid,
	}, nil
}
