-- +goose Up
CREATE TABLE invoices (
  id bigserial PRIMARY KEY,
  booking_id bigint NOT NULL REFERENCES bookings (id),
  number varchar(30) NOT NULL UNIQUE,
  type varchar(10) NOT NULL CHECK (type IN ('initial', 'renewal', 'extension')),
  period_start date NOT NULL,
  period_end date NOT NULL,
  periods smallint NOT NULL,
  rent_amount bigint NOT NULL,
  discount bigint NOT NULL DEFAULT 0,
  deposit bigint NOT NULL DEFAULT 0,
  admin_fee bigint NOT NULL DEFAULT 0,
  total bigint NOT NULL,
  status varchar(10) NOT NULL DEFAULT 'unpaid' CHECK (status IN ('unpaid', 'paid', 'void')),
  due_date date NOT NULL,
  issued_at timestamptz NOT NULL,
  paid_at timestamptz,
  void_reason varchar(30),
  reminder_sent_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX idx_invoices_booking_renewal_period ON invoices (booking_id, type, period_start)
  WHERE type = 'renewal';
CREATE UNIQUE INDEX idx_invoices_booking_initial ON invoices (booking_id) WHERE type = 'initial';
CREATE INDEX idx_invoices_status_due_date ON invoices (status, due_date);

CREATE TABLE payments (
  id bigserial PRIMARY KEY,
  invoice_id bigint NOT NULL REFERENCES invoices (id),
  order_id varchar(50) NOT NULL UNIQUE,
  gross_amount bigint NOT NULL,
  status varchar(10) NOT NULL DEFAULT 'pending'
    CHECK (status IN ('pending', 'settlement', 'expire', 'cancel', 'deny', 'failure')),
  snap_token varchar(100),
  redirect_url text,
  payment_type varchar(50),
  midtrans_transaction_id varchar(100),
  expires_at timestamptz NOT NULL,
  paid_at timestamptz,
  last_notification jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX idx_payments_invoice_pending ON payments (invoice_id) WHERE status = 'pending';
CREATE INDEX idx_payments_status_expires_at ON payments (status, expires_at);

CREATE TABLE refunds (
  id bigserial PRIMARY KEY,
  booking_id bigint NOT NULL REFERENCES bookings (id),
  invoice_id bigint NOT NULL UNIQUE REFERENCES invoices (id),
  amount bigint NOT NULL,
  reason varchar(20) NOT NULL
    CHECK (reason IN ('owner_rejected', 'owner_timeout', 'tenant_cancelled', 'stock_unavailable', 'invoice_void')),
  status varchar(20) NOT NULL DEFAULT 'awaiting_account'
    CHECK (status IN ('awaiting_account', 'ready', 'completed')),
  bank_name varchar(100),
  account_number varchar(20),
  account_name varchar(255),
  transfer_reference varchar(100),
  note text,
  completed_by bigint REFERENCES users (id),
  completed_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE payouts (
  id bigserial PRIMARY KEY,
  booking_id bigint NOT NULL REFERENCES bookings (id),
  invoice_id bigint NOT NULL UNIQUE REFERENCES invoices (id),
  owner_id bigint NOT NULL REFERENCES users (id),
  amount bigint NOT NULL,
  status varchar(10) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'completed')),
  bank_name varchar(100),
  account_number varchar(20),
  account_name varchar(255),
  transfer_reference varchar(100),
  note text,
  completed_by bigint REFERENCES users (id),
  completed_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_payouts_owner_id_created_at ON payouts (owner_id, created_at DESC);

CREATE TABLE owner_bank_accounts (
  user_id bigint PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
  bank_name varchar(100) NOT NULL,
  account_number varchar(20) NOT NULL,
  account_name varchar(255) NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE settings (
  key varchar(50) PRIMARY KEY,
  value jsonb NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE settings;
DROP TABLE owner_bank_accounts;
DROP TABLE payouts;
DROP TABLE refunds;
DROP TABLE payments;
DROP TABLE invoices;
