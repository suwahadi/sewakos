package finance

import (
	"context"
	"testing"

	"github.com/suwahadi/sewakos/backend/internal/platform/postgres/dbgen"
)

type fakeStore struct {
	inserted []dbgen.InsertLedgerEntryParams
}

func (s *fakeStore) InsertLedgerEntry(_ context.Context, arg dbgen.InsertLedgerEntryParams) (int64, error) {
	s.inserted = append(s.inserted, arg)
	return int64(len(s.inserted)), nil
}

func TestRecordMapsOptionalFKsToNull(t *testing.T) {
	store := &fakeStore{}
	ownerID := int64(7)

	id, err := Record(context.Background(), store, Entry{
		Type:        TypeOwnerEarning,
		Amount:      4250000,
		OwnerID:     &ownerID,
		Description: "Sewa + deposit",
	})
	if err != nil {
		t.Fatalf("Record: %v", err)
	}
	if id != 1 {
		t.Errorf("id = %d, want 1", id)
	}

	arg := store.inserted[0]
	if !arg.OwnerID.Valid || arg.OwnerID.Int64 != 7 {
		t.Errorf("OwnerID = %+v", arg.OwnerID)
	}
	if arg.TenantID.Valid || arg.BookingID.Valid || arg.CreatedBy.Valid {
		t.Errorf("unset optional FKs should be NULL, got %+v", arg)
	}
	if arg.Amount != 4250000 || arg.Type != TypeOwnerEarning {
		t.Errorf("arg = %+v", arg)
	}
}
