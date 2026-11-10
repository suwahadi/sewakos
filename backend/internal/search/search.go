package search

import (
	"context"

	"github.com/suwahadi/sewakos/backend/internal/platform/postgres/dbgen"
)

// Store is the subset of dbgen.Querier this package needs.
type Store interface {
	SuggestAreas(ctx context.Context, keyword string) ([]dbgen.SuggestAreasRow, error)
	SuggestLandings(ctx context.Context, keyword string) ([]dbgen.SuggestLandingsRow, error)
	SuggestRoomTypes(ctx context.Context, keyword string) ([]dbgen.SuggestRoomTypesRow, error)
	GetAreaByID(ctx context.Context, id int64) (dbgen.GetAreaByIDRow, error)
	SearchRooms(ctx context.Context, arg dbgen.SearchRoomsParams) ([]dbgen.SearchRoomsRow, error)
	CountSearchRooms(ctx context.Context, arg dbgen.CountSearchRoomsParams) (dbgen.CountSearchRoomsRow, error)
}

// Deps bundles this package's shared dependencies.
type Deps struct {
	Store          Store
	AreaRadiusKM   float64
	StorageBaseURL string
}
