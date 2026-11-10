-- +goose Up
CREATE TABLE users (
  id bigserial PRIMARY KEY,
  name varchar(255) NOT NULL,
  role varchar(10) NOT NULL DEFAULT 'tenant' CHECK (role IN ('tenant', 'owner', 'admin')),
  phone varchar(20) NOT NULL UNIQUE,
  email varchar(255) UNIQUE,
  password varchar(255),
  phone_verified_at timestamptz,
  email_verified_at timestamptz,
  suspended_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE areas (
  id bigserial PRIMARY KEY,
  name varchar(255) NOT NULL,
  slug varchar(255) NOT NULL UNIQUE,
  type varchar(10) NOT NULL CHECK (type IN ('city', 'campus', 'mrt')),
  is_top boolean NOT NULL DEFAULT false,
  latitude double precision NOT NULL,
  longitude double precision NOT NULL,
  place_id varchar(255),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_areas_type_is_top ON areas (type, is_top);

CREATE TABLE properties (
  id bigserial PRIMARY KEY,
  owner_id bigint NOT NULL REFERENCES users (id),
  area_id bigint NOT NULL REFERENCES areas (id),
  name varchar(255) NOT NULL,
  slug varchar(255) NOT NULL UNIQUE,
  gender varchar(10) NOT NULL CHECK (gender IN ('campur', 'putra', 'putri')),
  description text NOT NULL DEFAULT '',
  address text NOT NULL DEFAULT '',
  latitude double precision NOT NULL,
  longitude double precision NOT NULL,
  is_active boolean NOT NULL DEFAULT true,
  moderation_status varchar(10) NOT NULL DEFAULT 'pending'
    CHECK (moderation_status IN ('pending', 'approved', 'rejected', 'suspended')),
  moderation_note text,
  moderated_at timestamptz,
  is_flash_sale boolean NOT NULL DEFAULT false,
  flash_sale_discount bigint NOT NULL DEFAULT 0,
  views bigint NOT NULL DEFAULT 0,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_properties_active_moderation ON properties (is_active, moderation_status);
CREATE INDEX idx_properties_owner_id ON properties (owner_id);

CREATE TABLE rooms (
  id bigserial PRIMARY KEY,
  property_id bigint NOT NULL REFERENCES properties (id) ON DELETE CASCADE,
  name varchar(255) NOT NULL,
  price_monthly bigint NOT NULL,
  price_weekly bigint,
  price_daily bigint,
  available smallint NOT NULL DEFAULT 0,
  capacity smallint NOT NULL DEFAULT 1,
  facilities jsonb NOT NULL DEFAULT '[]',
  monthly_duration_options jsonb NOT NULL DEFAULT '[1]',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_rooms_property_id ON rooms (property_id);

CREATE TABLE bookings (
  id bigserial PRIMARY KEY,
  tenant_id bigint NOT NULL REFERENCES users (id),
  room_id bigint NOT NULL REFERENCES rooms (id),
  rent_type varchar(10) NOT NULL CHECK (rent_type IN ('daily', 'weekly', 'monthly')),
  start_date date NOT NULL,
  status varchar(10) NOT NULL DEFAULT 'pending'
    CHECK (status IN ('pending', 'paid', 'approved', 'completed', 'terminated', 'rejected', 'cancelled')),
  duration smallint NOT NULL DEFAULT 1,
  end_date date NOT NULL,
  period_price bigint NOT NULL,
  flash_sale_discount bigint NOT NULL DEFAULT 0,
  price bigint NOT NULL,
  deposit bigint NOT NULL,
  admin_fee bigint NOT NULL DEFAULT 0,
  expires_at timestamptz NOT NULL,
  stock_held boolean NOT NULL DEFAULT false,
  paid_at timestamptz,
  owner_deadline_at timestamptz,
  reminder_sent_at timestamptz,
  responded_at timestamptz,
  reject_reason text,
  rejected_by varchar(10) CHECK (rejected_by IN ('owner', 'system')),
  cancelled_by varchar(10) CHECK (cancelled_by IN ('tenant', 'system')),
  ended_at timestamptz,
  deposit_returned_at timestamptz,
  deposit_return_note text,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_bookings_status_expires_at ON bookings (status, expires_at);
CREATE INDEX idx_bookings_status_owner_deadline_at ON bookings (status, owner_deadline_at);
CREATE INDEX idx_bookings_tenant_id_created_at ON bookings (tenant_id, created_at DESC);

CREATE TABLE user_notifications (
  id bigserial PRIMARY KEY,
  user_id bigint NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  category varchar(10) NOT NULL CHECK (category IN ('booking', 'payment', 'promo', 'chat')),
  title varchar(255) NOT NULL,
  body text NOT NULL,
  link varchar(255),
  read_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_user_notifications_user_id_created_at ON user_notifications (user_id, created_at DESC);
CREATE INDEX idx_user_notifications_unread ON user_notifications (user_id) WHERE read_at IS NULL;

CREATE TABLE access_tokens (
  id bigserial PRIMARY KEY,
  user_id bigint NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  token_hash char(64) NOT NULL UNIQUE,
  last_used_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE access_tokens;
DROP TABLE user_notifications;
DROP TABLE bookings;
DROP TABLE rooms;
DROP TABLE properties;
DROP TABLE areas;
DROP TABLE users;
