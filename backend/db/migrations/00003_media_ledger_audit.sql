-- +goose Up
CREATE TABLE property_photos (
  id bigserial PRIMARY KEY,
  property_id bigint NOT NULL REFERENCES properties (id) ON DELETE CASCADE,
  room_id bigint REFERENCES rooms (id) ON DELETE CASCADE,
  object_key varchar(500) NOT NULL UNIQUE,
  thumb_key varchar(500) NOT NULL UNIQUE,
  width integer NOT NULL,
  height integer NOT NULL,
  size_bytes integer NOT NULL,
  caption varchar(255),
  sort_order integer NOT NULL DEFAULT 0,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_property_photos_property_sort ON property_photos (property_id, sort_order, id);

CREATE TABLE storage_deletions (
  id bigserial PRIMARY KEY,
  object_key varchar(500) NOT NULL,
  attempts smallint NOT NULL DEFAULT 0,
  next_attempt_at timestamptz NOT NULL DEFAULT now(),
  last_error text,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_storage_deletions_next_attempt_at ON storage_deletions (next_attempt_at);

CREATE TABLE ledger_entries (
  id bigserial PRIMARY KEY,
  occurred_at timestamptz NOT NULL DEFAULT now(),
  type varchar(30) NOT NULL,
  amount bigint NOT NULL,
  owner_id bigint REFERENCES users (id),
  tenant_id bigint REFERENCES users (id),
  booking_id bigint REFERENCES bookings (id),
  invoice_id bigint REFERENCES invoices (id),
  payment_id bigint REFERENCES payments (id),
  refund_id bigint REFERENCES refunds (id),
  payout_id bigint REFERENCES payouts (id),
  description varchar(255) NOT NULL,
  created_by bigint REFERENCES users (id)
);
CREATE INDEX idx_ledger_entries_owner_occurred_at ON ledger_entries (owner_id, occurred_at DESC);
CREATE INDEX idx_ledger_entries_type_occurred_at ON ledger_entries (type, occurred_at);

CREATE TABLE audit_logs (
  id bigserial PRIMARY KEY,
  created_at timestamptz NOT NULL DEFAULT now(),
  actor_id bigint REFERENCES users (id),
  actor_role varchar(10),
  action varchar(50) NOT NULL,
  entity_type varchar(30) NOT NULL,
  entity_id bigint NOT NULL,
  owner_id bigint,
  changes jsonb NOT NULL DEFAULT '{}',
  ip inet,
  user_agent varchar(255)
);
CREATE INDEX idx_audit_logs_owner_created_at ON audit_logs (owner_id, created_at DESC);
CREATE INDEX idx_audit_logs_entity ON audit_logs (entity_type, entity_id);

CREATE TABLE reconciliation_issues (
  id bigserial PRIMARY KEY,
  payment_id bigint NOT NULL REFERENCES payments (id),
  kind varchar(30) NOT NULL CHECK (kind IN ('status_mismatch', 'amount_mismatch', 'not_found_at_midtrans')),
  local_status varchar(20) NOT NULL,
  midtrans_status varchar(20),
  local_amount bigint NOT NULL,
  midtrans_amount bigint,
  status varchar(10) NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'resolved')),
  note text,
  detected_at timestamptz NOT NULL DEFAULT now(),
  resolved_by bigint REFERENCES users (id),
  resolved_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX idx_reconciliation_issues_open ON reconciliation_issues (payment_id, kind) WHERE status = 'open';

CREATE TABLE reviews (
  id bigserial PRIMARY KEY,
  booking_id bigint NOT NULL UNIQUE REFERENCES bookings (id),
  property_id bigint NOT NULL REFERENCES properties (id),
  tenant_id bigint NOT NULL REFERENCES users (id),
  rating smallint NOT NULL CHECK (rating BETWEEN 1 AND 5),
  comment text,
  is_hidden boolean NOT NULL DEFAULT false,
  hidden_reason text,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_reviews_property_hidden_created_at ON reviews (property_id, is_hidden, created_at DESC);

CREATE TABLE favorites (
  user_id bigint NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  property_id bigint NOT NULL REFERENCES properties (id) ON DELETE CASCADE,
  created_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id, property_id)
);

CREATE TABLE property_views_daily (
  property_id bigint NOT NULL REFERENCES properties (id) ON DELETE CASCADE,
  view_date date NOT NULL,
  views integer NOT NULL DEFAULT 0,
  PRIMARY KEY (property_id, view_date)
);

CREATE TABLE email_outbox (
  id bigserial PRIMARY KEY,
  to_email varchar(255) NOT NULL,
  subject varchar(255) NOT NULL,
  body_text text NOT NULL,
  attempts smallint NOT NULL DEFAULT 0,
  next_attempt_at timestamptz NOT NULL DEFAULT now(),
  sent_at timestamptz,
  last_error text,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_email_outbox_sent_next_attempt ON email_outbox (sent_at, next_attempt_at);

-- +goose Down
DROP TABLE email_outbox;
DROP TABLE property_views_daily;
DROP TABLE favorites;
DROP TABLE reviews;
DROP TABLE reconciliation_issues;
DROP TABLE audit_logs;
DROP TABLE ledger_entries;
DROP TABLE storage_deletions;
DROP TABLE property_photos;
