// Package db holds integration tests that exercise the migrated schema directly
// (append-only enforcement, role grants) rather than any Go package's behavior.
package db

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func dsn(envVar, fallback string) string {
	if v := os.Getenv(envVar); v != "" {
		return v
	}
	return fallback
}

func connect(t *testing.T, dsn string) *pgxpool.Pool {
	t.Helper()
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)
	if err := pool.Ping(context.Background()); err != nil {
		t.Skipf("postgres not reachable, skipping integration test: %v", err)
	}
	return pool
}

// TestLedgerAndAuditAreAppendOnly proves SPEC-GO.md §4's two layers of protection:
// the BEFORE UPDATE OR DELETE trigger rejects mutation even for the migration-owner
// role, and the restricted app role additionally lacks the UPDATE/DELETE grant.
func TestLedgerAndAuditAreAppendOnly(t *testing.T) {
	owner := connect(t, dsn("TEST_DATABASE_URL", "postgres://kos:kos@localhost:5432/kos?sslmode=disable"))
	app := connect(t, dsn("TEST_APP_DATABASE_URL", "postgres://kos_app:kos_app@localhost:5432/kos?sslmode=disable"))
	ctx := context.Background()

	for _, table := range []string{"ledger_entries", "audit_logs"} {
		var insertSQL string
		switch table {
		case "ledger_entries":
			insertSQL = `INSERT INTO ledger_entries (type, amount, description) VALUES ('payment_received', 1, 'schema_test') RETURNING id`
		case "audit_logs":
			insertSQL = `INSERT INTO audit_logs (action, entity_type, entity_id) VALUES ('schema_test.probe', 'booking', 1) RETURNING id`
		}

		// The row inserted below is never cleaned up: that's the point of append-only
		// storage, and bypassing it would need superuser, which the migration-owner
		// role intentionally isn't. Fine in CI/dev, where the database is ephemeral.
		var id int64
		if err := owner.QueryRow(ctx, insertSQL).Scan(&id); err != nil {
			t.Fatalf("insert into %s as owner role: %v", table, err)
		}

		if _, err := owner.Exec(ctx, "UPDATE "+table+" SET created_at = now() WHERE id = $1", id); err == nil {
			t.Errorf("%s: owner role UPDATE should be rejected by the append-only trigger", table)
		}
		if _, err := owner.Exec(ctx, "DELETE FROM "+table+" WHERE id = $1", id); err == nil {
			t.Errorf("%s: owner role DELETE should be rejected by the append-only trigger", table)
		}

		if _, err := app.Exec(ctx, "UPDATE "+table+" SET created_at = now() WHERE id = $1", id); err == nil {
			t.Errorf("%s: app role UPDATE should be rejected (no grant)", table)
		}
		if _, err := app.Exec(ctx, "DELETE FROM "+table+" WHERE id = $1", id); err == nil {
			t.Errorf("%s: app role DELETE should be rejected (no grant)", table)
		}
	}
}

// TestDeletingPhotoQueuesStorageDeletion proves the AFTER DELETE trigger on
// property_photos queues both object keys, including when the row disappears
// via a cascading delete (e.g. its room being removed).
func TestDeletingPhotoQueuesStorageDeletion(t *testing.T) {
	owner := connect(t, dsn("TEST_DATABASE_URL", "postgres://kos:kos@localhost:5432/kos?sslmode=disable"))
	ctx := context.Background()

	var userID, areaID, propertyID, roomID, photoID int64
	if err := owner.QueryRow(ctx,
		`INSERT INTO users (name, phone) VALUES ('schema_test owner', '089900000001') RETURNING id`,
	).Scan(&userID); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	t.Cleanup(func() { _, _ = owner.Exec(ctx, "DELETE FROM users WHERE id = $1", userID) })

	if err := owner.QueryRow(ctx,
		`INSERT INTO areas (name, slug, type, latitude, longitude) VALUES ('schema_test area', 'schema-test-area', 'city', 0, 0) RETURNING id`,
	).Scan(&areaID); err != nil {
		t.Fatalf("insert area: %v", err)
	}
	t.Cleanup(func() { _, _ = owner.Exec(ctx, "DELETE FROM areas WHERE id = $1", areaID) })

	if err := owner.QueryRow(ctx,
		`INSERT INTO properties (owner_id, area_id, name, slug, gender, latitude, longitude)
		 VALUES ($1, $2, 'schema_test property', 'schema-test-property', 'campur', 0, 0) RETURNING id`,
		userID, areaID,
	).Scan(&propertyID); err != nil {
		t.Fatalf("insert property: %v", err)
	}
	t.Cleanup(func() { _, _ = owner.Exec(ctx, "DELETE FROM properties WHERE id = $1", propertyID) })

	if err := owner.QueryRow(ctx,
		`INSERT INTO rooms (property_id, name, price_monthly) VALUES ($1, 'schema_test room', 1) RETURNING id`,
		propertyID,
	).Scan(&roomID); err != nil {
		t.Fatalf("insert room: %v", err)
	}

	if err := owner.QueryRow(ctx,
		`INSERT INTO property_photos (property_id, room_id, object_key, thumb_key, width, height, size_bytes)
		 VALUES ($1, $2, 'schema_test/large.webp', 'schema_test/thumb.webp', 10, 10, 10) RETURNING id`,
		propertyID, roomID,
	).Scan(&photoID); err != nil {
		t.Fatalf("insert photo: %v", err)
	}

	// Deleting the room cascades to property_photos; the trigger must still fire.
	if _, err := owner.Exec(ctx, "DELETE FROM rooms WHERE id = $1", roomID); err != nil {
		t.Fatalf("delete room: %v", err)
	}

	rows, err := owner.Query(ctx, "SELECT object_key FROM storage_deletions WHERE object_key LIKE 'schema_test/%' ORDER BY object_key")
	if err != nil {
		t.Fatalf("query storage_deletions: %v", err)
	}
	defer rows.Close()

	var keys []string
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			t.Fatalf("scan: %v", err)
		}
		keys = append(keys, key)
	}
	t.Cleanup(func() { _, _ = owner.Exec(ctx, "DELETE FROM storage_deletions WHERE object_key LIKE 'schema_test/%'") })

	if len(keys) != 2 || keys[0] != "schema_test/large.webp" || keys[1] != "schema_test/thumb.webp" {
		t.Fatalf("expected both object keys queued for deletion, got %v", keys)
	}
}
