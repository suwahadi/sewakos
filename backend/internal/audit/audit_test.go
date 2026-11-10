package audit

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/suwahadi/sewakos/backend/internal/platform/postgres/dbgen"
)

func TestDiffOnlyIncludesChangedFields(t *testing.T) {
	before := map[string]any{"status": "paid", "price": float64(2000000)}
	after := map[string]any{"status": "approved", "price": float64(2000000)}

	got := Diff(before, after)
	want := map[string]any{"status": []any{"paid", "approved"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Diff() = %#v, want %#v", got, want)
	}
}

func TestDiffMasksSensitiveFieldsWhenChanged(t *testing.T) {
	got := Diff(
		map[string]any{"password": "old-hash", "account_number": "123", "token_hash": "abc"},
		map[string]any{"password": "new-hash", "account_number": "456", "token_hash": "def"},
	)
	want := map[string]any{
		"password":       []any{"***", "***"},
		"account_number": []any{"***", "***"},
		"token_hash":     []any{"***", "***"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Diff() = %#v, want %#v", got, want)
	}
}

func TestDiffUnchangedSensitiveFieldIsOmitted(t *testing.T) {
	got := Diff(map[string]any{"password": "same"}, map[string]any{"password": "same"})
	if len(got) != 0 {
		t.Errorf("Diff() = %#v, want empty", got)
	}
}

type fakeAuditStore struct {
	inserted []dbgen.InsertAuditLogParams
}

func (s *fakeAuditStore) InsertAuditLog(_ context.Context, arg dbgen.InsertAuditLogParams) (int64, error) {
	s.inserted = append(s.inserted, arg)
	return int64(len(s.inserted)), nil
}

func TestRecordWritesExpectedChangesShape(t *testing.T) {
	store := &fakeAuditStore{}
	actorID := int64(1)
	role := "owner"

	id, err := Record(context.Background(), store, Params{
		ActorID:    &actorID,
		ActorRole:  &role,
		Action:     "booking.approve",
		EntityType: "booking",
		EntityID:   2,
		Before:     map[string]any{"status": "paid"},
		After:      map[string]any{"status": "approved"},
	})
	if err != nil {
		t.Fatalf("Record: %v", err)
	}
	if id != 1 {
		t.Errorf("id = %d, want 1", id)
	}

	var changes map[string]any
	if err := json.Unmarshal(store.inserted[0].Changes, &changes); err != nil {
		t.Fatalf("unmarshal changes: %v", err)
	}
	status, ok := changes["status"].([]any)
	if !ok || len(status) != 2 || status[0] != "paid" || status[1] != "approved" {
		t.Errorf("changes[status] = %v", changes["status"])
	}
	if !store.inserted[0].ActorID.Valid || store.inserted[0].ActorID.Int64 != 1 {
		t.Errorf("ActorID = %+v", store.inserted[0].ActorID)
	}
}

func TestRecordActorNilMeansSystem(t *testing.T) {
	store := &fakeAuditStore{}
	_, err := Record(context.Background(), store, Params{
		Action:     "booking.expire",
		EntityType: "booking",
		EntityID:   2,
	})
	if err != nil {
		t.Fatalf("Record: %v", err)
	}
	if store.inserted[0].ActorID.Valid {
		t.Errorf("ActorID should be NULL for system actions, got %+v", store.inserted[0].ActorID)
	}
}
