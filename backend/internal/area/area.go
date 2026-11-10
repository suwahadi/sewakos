package area

import (
	"context"

	"github.com/suwahadi/sewakos/backend/internal/platform/postgres/dbgen"
)

var validTypes = map[string]bool{"city": true, "campus": true, "mrt": true}

// Store is the subset of dbgen.Querier this package needs.
type Store interface {
	ListAreasByType(ctx context.Context, arg dbgen.ListAreasByTypeParams) ([]dbgen.ListAreasByTypeRow, error)
}

// Deps bundles this package's shared dependencies.
type Deps struct {
	Store Store
}

func areaResponse(row dbgen.ListAreasByTypeRow) map[string]any {
	return map[string]any{
		"id": row.ID, "name": row.Name, "slug": row.Slug,
		"latitude": row.Latitude, "longitude": row.Longitude,
	}
}
