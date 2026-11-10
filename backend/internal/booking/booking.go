package booking

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/suwahadi/sewakos/backend/internal/platform/postgres/dbgen"
)

type Store interface {
	GetRoomForDraftPrice(ctx context.Context, id int64) (dbgen.GetRoomForDraftPriceRow, error)
}

type AdminFee interface {
	Get(ctx context.Context) (int64, error)
}

type TxBeginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

type Deps struct {
	Store    Store
	DB       TxBeginner
	AdminFee AdminFee
}
