package admin

import (
	"context"
	"errors"
	"net/http"
	"regexp"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/suwahadi/sewakos/backend/internal/authctx"
	"github.com/suwahadi/sewakos/backend/internal/platform/postgres/dbgen"
	"github.com/suwahadi/sewakos/backend/internal/response"
	"github.com/suwahadi/sewakos/backend/internal/validation"
)

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

func areaFromRow(row dbgen.AdminListAreasRow) dbgen.AdminUpdateAreaParams {
	return dbgen.AdminUpdateAreaParams{
		ID: row.ID, Name: row.Name, Slug: row.Slug, Type: row.Type, IsTop: row.IsTop,
		Latitude: row.Latitude, Longitude: row.Longitude, PlaceID: row.PlaceID,
	}
}

func areaBody(a dbgen.AdminUpdateAreaParams) map[string]any {
	return map[string]any{
		"id": a.ID, "name": a.Name, "slug": a.Slug, "type": a.Type, "is_top": a.IsTop,
		"latitude": a.Latitude, "longitude": a.Longitude, "place_id": nullableText(a.PlaceID),
	}
}

func validateArea(ctx context.Context, store dbgen.Querier, body map[string]any, areaID int64, required bool) (*validation.Validator, error) {
	v := validation.New()
	field := func(name string) *validation.FieldValidator {
		f := v.Field(name, body[name])
		if required {
			return f.Required()
		}
		return f.Optional()
	}
	field("name").String().MaxStringLen(255)
	slugField := field("slug").String().Regex(slugPattern)
	if slug, ok := body["slug"].(string); ok && slug != "" {
		taken, err := store.AreaSlugTaken(ctx, dbgen.AreaSlugTakenParams{Slug: slug, ExceptID: areaID})
		if err != nil {
			return nil, err
		}
		slugField.Unique(taken)
	}
	field("type").In("city", "campus", "mrt")
	v.Field("is_top", body["is_top"]).Optional().Boolean()
	field("latitude").Numeric().BetweenNumber(-90, 90)
	field("longitude").Numeric().BetweenNumber(-180, 180)
	v.Field("place_id", body["place_id"]).Optional().String().MaxStringLen(255)
	return v, nil
}

func applyAreaChanges(area dbgen.AdminUpdateAreaParams, body map[string]any) dbgen.AdminUpdateAreaParams {
	for field, target := range map[string]*string{"name": &area.Name, "slug": &area.Slug, "type": &area.Type} {
		if value, ok := body[field].(string); ok && value != "" {
			*target = value
		}
	}
	if isTop, ok := body["is_top"].(bool); ok {
		area.IsTop = isTop
	}
	if latitude, ok := body["latitude"].(float64); ok {
		area.Latitude = latitude
	}
	if longitude, ok := body["longitude"].(float64); ok {
		area.Longitude = longitude
	}
	if value, sent := body["place_id"]; sent {
		placeID, _ := value.(string)
		area.PlaceID = pgtype.Text{String: placeID, Valid: placeID != ""}
	}
	return area
}

func ListAreasHandler(deps Deps, _ authctx.User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, limit := pageParam(r), limitParam(r)
		rows, err := deps.Store.AdminListAreas(r.Context(), dbgen.AdminListAreasParams{
			Type: queryText(r, "type"), RowLimit: limit, RowOffset: (page - 1) * limit,
		})
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		areas := make([]map[string]any, len(rows))
		for i, row := range rows {
			areas[i] = areaBody(areaFromRow(row))
		}
		response.Success(w, http.StatusOK, "", map[string]any{"areas": areas, "current_page": page})
	}
}

func CreateAreaHandler(deps Deps, user authctx.User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		body := decodeBody(r)
		v, err := validateArea(ctx, deps.Store, body, 0, true)
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		if !v.Valid() {
			response.Fail(w, http.StatusUnprocessableEntity, response.MsgValidationFailed, v.Errors())
			return
		}
		area := applyAreaChanges(dbgen.AdminUpdateAreaParams{}, body)

		tx, err := deps.DB.Begin(ctx)
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		defer func() { _ = tx.Rollback(ctx) }()
		queries := dbgen.New(tx)
		area.ID, err = queries.AdminInsertArea(ctx, dbgen.AdminInsertAreaParams{
			Name: area.Name, Slug: area.Slug, Type: area.Type, IsTop: area.IsTop,
			Latitude: area.Latitude, Longitude: area.Longitude, PlaceID: area.PlaceID,
		})
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		if err := recordAudit(ctx, queries, r, user, "area.create", "area", area.ID, nil, nil, areaBody(area)); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		if err := tx.Commit(ctx); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		response.Success(w, http.StatusOK, "Area ditambahkan.", map[string]any{"area": areaBody(area)})
	}
}

func UpdateAreaHandler(deps Deps, user authctx.User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		areaID, ok := parseID(r, "area")
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

		current, err := queries.AdminLockArea(ctx, areaID)
		if errors.Is(err, pgx.ErrNoRows) {
			response.NotFoundHandler(w, r)
			return
		}
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		body := decodeBody(r)
		v, err := validateArea(ctx, deps.Store, body, areaID, false)
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		if !v.Valid() {
			response.Fail(w, http.StatusUnprocessableEntity, response.MsgValidationFailed, v.Errors())
			return
		}
		before := areaFromRow(dbgen.AdminListAreasRow(current))
		after := applyAreaChanges(before, body)
		if err := queries.AdminUpdateArea(ctx, after); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		if err := recordAudit(ctx, queries, r, user, "area.update", "area", areaID, nil, areaBody(before), areaBody(after)); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		if err := tx.Commit(ctx); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		response.Success(w, http.StatusOK, "Area diperbarui.", map[string]any{"area": areaBody(after)})
	}
}

func DeleteAreaHandler(deps Deps, user authctx.User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		areaID, ok := parseID(r, "area")
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

		current, err := queries.AdminLockArea(ctx, areaID)
		if errors.Is(err, pgx.ErrNoRows) {
			response.NotFoundHandler(w, r)
			return
		}
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		inUse, err := queries.AreaInUse(ctx, areaID)
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		if inUse {
			response.Fail(w, http.StatusBadRequest, "Area masih dipakai properti.", nil)
			return
		}
		if err := queries.AdminDeleteArea(ctx, areaID); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		if err := recordAudit(ctx, queries, r, user, "area.delete", "area", areaID, nil, areaBody(areaFromRow(dbgen.AdminListAreasRow(current))), nil); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		if err := tx.Commit(ctx); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		response.Success(w, http.StatusOK, "Area dihapus.", nil)
	}
}
