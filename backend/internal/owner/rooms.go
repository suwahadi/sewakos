package owner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/suwahadi/sewakos/backend/internal/authctx"
	"github.com/suwahadi/sewakos/backend/internal/platform/postgres/dbgen"
	"github.com/suwahadi/sewakos/backend/internal/response"
	"github.com/suwahadi/sewakos/backend/internal/validation"
)

const msgPropertySuspended = "Properti sedang dinonaktifkan admin."

func loadOwnerRooms(ctx context.Context, deps Deps, params dbgen.ListOwnerRoomsParams) ([]map[string]any, error) {
	rows, err := deps.Store.ListOwnerRooms(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("list owner rooms: %w", err)
	}
	ids := make([]int64, len(rows))
	for i, row := range rows {
		ids[i] = row.ID
	}
	photos, err := deps.Store.ListRoomPhotos(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("list room photos: %w", err)
	}
	photosByRoom := map[int64][]map[string]any{}
	for _, ph := range photos {
		photosByRoom[ph.RoomID.Int64] = append(photosByRoom[ph.RoomID.Int64],
			photoBody(deps.StorageBaseURL, ph.ID, ph.ObjectKey, ph.ThumbKey, ph.Caption, ph.RoomID, ph.SortOrder))
	}

	rooms := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		roomPhotos := photosByRoom[row.ID]
		if roomPhotos == nil {
			roomPhotos = []map[string]any{}
		}
		rooms = append(rooms, map[string]any{
			"id": row.ID, "property_id": row.PropertyID, "name": row.Name,
			"price_monthly": row.PriceMonthly, "price_weekly": nullableInt8(row.PriceWeekly), "price_daily": nullableInt8(row.PriceDaily),
			"available": row.Available, "held": row.Held, "occupied": row.Occupied, "capacity": row.Capacity,
			"monthly_duration_options": json.RawMessage(row.MonthlyDurationOptions), "facilities": json.RawMessage(row.Facilities),
			"photos": roomPhotos,
		})
	}
	return rooms, nil
}

func respondWithRoom(w http.ResponseWriter, r *http.Request, deps Deps, ownerID, roomID int64, message string) {
	rooms, err := loadOwnerRooms(r.Context(), deps, dbgen.ListOwnerRoomsParams{OwnerID: ownerID, RoomID: pgtype.Int8{Int64: roomID, Valid: true}})
	if err != nil || len(rooms) == 0 {
		response.InternalErrorHandler(w, r)
		return
	}
	response.Success(w, http.StatusOK, message, map[string]any{"room": rooms[0]})
}

func validateRoom(body map[string]any, required bool) *validation.Validator {
	v := validation.New()
	field := func(name string) *validation.FieldValidator {
		f := v.Field(name, body[name])
		if required {
			return f.Required()
		}
		return f.Optional()
	}
	field("name").String().MaxStringLen(255)
	field("price_monthly").Integer().MinNumber(1)
	v.Field("price_weekly", body["price_weekly"]).Optional().Integer().MinNumber(1)
	v.Field("price_daily", body["price_daily"]).Optional().Integer().MinNumber(1)
	field("available").Integer().BetweenNumber(0, 255)
	field("capacity").Integer().BetweenNumber(1, 20)
	field("facilities").Array().MaxArrayLen(30)
	if items, ok := body["facilities"].([]any); ok {
		for i, item := range items {
			v.Field(fmt.Sprintf("facilities.%d", i), item).String().MaxStringLen(50)
		}
	}
	v.Field("monthly_duration_options", body["monthly_duration_options"]).Optional().Array().MaxArrayLen(6)
	if items, ok := body["monthly_duration_options"].([]any); ok {
		seen := map[float64]bool{}
		for i, item := range items {
			n, isNumber := item.(float64)
			v.Field(fmt.Sprintf("monthly_duration_options.%d", i), item).Integer().BetweenNumber(1, 12).Distinct(isNumber && seen[n])
			if isNumber {
				seen[n] = true
			}
		}
	}
	return v
}

func facilitiesJSON(value any) ([]byte, bool) {
	items, ok := value.([]any)
	if !ok {
		return nil, false
	}
	facilities := make([]string, 0, len(items))
	for _, item := range items {
		facilities = append(facilities, item.(string))
	}
	encoded, err := json.Marshal(facilities)
	return encoded, err == nil
}

func durationsJSON(value any) ([]byte, bool) {
	items, ok := value.([]any)
	if !ok || len(items) == 0 {
		return nil, false
	}
	durations := make([]int64, 0, len(items))
	for _, item := range items {
		n, _ := wholeNumber(item)
		durations = append(durations, n)
	}
	slices.Sort(durations)
	encoded, err := json.Marshal(durations)
	return encoded, err == nil
}

func optionalPrice(value any) pgtype.Int8 {
	n, ok := wholeNumber(value)
	return pgtype.Int8{Int64: n, Valid: ok}
}

func roomSnapshot(p dbgen.UpdateRoomParams) map[string]any {
	return map[string]any{
		"name": p.Name, "price_monthly": p.PriceMonthly, "price_weekly": nullableInt8(p.PriceWeekly),
		"price_daily": nullableInt8(p.PriceDaily), "available": p.Available, "capacity": p.Capacity,
		"facilities": string(p.Facilities), "monthly_duration_options": string(p.MonthlyDurationOptions),
	}
}

func ListRoomsHandler(deps Deps, user authctx.User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		propertyID, ok := parseID(r, "property")
		if !ok {
			response.NotFoundHandler(w, r)
			return
		}
		ctx := r.Context()
		owned, err := deps.Store.OwnerPropertyExists(ctx, dbgen.OwnerPropertyExistsParams{ID: propertyID, OwnerID: user.UserID})
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		if !owned {
			response.NotFoundHandler(w, r)
			return
		}
		rooms, err := loadOwnerRooms(ctx, deps, dbgen.ListOwnerRoomsParams{OwnerID: user.UserID, PropertyID: pgtype.Int8{Int64: propertyID, Valid: true}})
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		response.Success(w, http.StatusOK, "", map[string]any{"rooms": rooms})
	}
}

func CreateRoomHandler(deps Deps, user authctx.User) http.HandlerFunc {
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

		property, err := queries.LockOwnerPropertyForEdit(ctx, dbgen.LockOwnerPropertyForEditParams{ID: propertyID, OwnerID: user.UserID})
		if errors.Is(err, pgx.ErrNoRows) {
			response.NotFoundHandler(w, r)
			return
		}
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		if property.ModerationStatus == "suspended" {
			response.Fail(w, http.StatusBadRequest, msgPropertySuspended, nil)
			return
		}

		body := decodeBody(r)
		if v := validateRoom(body, true); !v.Valid() {
			response.Fail(w, http.StatusUnprocessableEntity, response.MsgValidationFailed, v.Errors())
			return
		}
		facilities, _ := facilitiesJSON(body["facilities"])
		durations, ok := durationsJSON(body["monthly_duration_options"])
		if !ok {
			durations = []byte("[1]")
		}
		priceMonthly, _ := wholeNumber(body["price_monthly"])
		available, _ := wholeNumber(body["available"])
		capacity, _ := wholeNumber(body["capacity"])
		insert := dbgen.InsertRoomParams{
			PropertyID: propertyID, Name: body["name"].(string), PriceMonthly: priceMonthly,
			PriceWeekly: optionalPrice(body["price_weekly"]), PriceDaily: optionalPrice(body["price_daily"]),
			Available: int16(available), Capacity: int16(capacity), Facilities: facilities, MonthlyDurationOptions: durations,
		}
		roomID, err := queries.InsertRoom(ctx, insert)
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		after := roomSnapshot(dbgen.UpdateRoomParams{
			Name: insert.Name, PriceMonthly: insert.PriceMonthly, PriceWeekly: insert.PriceWeekly, PriceDaily: insert.PriceDaily,
			Available: insert.Available, Capacity: insert.Capacity, Facilities: insert.Facilities, MonthlyDurationOptions: insert.MonthlyDurationOptions,
		})
		after["property_id"] = propertyID
		if err := recordAudit(ctx, queries, r, user, "room.create", "room", roomID, nil, after); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		if err := tx.Commit(ctx); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		respondWithRoom(w, r, deps, user.UserID, roomID, "Kamar ditambahkan.")
	}
}

func UpdateRoomHandler(deps Deps, user authctx.User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roomID, ok := parseID(r, "room")
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

		current, err := queries.LockOwnerRoom(ctx, dbgen.LockOwnerRoomParams{ID: roomID, OwnerID: user.UserID})
		if errors.Is(err, pgx.ErrNoRows) {
			response.NotFoundHandler(w, r)
			return
		}
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}

		body := decodeBody(r)
		if v := validateRoom(body, false); !v.Valid() {
			response.Fail(w, http.StatusUnprocessableEntity, response.MsgValidationFailed, v.Errors())
			return
		}
		before := dbgen.UpdateRoomParams{
			ID: current.ID, Name: current.Name, PriceMonthly: current.PriceMonthly, PriceWeekly: current.PriceWeekly,
			PriceDaily: current.PriceDaily, Available: current.Available, Capacity: current.Capacity,
			Facilities: current.Facilities, MonthlyDurationOptions: current.MonthlyDurationOptions,
		}
		after := applyRoomChanges(before, body)
		if err := queries.UpdateRoom(ctx, after); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		if err := recordAudit(ctx, queries, r, user, "room.update", "room", current.ID, roomSnapshot(before), roomSnapshot(after)); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		if err := tx.Commit(ctx); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		respondWithRoom(w, r, deps, user.UserID, roomID, "Kamar diperbarui.")
	}
}

func applyRoomChanges(update dbgen.UpdateRoomParams, body map[string]any) dbgen.UpdateRoomParams {
	if name, ok := body["name"].(string); ok && name != "" {
		update.Name = name
	}
	if price, ok := wholeNumber(body["price_monthly"]); ok {
		update.PriceMonthly = price
	}
	for field, target := range map[string]*pgtype.Int8{"price_weekly": &update.PriceWeekly, "price_daily": &update.PriceDaily} {
		if value, sent := body[field]; sent && (value == nil || optionalPrice(value).Valid) {
			*target = optionalPrice(value)
		}
	}
	if available, ok := wholeNumber(body["available"]); ok {
		update.Available = int16(available)
	}
	if capacity, ok := wholeNumber(body["capacity"]); ok {
		update.Capacity = int16(capacity)
	}
	if facilities, ok := facilitiesJSON(body["facilities"]); ok {
		update.Facilities = facilities
	}
	if durations, ok := durationsJSON(body["monthly_duration_options"]); ok {
		update.MonthlyDurationOptions = durations
	}
	return update
}

func DeleteRoomHandler(deps Deps, user authctx.User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roomID, ok := parseID(r, "room")
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

		room, err := queries.LockOwnerRoom(ctx, dbgen.LockOwnerRoomParams{ID: roomID, OwnerID: user.UserID})
		if errors.Is(err, pgx.ErrNoRows) {
			response.NotFoundHandler(w, r)
			return
		}
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		hasBookings, err := queries.RoomHasBookings(ctx, room.ID)
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		if hasBookings {
			response.Fail(w, http.StatusBadRequest, "Kamar sudah memiliki booking dan tidak dapat dihapus.", nil)
			return
		}
		if err := queries.DeleteRoom(ctx, room.ID); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		before := roomSnapshot(dbgen.UpdateRoomParams{
			Name: room.Name, PriceMonthly: room.PriceMonthly, PriceWeekly: room.PriceWeekly, PriceDaily: room.PriceDaily,
			Available: room.Available, Capacity: room.Capacity, Facilities: room.Facilities, MonthlyDurationOptions: room.MonthlyDurationOptions,
		})
		before["property_id"] = room.PropertyID
		if err := recordAudit(ctx, queries, r, user, "room.delete", "room", room.ID, before, nil); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		if err := tx.Commit(ctx); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		response.Success(w, http.StatusOK, "Kamar dihapus.", nil)
	}
}
