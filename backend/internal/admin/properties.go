package admin

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/suwahadi/sewakos/backend/internal/authctx"
	"github.com/suwahadi/sewakos/backend/internal/money"
	"github.com/suwahadi/sewakos/backend/internal/notification"
	"github.com/suwahadi/sewakos/backend/internal/owner"
	"github.com/suwahadi/sewakos/backend/internal/platform/postgres/dbgen"
	"github.com/suwahadi/sewakos/backend/internal/response"
	"github.com/suwahadi/sewakos/backend/internal/validation"
)

const msgModerationNotAllowed = "Status properti tidak sesuai untuk tindakan ini."

type moderation struct {
	action    string
	from      []string
	to        string
	needsNote bool
	message   string
	title     string
	body      func(property, reason string) string
}

var (
	approveModeration = moderation{
		action: "property.approve", from: []string{"pending", "suspended"}, to: "approved", message: "Properti disetujui.",
		title: "Properti disetujui", body: func(property, _ string) string { return property + " sudah disetujui dan dapat tayang." },
	}
	rejectModeration = moderation{
		action: "property.reject", from: []string{"pending"}, to: "rejected", needsNote: true, message: "Properti ditolak.",
		title: "Properti ditolak", body: func(property, reason string) string { return property + " ditolak admin. Alasan: " + reason },
	}
	suspendModeration = moderation{
		action: "property.suspend", from: []string{"approved"}, to: "suspended", needsNote: true, message: "Properti dinonaktifkan.",
		title: "Properti dinonaktifkan", body: func(property, reason string) string { return property + " dinonaktifkan admin. Alasan: " + reason },
	}
)

func loadProperties(ctx context.Context, deps Deps, params dbgen.AdminListPropertiesParams) ([]map[string]any, error) {
	rows, err := deps.Store.AdminListProperties(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("list properties: %w", err)
	}
	ids := make([]int64, len(rows))
	for i, row := range rows {
		ids[i] = row.ID
	}
	photos, err := deps.Store.ListPhotosForProperties(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("list property photos: %w", err)
	}
	photosByProperty := map[int64][]map[string]any{}
	for _, ph := range photos {
		photosByProperty[ph.PropertyID] = append(photosByProperty[ph.PropertyID], map[string]any{
			"id": ph.ID, "url": deps.StorageBaseURL + "/" + ph.ObjectKey, "thumbnail_url": deps.StorageBaseURL + "/" + ph.ThumbKey,
			"caption": nullableText(ph.Caption), "room_id": nullableInt8(ph.RoomID), "sort_order": ph.SortOrder,
		})
	}

	properties := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		propertyPhotos := photosByProperty[row.ID]
		if propertyPhotos == nil {
			propertyPhotos = []map[string]any{}
		}
		var ratingAverage any
		if row.RatingCount > 0 {
			ratingAverage = row.RatingAverage
		}
		properties = append(properties, map[string]any{
			"id": row.ID, "name": row.Name, "slug": row.Slug, "gender": row.Gender,
			"description": row.Description, "address": row.Address, "latitude": row.Latitude, "longitude": row.Longitude,
			"area":      map[string]any{"id": row.AreaID, "name": row.AreaName},
			"is_active": row.IsActive, "moderation_status": row.ModerationStatus, "moderation_note": nullableText(row.ModerationNote),
			"is_flash_sale": row.IsFlashSale, "flash_sale_discount": row.FlashSaleDiscount,
			"flash_sale_discount_label": money.FormatRupiah(row.FlashSaleDiscount),
			"views":                     row.Views, "rooms_count": row.RoomsCount, "photos": propertyPhotos,
			"rating":     map[string]any{"average": ratingAverage, "count": row.RatingCount},
			"created_at": money.FormatTimestamp(row.CreatedAt.Time),
			"owner":      map[string]any{"id": row.OwnerID, "name": row.OwnerName, "phone_number": row.OwnerPhone},
		})
	}
	return properties, nil
}

func nullableInt8(n pgtype.Int8) any {
	if !n.Valid {
		return nil
	}
	return n.Int64
}

func ListPropertiesHandler(deps Deps, _ authctx.User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, limit := pageParam(r), limitParam(r)
		properties, err := loadProperties(r.Context(), deps, dbgen.AdminListPropertiesParams{
			ModerationStatus: queryText(r, "moderation_status"), Q: queryText(r, "q"), RowLimit: limit, RowOffset: (page - 1) * limit,
		})
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		response.Success(w, http.StatusOK, "", map[string]any{"properties": properties, "current_page": page})
	}
}

func GetPropertyHandler(deps Deps, _ authctx.User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		propertyID, ok := parseID(r, "property")
		if !ok {
			response.NotFoundHandler(w, r)
			return
		}
		properties, err := loadProperties(r.Context(), deps, dbgen.AdminListPropertiesParams{
			PropertyID: pgtype.Int8{Int64: propertyID, Valid: true}, RowLimit: 1,
		})
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		if len(properties) == 0 {
			response.NotFoundHandler(w, r)
			return
		}
		response.Success(w, http.StatusOK, "", map[string]any{"property": properties[0]})
	}
}

func ApprovePropertyHandler(deps Deps, user authctx.User) http.HandlerFunc {
	return moderate(deps, user, approveModeration)
}

func RejectPropertyHandler(deps Deps, user authctx.User) http.HandlerFunc {
	return moderate(deps, user, rejectModeration)
}

func SuspendPropertyHandler(deps Deps, user authctx.User) http.HandlerFunc {
	return moderate(deps, user, suspendModeration)
}

func moderate(deps Deps, user authctx.User, m moderation) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		propertyID, ok := parseID(r, "property")
		if !ok {
			response.NotFoundHandler(w, r)
			return
		}
		ctx := r.Context()
		tx, err := deps.DB.Begin(ctx)
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		defer func() { _ = tx.Rollback(ctx) }()
		queries := dbgen.New(tx)

		property, err := queries.AdminLockProperty(ctx, propertyID)
		if errors.Is(err, pgx.ErrNoRows) {
			response.NotFoundHandler(w, r)
			return
		}
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		if !slices.Contains(m.from, property.ModerationStatus) {
			response.Fail(w, http.StatusBadRequest, msgModerationNotAllowed, nil)
			return
		}

		var reason string
		if m.needsNote {
			body := decodeBody(r)
			v := validation.New()
			v.Field("reason", body["reason"]).Required().String().MaxStringLen(500)
			if !v.Valid() {
				response.Fail(w, http.StatusUnprocessableEntity, response.MsgValidationFailed, v.Errors())
				return
			}
			reason = body["reason"].(string)
		}
		if err := queries.AdminSetModeration(ctx, dbgen.AdminSetModerationParams{
			ID: property.ID, ModerationStatus: m.to, ModerationNote: pgtype.Text{String: reason, Valid: reason != ""},
		}); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		if err := owner.NotifyUser(ctx, queries, deps.AppPublicURL, property.OwnerID, notification.CategoryBooking, m.title,
			m.body(property.Name, reason), fmt.Sprintf("/owner/properti/%d", property.ID)); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		if err := recordAudit(ctx, queries, r, user, m.action, "property", property.ID, &property.OwnerID,
			map[string]any{"moderation_status": property.ModerationStatus},
			map[string]any{"moderation_status": m.to, "moderation_note": reason}); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		if err := tx.Commit(ctx); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		response.Success(w, http.StatusOK, m.message, nil)
	}
}
