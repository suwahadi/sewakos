package notification

import (
	"context"
	"testing"

	"github.com/suwahadi/sewakos/backend/internal/platform/postgres/dbgen"
)

type fakeStore struct {
	notifications []dbgen.CreateUserNotificationParams
	emails        []dbgen.EnqueueEmailParams
}

func (s *fakeStore) CreateUserNotification(_ context.Context, arg dbgen.CreateUserNotificationParams) (int64, error) {
	s.notifications = append(s.notifications, arg)
	return int64(len(s.notifications)), nil
}

func (s *fakeStore) EnqueueEmail(_ context.Context, arg dbgen.EnqueueEmailParams) (int64, error) {
	s.emails = append(s.emails, arg)
	return int64(len(s.emails)), nil
}

func strPtr(s string) *string { return &s }

func TestCreateEnqueuesEmailWhenRecipientHasOne(t *testing.T) {
	store := &fakeStore{}
	_, err := Create(context.Background(), store, "https://sewakos.web.id", Params{
		UserID:         2,
		RecipientEmail: strPtr("tenant@example.com"),
		Category:       CategoryBooking,
		Title:          "Booking disetujui",
		Body:           "Pemilik menyetujui booking Kamar Tipe A di Kost Asti Beji.",
		Link:           strPtr("/booking/2"),
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if len(store.emails) != 1 {
		t.Fatalf("expected one email enqueued, got %d", len(store.emails))
	}
	if store.emails[0].ToEmail != "tenant@example.com" || store.emails[0].Subject != "Booking disetujui" {
		t.Errorf("email = %+v", store.emails[0])
	}
	if got := store.emails[0].BodyText; got != "Pemilik menyetujui booking Kamar Tipe A di Kost Asti Beji.\n\nhttps://sewakos.web.id" {
		t.Errorf("body = %q", got)
	}
}

func TestCreateSkipsEmailWithoutRecipientAddress(t *testing.T) {
	store := &fakeStore{}
	_, err := Create(context.Background(), store, "https://sewakos.web.id", Params{
		UserID: 2, Category: CategoryBooking, Title: "t", Body: "b",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if len(store.emails) != 0 {
		t.Errorf("expected no email, got %d", len(store.emails))
	}
}

func TestCreateSkipsEmailForPromoBroadcast(t *testing.T) {
	store := &fakeStore{}
	_, err := Create(context.Background(), store, "https://sewakos.web.id", Params{
		UserID:         2,
		RecipientEmail: strPtr("tenant@example.com"),
		Category:       CategoryPromo,
		Title:          "Promo Ngebut!",
		Body:           "Diskon s.d. Rp250.000 untuk bulan pertama.",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if len(store.emails) != 0 {
		t.Errorf("promo broadcasts must never enqueue email, got %d", len(store.emails))
	}
	if len(store.notifications) != 1 {
		t.Errorf("notification itself should still be created")
	}
}
