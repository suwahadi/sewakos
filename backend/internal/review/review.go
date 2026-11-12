package review

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/suwahadi/sewakos/backend/internal/authctx"
	"github.com/suwahadi/sewakos/backend/internal/money"
	"github.com/suwahadi/sewakos/backend/internal/notification"
	"github.com/suwahadi/sewakos/backend/internal/platform/postgres/dbgen"
	"github.com/suwahadi/sewakos/backend/internal/response"
	"github.com/suwahadi/sewakos/backend/internal/validation"
)

const (
	defaultLimit = 20
	maxLimit     = 50

	msgAlreadyReviewed = "Booking ini sudah diulas."
	msgNotReviewable   = "Booking belum bisa diulas."
)

type TxBeginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

type Deps struct {
	Store        dbgen.Querier
	DB           TxBeginner
	Ratings      *RatingCache
	AppPublicURL string
}

func parseID(r *http.Request, param string) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, param), 10, 64)
	return id, err == nil && id > 0
}

func pageParam(r *http.Request) int32 {
	n, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || n < 1 {
		return 1
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

func MaskTenantName(name string) string {
	words := strings.Fields(name)
	if len(words) < 2 {
		return name
	}
	initial := []rune(words[1])[0]
	return words[0] + " " + string(initial) + "."
}

func Body(id int64, rating int16, comment pgtype.Text, tenantName, roomName string, createdAt time.Time) map[string]any {
	var commentValue any
	if comment.Valid {
		commentValue = comment.String
	}
	return map[string]any{
		"id": id, "rating": rating, "comment": commentValue,
		"tenant_name": tenantName, "room_name": roomName,
		"created_at": money.FormatTimestamp(createdAt),
	}
}

func reviewable(status string, startDate time.Time) bool {
	switch status {
	case "completed", "terminated":
		return true
	case "approved":
		return startDate.Format(time.DateOnly) <= money.TodayWIB().Format(time.DateOnly)
	default:
		return false
	}
}

func CreateHandler(deps Deps, user authctx.User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bookingID, ok := parseID(r, "booking")
		if !ok {
			response.NotFoundHandler(w, r)
			return
		}
		ctx := r.Context()

		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		v := validation.New()
		v.Field("rating", body["rating"]).Required().Integer().BetweenNumber(1, 5)
		v.Field("comment", body["comment"]).Optional().String().MaxStringLen(1000)
		if !v.Valid() {
			response.Fail(w, http.StatusUnprocessableEntity, response.MsgValidationFailed, v.Errors())
			return
		}
		ratingValue, _ := body["rating"].(float64)
		rating := int16(math.Trunc(ratingValue))
		commentText, _ := body["comment"].(string)
		comment := pgtype.Text{String: commentText, Valid: commentText != ""}

		tx, err := deps.DB.Begin(ctx)
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		committed := false
		defer func() {
			if !committed {
				_ = tx.Rollback(ctx)
			}
		}()
		queries := dbgen.New(tx)

		booking, err := queries.LockBookingForReview(ctx, bookingID)
		if errors.Is(err, pgx.ErrNoRows) {
			response.NotFoundHandler(w, r)
			return
		}
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		if booking.TenantID != user.UserID {
			response.Fail(w, http.StatusForbidden, response.MsgAccessDenied, nil)
			return
		}
		if booking.Reviewed {
			response.Fail(w, http.StatusBadRequest, msgAlreadyReviewed, nil)
			return
		}
		if !reviewable(booking.Status, booking.StartDate.Time) {
			response.Fail(w, http.StatusBadRequest, msgNotReviewable, nil)
			return
		}

		created, err := queries.InsertReview(ctx, dbgen.InsertReviewParams{
			BookingID: booking.ID, PropertyID: booking.PropertyID, TenantID: user.UserID, Rating: rating, Comment: comment,
		})
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}

		tenant, err := queries.GetUserByID(ctx, user.UserID)
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		owner, err := queries.GetUserByID(ctx, booking.OwnerID)
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		var ownerEmail *string
		if owner.Email.Valid {
			ownerEmail = &owner.Email.String
		}
		link := "/owner/ulasan"
		if _, err := notification.Create(ctx, queries, deps.AppPublicURL, notification.Params{
			UserID: booking.OwnerID, RecipientEmail: ownerEmail, Category: notification.CategoryBooking,
			Title: "Ulasan baru", Body: fmt.Sprintf("%s mendapat ulasan bintang %d.", booking.PropertyName, rating),
			Link: &link,
		}); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}

		if err := tx.Commit(ctx); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		committed = true
		deps.Ratings.Invalidate(ctx, booking.PropertyID)

		response.Success(w, http.StatusOK, "Terima kasih atas ulasannya.", map[string]any{
			"review": Body(created.ID, rating, comment, MaskTenantName(tenant.Name), booking.RoomName, created.CreatedAt.Time),
		})
	}
}

func ListHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		propertyID, ok := parseID(r, "property")
		if !ok {
			response.NotFoundHandler(w, r)
			return
		}
		ctx := r.Context()

		listed, err := deps.Store.ExistsListedProperty(ctx, propertyID)
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		if !listed {
			response.NotFoundHandler(w, r)
			return
		}

		rating, err := deps.Ratings.Get(ctx, deps.Store, propertyID)
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}

		page, limit := pageParam(r), limitParam(r)
		rows, err := deps.Store.ListPropertyReviews(ctx, dbgen.ListPropertyReviewsParams{
			PropertyID: propertyID, RowLimit: limit, RowOffset: (page - 1) * limit,
		})
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		reviews := make([]map[string]any, len(rows))
		for i, row := range rows {
			reviews[i] = Body(row.ID, row.Rating, row.Comment, MaskTenantName(row.TenantName), row.RoomName, row.CreatedAt.Time)
		}
		response.Success(w, http.StatusOK, "", map[string]any{
			"rating": rating.Body(), "reviews": reviews, "current_page": page,
		})
	}
}
