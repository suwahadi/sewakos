package listing

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/suwahadi/sewakos/backend/internal/platform/postgres/dbgen"
)

// Store is the subset of dbgen.Querier this package needs for reads.
type Store interface {
	GetListingProperty(ctx context.Context, id int64) (dbgen.GetListingPropertyRow, error)
	ListListingRooms(ctx context.Context, propertyID int64) ([]dbgen.ListListingRoomsRow, error)
	ListListingPropertyPhotos(ctx context.Context, propertyID int64) ([]dbgen.ListListingPropertyPhotosRow, error)
	ListListingRoomPhotos(ctx context.Context, propertyID int64) ([]dbgen.ListListingRoomPhotosRow, error)
	ExistsFavorite(ctx context.Context, arg dbgen.ExistsFavoriteParams) (bool, error)
	ListGalleryRooms(ctx context.Context, propertyID int64) ([]dbgen.ListGalleryRoomsRow, error)
}

// TxBeginner is the subset of *pgxpool.Pool the view counter needs to update
// properties.views and property_views_daily atomically (SPEC-GO.md §20).
type TxBeginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

// Deps bundles this package's shared dependencies.
type Deps struct {
	Store          Store
	DB             TxBeginner
	StorageBaseURL string
}

func photoURL(base, key string) string {
	return base + "/" + key
}

func nullableText(t pgtype.Text) any {
	if !t.Valid {
		return nil
	}
	return t.String
}
