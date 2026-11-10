package listing

import (
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/suwahadi/sewakos/backend/internal/money"
	"github.com/suwahadi/sewakos/backend/internal/platform/postgres/dbgen"
	"github.com/suwahadi/sewakos/backend/internal/response"
)

func ViewHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		propertyID, ok := parsePropertyID(r)
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
		views, err := queries.IncrementPropertyView(ctx, propertyID)
		if err != nil {
			if err == pgx.ErrNoRows {
				response.NotFoundHandler(w, r)
				return
			}
			response.InternalErrorHandler(w, r)
			return
		}

		today := money.TodayWIB()
		if err := queries.UpsertPropertyViewDaily(ctx, dbgen.UpsertPropertyViewDailyParams{
			PropertyID: propertyID,
			ViewDate:   pgtype.Date{Time: today, Valid: true},
		}); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}

		if err := tx.Commit(ctx); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}

		response.Success(w, http.StatusOK, "", map[string]any{"views": views})
	}
}
