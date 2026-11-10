// Package audit writes the append-only audit log (SPEC-GO.md §19) via one
// helper, Record, always inside the caller's own transaction. changes is
// computed from before/after snapshots: only fields that actually differ are
// kept, and sensitive field values are masked regardless of whether they
// changed.
package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"net/netip"
	"reflect"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/suwahadi/sewakos/backend/internal/platform/postgres/dbgen"
)

// sensitiveFields are masked in Diff's output, never shown even when changed
// (SPEC-GO.md §19: password, account_number, token_hash).
var sensitiveFields = map[string]struct{}{
	"password":       {},
	"account_number": {},
	"token_hash":     {},
}

const masked = "***"

// Diff returns only the fields where before and after differ, as
// {field: [before, after]} (API-OWNER.md §11's exact shape). Sensitive field
// values are replaced with "***" in the output whenever that field changed.
func Diff(before, after map[string]any) map[string]any {
	changes := map[string]any{}
	seen := map[string]struct{}{}
	for k := range before {
		seen[k] = struct{}{}
	}
	for k := range after {
		seen[k] = struct{}{}
	}
	for k := range seen {
		b, a := before[k], after[k]
		if reflect.DeepEqual(b, a) {
			continue
		}
		if _, sensitive := sensitiveFields[k]; sensitive {
			changes[k] = []any{masked, masked}
			continue
		}
		changes[k] = []any{b, a}
	}
	return changes
}

// Params is one audit log entry to append.
type Params struct {
	ActorID    *int64 // nil means the system (a worker job), not a user
	ActorRole  *string
	Action     string // e.g. "booking.approve", "property.update"
	EntityType string
	EntityID   int64
	OwnerID    *int64 // the entity's owner, for owner-scoped audit queries
	Before     map[string]any
	After      map[string]any
	IP         *netip.Addr
	UserAgent  *string
}

// Store is the subset of dbgen.Querier this package needs.
type Store interface {
	InsertAuditLog(ctx context.Context, arg dbgen.InsertAuditLogParams) (int64, error)
}

// Record appends one audit log row and returns its id.
func Record(ctx context.Context, store Store, p Params) (int64, error) {
	changes, err := json.Marshal(Diff(p.Before, p.After))
	if err != nil {
		return 0, fmt.Errorf("marshal audit changes: %w", err)
	}

	id, err := store.InsertAuditLog(ctx, dbgen.InsertAuditLogParams{
		ActorID:    toPgInt8(p.ActorID),
		ActorRole:  toPgText(p.ActorRole),
		Action:     p.Action,
		EntityType: p.EntityType,
		EntityID:   p.EntityID,
		OwnerID:    toPgInt8(p.OwnerID),
		Changes:    changes,
		Ip:         p.IP,
		UserAgent:  toPgText(p.UserAgent),
	})
	if err != nil {
		return 0, fmt.Errorf("record audit log %q: %w", p.Action, err)
	}
	return id, nil
}

func toPgInt8(v *int64) pgtype.Int8 {
	if v == nil {
		return pgtype.Int8{}
	}
	return pgtype.Int8{Int64: *v, Valid: true}
}

func toPgText(v *string) pgtype.Text {
	if v == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *v, Valid: true}
}
