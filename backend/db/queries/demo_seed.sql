-- name: DemoUpsertOwner :one
INSERT INTO users (name, role, phone, phone_verified_at)
VALUES (sqlc.arg(name), 'owner', sqlc.arg(phone), now())
ON CONFLICT (phone) DO UPDATE SET name = EXCLUDED.name
RETURNING id;

-- name: DemoUpsertOwnerBankAccount :exec
INSERT INTO owner_bank_accounts (user_id, bank_name, account_number, account_name)
VALUES (sqlc.arg(user_id), sqlc.arg(bank_name), sqlc.arg(account_number), sqlc.arg(account_name))
ON CONFLICT (user_id) DO NOTHING;

-- name: DemoUpsertArea :one
INSERT INTO areas (name, slug, type, is_top, latitude, longitude)
VALUES (sqlc.arg(name), sqlc.arg(slug), sqlc.arg(type), sqlc.arg(is_top), sqlc.arg(latitude), sqlc.arg(longitude))
ON CONFLICT (slug) DO UPDATE SET name = EXCLUDED.name
RETURNING id;

-- name: DemoPropertyExists :one
SELECT EXISTS (SELECT 1 FROM properties WHERE slug = sqlc.arg(slug));

-- name: DemoCreateProperty :one
INSERT INTO properties (
  owner_id, area_id, name, slug, gender, description, address, latitude, longitude,
  is_active, moderation_status, moderated_at, is_flash_sale, flash_sale_discount, views
) VALUES (
  sqlc.arg(owner_id), sqlc.arg(area_id), sqlc.arg(name), sqlc.arg(slug), sqlc.arg(gender), sqlc.arg(description),
  sqlc.arg(address), sqlc.arg(latitude), sqlc.arg(longitude), true, 'approved', now(),
  sqlc.arg(is_flash_sale), sqlc.arg(flash_sale_discount), sqlc.arg(views)
)
RETURNING id;

-- name: DemoCreateRoom :one
INSERT INTO rooms (property_id, name, price_monthly, price_weekly, price_daily, available, capacity, facilities, monthly_duration_options)
VALUES (
  sqlc.arg(property_id), sqlc.arg(name), sqlc.arg(price_monthly), sqlc.narg(price_weekly), sqlc.narg(price_daily),
  sqlc.arg(available), sqlc.arg(capacity), sqlc.arg(facilities), sqlc.arg(monthly_duration_options)
)
RETURNING id;
