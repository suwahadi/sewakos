-- +goose Up
-- The app role's password is never committed to version control: this migration only
-- shapes the role and its grants. Set/rotate the password out of band after migrating
-- (`ALTER ROLE kos_app WITH PASSWORD '...'`), then point DATABASE_URL at it.
-- +goose StatementBegin
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'kos_app') THEN
    CREATE ROLE kos_app LOGIN;
  END IF;
END;
$$;
-- +goose StatementEnd

GRANT USAGE ON SCHEMA public TO kos_app;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO kos_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT USAGE, SELECT ON SEQUENCES TO kos_app;

-- Full read/write on ordinary application tables.
GRANT SELECT, INSERT, UPDATE, DELETE ON
  users, areas, properties, rooms, bookings, user_notifications, access_tokens,
  invoices, payments, refunds, payouts, owner_bank_accounts, settings,
  property_photos, storage_deletions, reconciliation_issues, reviews, favorites,
  property_views_daily, email_outbox
  TO kos_app;

-- Ledger & audit are append-only for the application: INSERT and SELECT only,
-- no UPDATE/DELETE/TRUNCATE (enforced both here and by the triggers in 00004).
GRANT SELECT, INSERT ON ledger_entries, audit_logs TO kos_app;

-- +goose Down
REVOKE ALL PRIVILEGES ON ALL TABLES IN SCHEMA public FROM kos_app;
REVOKE ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public FROM kos_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public REVOKE USAGE, SELECT ON SEQUENCES FROM kos_app;
REVOKE USAGE ON SCHEMA public FROM kos_app;
