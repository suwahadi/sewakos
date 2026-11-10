// Package notification creates in-app notifications and, when the recipient
// has an email on file, enqueues the matching email in the same transaction
// (SPEC-GO.md §17). The text table per event (title/body/link) is built by
// each feature as it's implemented (B2+); this package is the one shared
// primitive they all call.
package notification

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/suwahadi/sewakos/backend/internal/platform/postgres/dbgen"
)

// Categories (API.md §6).
const (
	CategoryBooking = "booking"
	CategoryPayment = "payment"
	CategoryPromo   = "promo"
	CategoryChat    = "chat"
)

// Params is one notification to create.
type Params struct {
	UserID int64
	// RecipientEmail is the recipient's email, or nil if they have none on
	// file (API.md §11.4: email is never verified, just optionally set).
	RecipientEmail *string
	Category       string
	Title          string
	Body           string
	// Link is the frontend path this notification points to (DESIGN.md
	// §7.0), or nil (e.g. broadcast promo, API.md §6).
	Link *string
}

// Store is the subset of dbgen.Querier this package needs.
type Store interface {
	CreateUserNotification(ctx context.Context, arg dbgen.CreateUserNotificationParams) (int64, error)
	EnqueueEmail(ctx context.Context, arg dbgen.EnqueueEmailParams) (int64, error)
}

// Create inserts the notification and, unless it's a promo broadcast (SPEC-GO.md
// §17: "Broadcast promo admin tidak dikirim lewat email") and the recipient has
// an email, enqueues the matching outbound email. appPublicURL is appended to
// the email body as the notification's link base (never any account/bank data).
func Create(ctx context.Context, store Store, appPublicURL string, p Params) (int64, error) {
	id, err := store.CreateUserNotification(ctx, dbgen.CreateUserNotificationParams{
		UserID:   p.UserID,
		Category: p.Category,
		Title:    p.Title,
		Body:     p.Body,
		Link:     toPgText(p.Link),
	})
	if err != nil {
		return 0, fmt.Errorf("create notification: %w", err)
	}

	if p.Category == CategoryPromo || p.RecipientEmail == nil || *p.RecipientEmail == "" {
		return id, nil
	}

	body := p.Body
	if appPublicURL != "" {
		body += "\n\n" + appPublicURL
	}
	if _, err := store.EnqueueEmail(ctx, dbgen.EnqueueEmailParams{
		ToEmail:  *p.RecipientEmail,
		Subject:  p.Title,
		BodyText: body,
	}); err != nil {
		return 0, fmt.Errorf("enqueue email: %w", err)
	}
	return id, nil
}

func toPgText(v *string) pgtype.Text {
	if v == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *v, Valid: true}
}
