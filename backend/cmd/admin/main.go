// Command admin manages platform admin accounts, which have no self-service
// creation endpoint by design (API-ADMIN.md §1: "Akun admin hanya dibuat
// lewat CLI atau seed dev").
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/mail"
	"os"
	"regexp"

	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"

	"github.com/suwahadi/sewakos/backend/internal/config"
	"github.com/suwahadi/sewakos/backend/internal/platform/postgres"
	"github.com/suwahadi/sewakos/backend/internal/platform/postgres/dbgen"
)

var phonePattern = regexp.MustCompile(`^08\d{7,12}$`)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	if len(os.Args) < 2 || os.Args[1] != "create" {
		fmt.Fprintln(os.Stderr, "usage: admin create --name NAME --phone 08... --password PASSWORD [--email EMAIL]")
		os.Exit(2)
	}

	fs := flag.NewFlagSet("create", flag.ExitOnError)
	name := fs.String("name", "", "admin's full name")
	phone := fs.String("phone", "", "admin's phone number, format 08xxxxxxxxx")
	password := fs.String("password", "", "admin's password")
	email := fs.String("email", "", "admin's email address (optional)")
	_ = fs.Parse(os.Args[2:])

	if *name == "" || *phone == "" || *password == "" {
		fmt.Fprintln(os.Stderr, "all of --name, --phone, --password are required")
		os.Exit(2)
	}
	if !phonePattern.MatchString(*phone) {
		fmt.Fprintln(os.Stderr, "phone must match ^08\\d{7,12}$")
		os.Exit(2)
	}
	if *email != "" {
		if parsed, err := mail.ParseAddress(*email); err != nil || parsed.Address != *email {
			fmt.Fprintln(os.Stderr, "email must be a plain address such as admin@example.com")
			os.Exit(2)
		}
	}

	cfg, err := config.Load()
	if err != nil {
		logger.Error("load config", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()
	pool, err := postgres.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("connect postgres", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	q := dbgen.New(pool)

	taken, err := q.CountUsersByPhone(ctx, *phone)
	if err != nil {
		logger.Error("check phone", "error", err)
		os.Exit(1)
	}
	if taken > 0 {
		fmt.Fprintln(os.Stderr, "phone number is already in use")
		os.Exit(1)
	}

	if *email != "" {
		emailTaken, err := q.ExistsUserByEmail(ctx, pgtype.Text{String: *email, Valid: true})
		if err != nil {
			logger.Error("check email", "error", err)
			os.Exit(1)
		}
		if emailTaken {
			fmt.Fprintln(os.Stderr, "email is already in use")
			os.Exit(1)
		}
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(*password), 12)
	if err != nil {
		logger.Error("hash password", "error", err)
		os.Exit(1)
	}

	id, err := q.CreateAdminUser(ctx, dbgen.CreateAdminUserParams{
		Name: *name, Phone: *phone, Email: pgtype.Text{String: *email, Valid: *email != ""},
		Password: pgtype.Text{String: string(hash), Valid: true},
	})
	if err != nil {
		logger.Error("create admin", "error", err)
		os.Exit(1)
	}

	fmt.Printf("Created admin user id=%d\n", id)
}
