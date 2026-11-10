// Package account implements API.md §11: logout, password reset, profile,
// password change, and change-phone. It depends on internal/auth only for
// the phone number format (auth.PhoneRegex) and token issuance primitives,
// not the other way around.
package account

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/redis/go-redis/v9"

	"github.com/suwahadi/sewakos/backend/internal/otp"
	"github.com/suwahadi/sewakos/backend/internal/platform/postgres/dbgen"
)

// Store is the subset of dbgen.Querier this package needs.
type Store interface {
	GetUserByID(ctx context.Context, id int64) (dbgen.GetUserByIDRow, error)
	GetUserPasswordByID(ctx context.Context, id int64) (pgtype.Text, error)
	GetUserByPhone(ctx context.Context, phone string) (dbgen.GetUserByPhoneRow, error)
	ExistsUserByPhone(ctx context.Context, phone string) (bool, error)
	ExistsUserByEmail(ctx context.Context, email pgtype.Text) (bool, error)
	UpdateUserProfile(ctx context.Context, arg dbgen.UpdateUserProfileParams) error
	UpdateUserPassword(ctx context.Context, arg dbgen.UpdateUserPasswordParams) error
	UpdateUserPasswordAndVerify(ctx context.Context, arg dbgen.UpdateUserPasswordAndVerifyParams) error
	UpdateUserPhone(ctx context.Context, arg dbgen.UpdateUserPhoneParams) error
	DeleteAccessToken(ctx context.Context, id int64) error
	DeleteAllAccessTokensForUser(ctx context.Context, userID int64) error
	DeleteOtherAccessTokensForUser(ctx context.Context, arg dbgen.DeleteOtherAccessTokensForUserParams) error
}

// Deps bundles every account handler's shared dependencies.
type Deps struct {
	Store        Store
	OTP          *otp.Service
	Redis        *redis.Client
	MaxPerMinute int
	DebugEcho    bool
}

// userResponse builds the {id,name,role,phone_number,email,is_verified}
// shape API.md §11.3-11.4 and §11.6 share.
func userResponse(id int64, name, role, phone string, email pgtype.Text, verified bool) map[string]any {
	var emailValue any
	if email.Valid {
		emailValue = email.String
	}
	return map[string]any{
		"id":           id,
		"name":         name,
		"role":         role,
		"phone_number": phone,
		"email":        emailValue,
		"is_verified":  verified,
	}
}
