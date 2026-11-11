package admin

import (
	"encoding/json"
	"net/http"
	"net/netip"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/suwahadi/sewakos/backend/internal/authctx"
	"github.com/suwahadi/sewakos/backend/internal/money"
	"github.com/suwahadi/sewakos/backend/internal/platform/postgres/dbgen"
	"github.com/suwahadi/sewakos/backend/internal/response"
)

func optionalWIBDate(r *http.Request, key string) pgtype.Timestamptz {
	v := r.URL.Query().Get(key)
	if v == "" {
		return pgtype.Timestamptz{}
	}
	t, err := time.ParseInLocation(time.DateOnly, v, money.WIB)
	if err != nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: t, Valid: true}
}

func auditActorBody(actorID pgtype.Int8, actorName, actorRole pgtype.Text) any {
	if !actorID.Valid {
		return nil
	}
	return map[string]any{"id": actorID.Int64, "name": actorName.String, "role": actorRole.String}
}

func nullableIP(ip *netip.Addr) any {
	if ip == nil {
		return nil
	}
	return ip.String()
}

func AuditLogsHandler(deps Deps, _ authctx.User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, limit := pageParam(r), limitParam(r)
		to := optionalWIBDate(r, "to")
		if to.Valid {
			to.Time = to.Time.AddDate(0, 0, 1)
		}
		rows, err := deps.Store.ListAdminAuditLogs(r.Context(), dbgen.ListAdminAuditLogsParams{
			ActorID: queryID(r, "actor_id"), Action: queryText(r, "action"), EntityType: queryText(r, "entity_type"),
			EntityID: queryID(r, "entity_id"), FromTime: optionalWIBDate(r, "from"), ToTime: to,
			RowLimit: limit, RowOffset: (page - 1) * limit,
		})
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		logs := make([]map[string]any, len(rows))
		for i, row := range rows {
			var changes map[string]any
			_ = json.Unmarshal(row.Changes, &changes)
			logs[i] = map[string]any{
				"id": row.ID, "created_at": money.FormatExpiresAt(row.CreatedAt.Time),
				"actor":  auditActorBody(row.ActorID, row.ActorName, row.ActorRole),
				"action": row.Action, "entity_type": row.EntityType, "entity_id": row.EntityID, "changes": changes,
				"ip": nullableIP(row.Ip), "user_agent": nullableText(row.UserAgent),
			}
		}
		response.Success(w, http.StatusOK, "", map[string]any{"audit_logs": logs, "current_page": page})
	}
}
