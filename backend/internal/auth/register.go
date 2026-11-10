package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"

	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"

	"github.com/suwahadi/sewakos/backend/internal/platform/postgres/dbgen"
	"github.com/suwahadi/sewakos/backend/internal/response"
	"github.com/suwahadi/sewakos/backend/internal/validation"
)

// bcryptCost is fixed by SPEC-GO.md §5.1.
const bcryptCost = 12

// PhoneRegex is the Indonesian mobile number format every endpoint that
// accepts a phone number validates against (API.md §2.1 and others).
var PhoneRegex = regexp.MustCompile(`^08\d{7,12}$`)

// HashPassword hashes password with bcrypt at the fixed cost SPEC-GO.md §5.1 requires.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(hash), nil
}

// RegisterStore is the subset of dbgen.Querier RegisterService needs.
type RegisterStore interface {
	ExistsUserByPhone(ctx context.Context, phone string) (bool, error)
	ExistsUserByEmail(ctx context.Context, email pgtype.Text) (bool, error)
	CreateUser(ctx context.Context, arg dbgen.CreateUserParams) (int64, error)
}

// RegisterInput is RegisterService.Register's already-validated input.
type RegisterInput struct {
	Name     string
	Phone    string
	Email    *string
	Password string
	Role     string
}

// RegisterResult is what Register returns on success.
type RegisterResult struct {
	UserID int64
	Name   string
	Phone  string
}

// RegisterService implements API.md §2.1 / SPEC-GO.md §6.2's user creation.
// Phone/email uniqueness is checked by the handler (so a 422 attaches to the
// right field via validation.Unique); Register itself assumes its input is
// already valid.
type RegisterService struct {
	store RegisterStore
}

func NewRegisterService(store RegisterStore) *RegisterService {
	return &RegisterService{store: store}
}

func (s *RegisterService) Register(ctx context.Context, in RegisterInput) (RegisterResult, error) {
	hash, err := HashPassword(in.Password)
	if err != nil {
		return RegisterResult{}, err
	}

	id, err := s.store.CreateUser(ctx, dbgen.CreateUserParams{
		Name:     in.Name,
		Phone:    in.Phone,
		Email:    toPgText(in.Email),
		Password: pgtype.Text{String: hash, Valid: true},
		Role:     in.Role,
	})
	if err != nil {
		return RegisterResult{}, fmt.Errorf("create user: %w", err)
	}

	return RegisterResult{UserID: id, Name: in.Name, Phone: in.Phone}, nil
}

// RegisterHandler implements POST /auth/register (API.md §2.1, SPEC-GO.md §6.2).
func RegisterHandler(svc *RegisterService, store RegisterStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		ctx := r.Context()

		phone, _ := body["phone_number"].(string)
		phoneTaken := false
		if phone != "" {
			phoneTaken, _ = store.ExistsUserByPhone(ctx, phone)
		}
		email, hasEmail := body["email"].(string)
		emailTaken := false
		if hasEmail && email != "" {
			emailTaken, _ = store.ExistsUserByEmail(ctx, pgtype.Text{String: email, Valid: true})
		}

		v := validation.New()
		v.Field("name", body["name"]).Required().String().MaxStringLen(255)
		v.Field("phone_number", body["phone_number"]).Required().String().Regex(PhoneRegex).Unique(phoneTaken)
		v.Field("email", body["email"]).Optional().Email().Unique(emailTaken)
		v.Field("password", body["password"]).Required().String().MinStringLen(8)
		v.Field("role", body["role"]).Optional().In("tenant", "owner")
		v.Field("is_agree_privacy_policy", body["is_agree_privacy_policy"]).Accepted()
		if !v.Valid() {
			response.Fail(w, http.StatusUnprocessableEntity, response.MsgValidationFailed, v.Errors())
			return
		}

		role, _ := body["role"].(string)
		if role == "" {
			role = "tenant"
		}
		var emailPtr *string
		if hasEmail && email != "" {
			emailPtr = &email
		}

		result, err := svc.Register(ctx, RegisterInput{
			Name:     body["name"].(string),
			Phone:    phone,
			Email:    emailPtr,
			Password: body["password"].(string),
			Role:     role,
		})
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}

		response.Success(w, http.StatusOK, "Registrasi berhasil. Silakan verifikasi nomor handphone.", map[string]any{
			"user": map[string]any{
				"id":           result.UserID,
				"name":         result.Name,
				"phone_number": result.Phone,
			},
			"next_step": "otp_verification",
		})
	}
}

func toPgText(v *string) pgtype.Text {
	if v == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *v, Valid: true}
}
