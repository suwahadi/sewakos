package booking

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/suwahadi/sewakos/backend/internal/platform/postgres/dbgen"
)

// Store is the subset of dbgen.Querier this package needs.
type Store interface {
	GetRoomForDraftPrice(ctx context.Context, id int64) (dbgen.GetRoomForDraftPriceRow, error)
}

// AdminFee supplies the cached admin_fee setting (SPEC-GO.md §16).
type AdminFee interface {
	Get(ctx context.Context) (int64, error)
}

// TxBeginner is the subset of *pgxpool.Pool the transactional handlers need.
type TxBeginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

// Deps bundles this package's shared dependencies.
type Deps struct {
	Store    Store
	DB       TxBeginner
	AdminFee AdminFee
}
