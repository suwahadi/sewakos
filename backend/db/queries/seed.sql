-- name: CountAreasBySlug :one
SELECT count(*) FROM areas WHERE slug = $1;

-- name: SeedCreateUser :one
INSERT INTO users (name, role, phone, password, phone_verified_at)
VALUES ($1, $2, $3, $4, now())
RETURNING id;

-- name: SeedUpsertSetting :exec
INSERT INTO settings (key, value) VALUES ($1, $2)
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = now();

-- name: SeedCreateOwnerBankAccount :exec
INSERT INTO owner_bank_accounts (user_id, bank_name, account_number, account_name)
VALUES ($1, $2, $3, $4);

-- name: SeedCreateArea :one
INSERT INTO areas (name, slug, type, is_top, latitude, longitude)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id;

-- name: SeedCreateProperty :one
INSERT INTO properties (
  owner_id, area_id, name, slug, gender, description, address, latitude, longitude,
  is_active, moderation_status, is_flash_sale, flash_sale_discount
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9, true, 'approved', $10, $11
)
RETURNING id;

-- name: SeedCreateRoom :one
INSERT INTO rooms (property_id, name, price_monthly, available, capacity, facilities)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id;
