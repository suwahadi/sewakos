package owner

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/suwahadi/sewakos/backend/internal/authctx"
	"github.com/suwahadi/sewakos/backend/internal/money"
	"github.com/suwahadi/sewakos/backend/internal/platform/postgres/dbgen"
	"github.com/suwahadi/sewakos/backend/internal/response"
	"github.com/suwahadi/sewakos/backend/internal/validation"
)

const (
	defaultPage  = 1
	defaultLimit = 20
	maxLimit     = 50
)

var slugSeparators = regexp.MustCompile(`[^a-z0-9]+`)

func propertySlug(name string, id int64) string {
	base := strings.Trim(slugSeparators.ReplaceAllString(strings.ToLower(name), "-"), "-")
	if base == "" {
		return strconv.FormatInt(id, 10)
	}
	return fmt.Sprintf("%s-%d", base, id)
}

func pageParam(r *http.Request) int32 {
	n, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || n < 1 {
		return defaultPage
	}
	return int32(n)
}

func limitParam(r *http.Request) int32 {
	n, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil || n < 1 {
		return defaultLimit
	}
	return int32(min(n, maxLimit))
}

func optionalQueryText(r *http.Request, key string) pgtype.Text {
	value := r.URL.Query().Get(key)
	return pgtype.Text{String: value, Valid: value != ""}
}

func photoBody(baseURL string, id int64, objectKey, thumbKey string, caption pgtype.Text, roomID pgtype.Int8, sortOrder int32) map[string]any {
	return map[string]any{
		"id": id, "url": photoURL(baseURL, objectKey), "thumbnail_url": photoURL(baseURL, thumbKey),
		"caption": nullableText(caption), "room_id": nullableInt8(roomID), "sort_order": sortOrder,
	}
}

func loadOwnerProperties(ctx context.Context, store dbgen.Querier, baseURL string, params dbgen.ListOwnerPropertiesParams) ([]map[string]any, error) {
	rows, err := store.ListOwnerProperties(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("list owner properties: %w", err)
	}
	ids := make([]int64, len(rows))
	for i, row := range rows {
		ids[i] = row.ID
	}
	photos, err := store.ListPhotosForProperties(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("list property photos: %w", err)
	}
	photosByProperty := map[int64][]map[string]any{}
	for _, ph := range photos {
		photosByProperty[ph.PropertyID] = append(photosByProperty[ph.PropertyID],
			photoBody(baseURL, ph.ID, ph.ObjectKey, ph.ThumbKey, ph.Caption, ph.RoomID, ph.SortOrder))
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
		})
	}
	return properties, nil
}

func loadOwnerProperty(ctx context.Context, deps Deps, ownerID, propertyID int64) (map[string]any, error) {
	properties, err := loadOwnerProperties(ctx, deps.Store, deps.StorageBaseURL, dbgen.ListOwnerPropertiesParams{
		OwnerID: ownerID, PropertyID: pgtype.Int8{Int64: propertyID, Valid: true}, RowLimit: 1,
	})
	if err != nil {
		return nil, err
	}
	if len(properties) == 0 {
		return nil, pgx.ErrNoRows
	}
	return properties[0], nil
}

func validateProperty(ctx context.Context, store dbgen.Querier, body map[string]any, required bool) (*validation.Validator, error) {
	v := validation.New()
	field := func(name string) *validation.FieldValidator {
		f := v.Field(name, body[name])
		if required {
			return f.Required()
		}
		return f.Optional()
	}
	areaField := field("area_id").Integer()
	if areaID, ok := wholeNumber(body["area_id"]); ok {
		_, err := store.GetAreaByID(ctx, areaID)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("look up area %d: %w", areaID, err)
		}
		areaField.Exists(err == nil)
	}
	field("name").String().MaxStringLen(255)
	field("gender").In("campur", "putra", "putri")
	field("description").String().MaxStringLen(5000)
	field("address").String().MaxStringLen(500)
	field("latitude").Numeric().BetweenNumber(-90, 90)
	field("longitude").Numeric().BetweenNumber(-180, 180)
	return v, nil
}

func editableSnapshot(p dbgen.UpdatePropertyParams) map[string]any {
	return map[string]any{
		"area_id": p.AreaID, "name": p.Name, "gender": p.Gender, "description": p.Description, "address": p.Address,
		"latitude": p.Latitude, "longitude": p.Longitude, "is_active": p.IsActive,
		"is_flash_sale": p.IsFlashSale, "flash_sale_discount": p.FlashSaleDiscount,
	}
}

func ListPropertiesHandler(deps Deps, user authctx.User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, limit := pageParam(r), limitParam(r)
		properties, err := loadOwnerProperties(r.Context(), deps.Store, deps.StorageBaseURL, dbgen.ListOwnerPropertiesParams{
			OwnerID: user.UserID, ModerationStatus: optionalQueryText(r, "moderation_status"),
			RowLimit: limit, RowOffset: (page - 1) * limit,
		})
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		response.Success(w, http.StatusOK, "", map[string]any{"properties": properties, "current_page": page})
	}
}

func GetPropertyHandler(deps Deps, user authctx.User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		propertyID, ok := parseID(r, "property")
		if !ok {
			response.NotFoundHandler(w, r)
			return
		}
		respondWithProperty(w, r, deps, user.UserID, propertyID, "")
	}
}

func respondWithProperty(w http.ResponseWriter, r *http.Request, deps Deps, ownerID, propertyID int64, message string) {
	property, err := loadOwnerProperty(r.Context(), deps, ownerID, propertyID)
	if errors.Is(err, pgx.ErrNoRows) {
		response.NotFoundHandler(w, r)
		return
	}
	if err != nil {
		response.InternalErrorHandler(w, r)
		return
	}
	response.Success(w, http.StatusOK, message, map[string]any{"property": property})
}

func CreatePropertyHandler(deps Deps, user authctx.User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		body := decodeBody(r)
		v, err := validateProperty(ctx, deps.Store, body, true)
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		if !v.Valid() {
			response.Fail(w, http.StatusUnprocessableEntity, response.MsgValidationFailed, v.Errors())
			return
		}

		areaID, _ := wholeNumber(body["area_id"])
		insert := dbgen.InsertPropertyParams{
			OwnerID: user.UserID, AreaID: areaID, Name: body["name"].(string), Slug: uuid.NewString(),
			Gender: body["gender"].(string), Description: body["description"].(string), Address: body["address"].(string),
			Latitude: body["latitude"].(float64), Longitude: body["longitude"].(float64),
		}
		propertyID, err := insertProperty(ctx, deps, r, user, insert)
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		respondWithProperty(w, r, deps, user.UserID, propertyID, "Properti dibuat dan menunggu persetujuan admin.")
	}
}

func insertProperty(ctx context.Context, deps Deps, r *http.Request, user authctx.User, insert dbgen.InsertPropertyParams) (int64, error) {
	tx, err := deps.DB.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := dbgen.New(tx)

	id, err := queries.InsertProperty(ctx, insert)
	if err != nil {
		return 0, fmt.Errorf("insert property: %w", err)
	}
	slug := propertySlug(insert.Name, id)
	if err := queries.SetPropertySlug(ctx, dbgen.SetPropertySlugParams{ID: id, Slug: slug}); err != nil {
		return 0, fmt.Errorf("set slug for property %d: %w", id, err)
	}
	after := editableSnapshot(dbgen.UpdatePropertyParams{
		AreaID: insert.AreaID, Name: insert.Name, Gender: insert.Gender, Description: insert.Description,
		Address: insert.Address, Latitude: insert.Latitude, Longitude: insert.Longitude, IsActive: true,
	})
	after["slug"], after["moderation_status"] = slug, "pending"
	if err := recordAudit(ctx, queries, r, user, "property.create", "property", id, nil, after); err != nil {
		return 0, err
	}
	return id, tx.Commit(ctx)
}

func UpdatePropertyHandler(deps Deps, user authctx.User) http.HandlerFunc {
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

		current, err := queries.LockOwnerPropertyForEdit(ctx, dbgen.LockOwnerPropertyForEditParams{ID: propertyID, OwnerID: user.UserID})
		if errors.Is(err, pgx.ErrNoRows) {
			response.NotFoundHandler(w, r)
			return
		}
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		if current.ModerationStatus == "suspended" {
			response.Fail(w, http.StatusBadRequest, "Properti sedang dinonaktifkan admin.", nil)
			return
		}

		body := decodeBody(r)
		v, err := validateProperty(ctx, deps.Store, body, false)
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		v.Field("is_active", body["is_active"]).Optional().Boolean()
		v.Field("is_flash_sale", body["is_flash_sale"]).Optional().Boolean()
		v.Field("flash_sale_discount", body["flash_sale_discount"]).Optional().Integer().MinNumber(0)
		if !v.Valid() {
			response.Fail(w, http.StatusUnprocessableEntity, response.MsgValidationFailed, v.Errors())
			return
		}

		before := dbgen.UpdatePropertyParams{
			ID: current.ID, AreaID: current.AreaID, Name: current.Name, Gender: current.Gender,
			Description: current.Description, Address: current.Address, Latitude: current.Latitude,
			Longitude: current.Longitude, IsActive: current.IsActive, IsFlashSale: current.IsFlashSale,
			FlashSaleDiscount: current.FlashSaleDiscount,
		}
		after := applyPropertyChanges(before, body)
		if err := queries.UpdateProperty(ctx, after); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		if err := recordAudit(ctx, queries, r, user, "property.update", "property", current.ID, editableSnapshot(before), editableSnapshot(after)); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		if err := tx.Commit(ctx); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		respondWithProperty(w, r, deps, user.UserID, propertyID, "Properti diperbarui.")
	}
}

func applyPropertyChanges(update dbgen.UpdatePropertyParams, body map[string]any) dbgen.UpdatePropertyParams {
	if areaID, ok := wholeNumber(body["area_id"]); ok {
		update.AreaID = areaID
	}
	for field, target := range map[string]*string{
		"name": &update.Name, "gender": &update.Gender, "description": &update.Description, "address": &update.Address,
	} {
		if value, ok := body[field].(string); ok && value != "" {
			*target = value
		}
	}
	if latitude, ok := body["latitude"].(float64); ok {
		update.Latitude = latitude
	}
	if longitude, ok := body["longitude"].(float64); ok {
		update.Longitude = longitude
	}
	if isActive, ok := body["is_active"].(bool); ok {
		update.IsActive = isActive
	}
	if isFlashSale, ok := body["is_flash_sale"].(bool); ok {
		update.IsFlashSale = isFlashSale
	}
	if discount, ok := wholeNumber(body["flash_sale_discount"]); ok {
		update.FlashSaleDiscount = discount
	}
	return update
}

func SubmitPropertyHandler(deps Deps, user authctx.User) http.HandlerFunc {
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

		current, err := queries.LockOwnerPropertyForEdit(ctx, dbgen.LockOwnerPropertyForEditParams{ID: propertyID, OwnerID: user.UserID})
		if errors.Is(err, pgx.ErrNoRows) {
			response.NotFoundHandler(w, r)
			return
		}
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		if current.ModerationStatus != "rejected" {
			response.Fail(w, http.StatusBadRequest, "Properti tidak dapat diajukan ulang.", nil)
			return
		}
		if err := queries.ResubmitProperty(ctx, current.ID); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		before := map[string]any{"moderation_status": current.ModerationStatus, "moderation_note": nullableText(current.ModerationNote)}
		after := map[string]any{"moderation_status": "pending", "moderation_note": nil}
		if err := recordAudit(ctx, queries, r, user, "property.submit", "property", current.ID, before, after); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		if err := tx.Commit(ctx); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		respondWithProperty(w, r, deps, user.UserID, propertyID, "Properti diajukan ulang ke admin.")
	}
}
