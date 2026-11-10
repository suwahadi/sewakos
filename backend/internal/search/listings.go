package search

import (
	"encoding/json"
	"math"
	"net/http"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/suwahadi/sewakos/backend/internal/money"
	"github.com/suwahadi/sewakos/backend/internal/platform/postgres/dbgen"
	"github.com/suwahadi/sewakos/backend/internal/response"
	"github.com/suwahadi/sewakos/backend/internal/validation"
)

const (
	defaultLimit = 20
	maxLimit     = 50
	earthRadiusK = 6371.0
)

var genderCodes = map[int]string{0: "campur", 1: "putra", 2: "putri"}

func asFloatSlice(v any) ([]float64, bool) {
	arr, ok := v.([]any)
	if !ok {
		return nil, false
	}
	out := make([]float64, len(arr))
	for i, item := range arr {
		n, ok := item.(float64)
		if !ok {
			return nil, false
		}
		out[i] = n
	}
	return out, true
}

func SearchHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Filters struct {
				Gender       []any `json:"gender"`
				PriceRange   []any `json:"price_range"`
				RentType     any   `json:"rent_type"`
				PropertyType any   `json:"property_type"`
				FlashSale    any   `json:"flash_sale"`
				IsAvailable  any   `json:"is_available"`
				AreaID       any   `json:"area_id"`
			} `json:"filters"`
			Sorting struct {
				Field     any `json:"field"`
				Direction any `json:"direction"`
			} `json:"sorting"`
			Limit  any `json:"limit"`
			Offset any `json:"offset"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		ctx := r.Context()

		v := validation.New()
		rentType := int16(3)
		if body.Filters.RentType != nil {
			n, ok := body.Filters.RentType.(float64)
			switch {
			case !ok:
				v.Field("filters.rent_type", body.Filters.RentType).Integer()
			case n < 1 || n > 3:
				v.Field("filters.rent_type", body.Filters.RentType).BetweenNumber(1, 3)
			default:
				rentType = int16(n)
			}
		}

		propertyType, _ := body.Filters.PropertyType.(string)
		if body.Filters.PropertyType != nil {
			v.Field("filters.property_type", body.Filters.PropertyType).String()
		}

		var areaID pgtype.Int8
		if body.Filters.AreaID != nil {
			v.Field("filters.area_id", body.Filters.AreaID).Integer()
			if n, ok := body.Filters.AreaID.(float64); ok {
				areaID = pgtype.Int8{Int64: int64(n), Valid: true}
			}
		}

		sortField, _ := body.Sorting.Field.(string)
		sortDir, _ := body.Sorting.Direction.(string)
		if sortField != "price" && sortField != "availability" {
			sortField = ""
		}
		if sortDir != "-" && sortDir != "+" {
			sortDir = ""
		}

		limit := int32(defaultLimit)
		if n, ok := body.Limit.(float64); ok {
			limit = int32(n)
		}
		if limit < 1 {
			limit = 1
		}
		if limit > maxLimit {
			limit = maxLimit
		}
		offset := int32(0)
		if n, ok := body.Offset.(float64); ok && n >= 0 {
			offset = int32(n)
		}

		var hasGenderFilter pgtype.Bool
		var genders []string
		if len(body.Filters.Gender) > 0 {
			hasGenderFilter = pgtype.Bool{Bool: true, Valid: true}
			genders = make([]string, 0, len(body.Filters.Gender))
			for _, g := range body.Filters.Gender {
				n, ok := g.(float64)
				if !ok {
					continue
				}
				if label, ok := genderCodes[int(n)]; ok {
					genders = append(genders, label)
				}
			}
		}

		var priceMin, priceMax pgtype.Int8
		if nums, ok := asFloatSlice(body.Filters.PriceRange); ok && len(nums) == 2 {
			priceMin = pgtype.Int8{Int64: int64(nums[0]), Valid: true}
			priceMax = pgtype.Int8{Int64: int64(nums[1]), Valid: true}
		}

		var flashSale pgtype.Bool
		if b, ok := body.Filters.FlashSale.(bool); ok {
			flashSale = pgtype.Bool{Bool: b, Valid: b}
		}
		var isAvailable pgtype.Bool
		if b, ok := body.Filters.IsAvailable.(bool); ok && b {
			isAvailable = pgtype.Bool{Bool: true, Valid: true}
		}

		var areaType pgtype.Text
		var areaLat, areaLng, areaLatMin, areaLatMax, areaLngMin, areaLngMax, areaRadiusKM pgtype.Float8
		if areaID.Valid {
			areaRow, err := deps.Store.GetAreaByID(ctx, areaID.Int64)
			if err != nil {
				response.Fail(w, http.StatusUnprocessableEntity, response.MsgValidationFailed,
					map[string][]string{"filters.area_id": {"The selected filters.area_id is invalid."}})
				return
			}
			areaType = pgtype.Text{String: areaRow.Type, Valid: true}
			if areaRow.Type != "city" {
				radiusKM := deps.AreaRadiusKM
				areaLat = pgtype.Float8{Float64: areaRow.Latitude, Valid: true}
				areaLng = pgtype.Float8{Float64: areaRow.Longitude, Valid: true}
				areaRadiusKM = pgtype.Float8{Float64: radiusKM, Valid: true}
				latDelta := radiusKM / 111.0
				lngDelta := radiusKM / (111.0 * math.Cos(areaRow.Latitude*math.Pi/180))
				areaLatMin = pgtype.Float8{Float64: areaRow.Latitude - latDelta, Valid: true}
				areaLatMax = pgtype.Float8{Float64: areaRow.Latitude + latDelta, Valid: true}
				areaLngMin = pgtype.Float8{Float64: areaRow.Longitude - math.Abs(lngDelta), Valid: true}
				areaLngMax = pgtype.Float8{Float64: areaRow.Longitude + math.Abs(lngDelta), Valid: true}
			}
		}

		if !v.Valid() {
			response.Fail(w, http.StatusUnprocessableEntity, response.MsgValidationFailed, v.Errors())
			return
		}

		if propertyType == "apartemen" {
			response.Success(w, http.StatusOK, "", map[string]any{
				"source": "db", "property_total": 0, "total": 0, "limit": limit,
				"has-more": false, "next-offset": nil, "rooms": []map[string]any{},
			})
			return
		}

		counts, err := deps.Store.CountSearchRooms(ctx, dbgen.CountSearchRoomsParams{
			RentType: rentType, HasGenderFilter: hasGenderFilter, Genders: genders,
			PriceMin: priceMin, PriceMax: priceMax, FlashSale: flashSale, IsAvailable: isAvailable,
			AreaID: areaID, AreaType: areaType,
			AreaLatMin: areaLatMin, AreaLatMax: areaLatMax, AreaLngMin: areaLngMin, AreaLngMax: areaLngMax,
			AreaLat: areaLat, AreaLng: areaLng, AreaRadiusKm: areaRadiusKM,
		})
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}

		rows, err := deps.Store.SearchRooms(ctx, dbgen.SearchRoomsParams{
			RentType: rentType, HasGenderFilter: hasGenderFilter, Genders: genders,
			PriceMin: priceMin, PriceMax: priceMax, FlashSale: flashSale, IsAvailable: isAvailable,
			AreaID: areaID, AreaType: areaType,
			AreaLatMin: areaLatMin, AreaLatMax: areaLatMax, AreaLngMin: areaLngMin, AreaLngMax: areaLngMax,
			AreaLat: areaLat, AreaLng: areaLng, AreaRadiusKm: areaRadiusKM,
			SortField: sortField, SortDir: sortDir, ResultOffset: offset, ResultLimit: limit,
		})
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}

		rooms := make([]map[string]any, len(rows))
		for i, row := range rows {
			var thumbnailURL any
			if row.ThumbnailKey != "" {
				thumbnailURL = deps.StorageBaseURL + "/" + row.ThumbnailKey
			}
			var ratingAverage any
			if row.RatingCount > 0 {
				ratingAverage = row.RatingAverage
			}
			rooms[i] = map[string]any{
				"id": row.ID, "property_id": row.PropertyID, "name": row.RoomName,
				"property_name": row.PropertyName, "area_name": row.AreaName,
				"thumbnail_url": thumbnailURL, "gender": row.Gender,
				"price": row.ActivePrice, "price_label": money.FormatRupiah(row.ActivePrice),
				"available": row.Available, "is_flash_sale": row.IsFlashSale,
				"rating_average": ratingAverage, "rating_count": row.RatingCount,
			}
		}

		hasMore := int64(offset)+int64(len(rows)) < counts.Total
		var nextOffset any
		if hasMore {
			nextOffset = offset + limit
		}

		response.Success(w, http.StatusOK, "", map[string]any{
			"source": "db", "property_total": counts.PropertyTotal, "total": counts.Total,
			"limit": limit, "has-more": hasMore, "next-offset": nextOffset, "rooms": rooms,
		})
	}
}
