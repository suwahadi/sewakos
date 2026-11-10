-- name: GetRoomForDraftPrice :one
SELECT r.name, r.price_monthly, r.price_weekly, r.price_daily,
       p.is_flash_sale, p.flash_sale_discount
FROM rooms r
JOIN properties p ON p.id = r.property_id
WHERE r.id = sqlc.arg(id);
