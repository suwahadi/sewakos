// Package finance writes to the append-only ledger (SPEC-GO.md §19) via one
// function, always inside the caller's own transaction: Record never opens or
// commits a transaction itself, it just executes one INSERT using whatever
// *dbgen.Queries the caller built from its own pgx.Tx.
package finance

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/suwahadi/sewakos/backend/internal/platform/postgres/dbgen"
)

// Ledger entry types (API-ADMIN.md §10.2) and their amount sign convention.
const (
	TypePaymentReceived  = "payment_received"   // +
	TypeAdminFeeEarned   = "admin_fee_earned"   // +
	TypeOwnerEarning     = "owner_earning"      // +
	TypePayoutCompleted  = "payout_completed"   // -
	TypeRefundDue        = "refund_due"         // +
	TypeRefundCompleted  = "refund_completed"   // -
	TypeAdminFeeReversed = "admin_fee_reversed" // -
	TypeDepositReturned  = "deposit_returned"   // note only, positive
)

// Entry is one ledger row to append.
type Entry struct {
	Type        string
	Amount      int64
	OwnerID     *int64
	TenantID    *int64
	BookingID   *int64
	InvoiceID   *int64
	PaymentID   *int64
	RefundID    *int64
	PayoutID    *int64
	Description string
	CreatedBy   *int64 // nil means the system, not a user action
}

// Store is the subset of dbgen.Querier this package needs.
type Store interface {
	InsertLedgerEntry(ctx context.Context, arg dbgen.InsertLedgerEntryParams) (int64, error)
}

// Record appends entry to the ledger and returns its id.
func Record(ctx context.Context, store Store, entry Entry) (int64, error) {
	id, err := store.InsertLedgerEntry(ctx, dbgen.InsertLedgerEntryParams{
		Type:        entry.Type,
		Amount:      entry.Amount,
		OwnerID:     toPgInt8(entry.OwnerID),
		TenantID:    toPgInt8(entry.TenantID),
		BookingID:   toPgInt8(entry.BookingID),
		InvoiceID:   toPgInt8(entry.InvoiceID),
		PaymentID:   toPgInt8(entry.PaymentID),
		RefundID:    toPgInt8(entry.RefundID),
		PayoutID:    toPgInt8(entry.PayoutID),
		Description: entry.Description,
		CreatedBy:   toPgInt8(entry.CreatedBy),
	})
	if err != nil {
		return 0, fmt.Errorf("record ledger entry %q: %w", entry.Type, err)
	}
	return id, nil
}

func toPgInt8(v *int64) pgtype.Int8 {
	if v == nil {
		return pgtype.Int8{}
	}
	return pgtype.Int8{Int64: *v, Valid: true}
}
