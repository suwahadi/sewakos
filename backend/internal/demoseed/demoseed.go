package demoseed

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"math/rand/v2"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/suwahadi/sewakos/backend/internal/media"
	"github.com/suwahadi/sewakos/backend/internal/platform/postgres/dbgen"
)

const (
	photoCacheControl = "public, max-age=31536000, immutable"
	maxSourceBytes    = 15 << 20
)

type ObjectStore interface {
	Put(ctx context.Context, key string, data []byte, contentType, cacheControl string) error
}

type ImageProcessor interface {
	Process(ctx context.Context, data []byte) (media.Image, error)
}

type Deps struct {
	DB           *pgxpool.Pool
	Storage      ObjectStore
	Images       ImageProcessor
	HTTP         *http.Client
	PhotoBaseURL string
	Logger       *slog.Logger
}

type Options struct {
	Count      int
	SlugPrefix string
}

type placement struct {
	city   cityDef
	spot   spot
	areaID int64
}

func Run(ctx context.Context, deps Deps, opts Options) (int, error) {
	photos, err := loadPhotos(ctx, deps)
	if err != nil {
		return 0, err
	}
	ownerIDs, placements, err := upsertOwnersAndAreas(ctx, deps)
	if err != nil {
		return 0, err
	}

	created := 0
	for i := 1; i <= opts.Count; i++ {
		slug := fmt.Sprintf("%s-%03d", opts.SlugPrefix, i)
		exists, err := dbgen.New(deps.DB).DemoPropertyExists(ctx, slug)
		if err != nil {
			return created, fmt.Errorf("check %s: %w", slug, err)
		}
		if exists {
			continue
		}
		r := rand.New(rand.NewPCG(uint64(i), 0x5e3a))
		p := placements[(i-1)%len(placements)]
		if err := createProperty(ctx, deps, r, slug, ownerIDs[(i-1)%len(ownerIDs)], p, photos); err != nil {
			return created, fmt.Errorf("create %s: %w", slug, err)
		}
		created++
		if created%10 == 0 {
			deps.Logger.Info("demo seed progress", "created", created)
		}
	}
	return created, nil
}

func loadPhotos(ctx context.Context, deps Deps) (map[photoKind][]media.Image, error) {
	photos := map[photoKind][]media.Image{}
	for kind, ids := range photoSources {
		for _, id := range ids {
			original, err := download(ctx, deps, fmt.Sprintf("%s/photo-%s?w=2000&q=85&fm=jpg", deps.PhotoBaseURL, id))
			if err != nil {
				return nil, fmt.Errorf("download unsplash photo %s: %w", id, err)
			}
			image, err := deps.Images.Process(ctx, original)
			if err != nil {
				return nil, fmt.Errorf("process unsplash photo %s: %w", id, err)
			}
			photos[kind] = append(photos[kind], image)
		}
	}
	return photos, nil
}

func download(ctx context.Context, deps Deps, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := deps.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, maxSourceBytes))
}

func upsertOwnersAndAreas(ctx context.Context, deps Deps) ([]int64, []placement, error) {
	tx, err := deps.DB.Begin(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := dbgen.New(tx)

	ownerIDs := make([]int64, 0, len(demoOwners))
	for _, o := range demoOwners {
		id, err := q.DemoUpsertOwner(ctx, dbgen.DemoUpsertOwnerParams{Name: o.name, Phone: o.phone})
		if err != nil {
			return nil, nil, fmt.Errorf("upsert owner %s: %w", o.phone, err)
		}
		if err := q.DemoUpsertOwnerBankAccount(ctx, dbgen.DemoUpsertOwnerBankAccountParams{
			UserID: id, BankName: o.bank, AccountNumber: o.account, AccountName: o.name,
		}); err != nil {
			return nil, nil, fmt.Errorf("upsert bank account %s: %w", o.phone, err)
		}
		ownerIDs = append(ownerIDs, id)
	}

	upsertArea := func(a areaDef) (int64, error) {
		return q.DemoUpsertArea(ctx, dbgen.DemoUpsertAreaParams{
			Name: a.name, Slug: a.slug, Type: a.kind, IsTop: a.isTop, Latitude: a.lat, Longitude: a.lng,
		})
	}
	var placements []placement
	for _, c := range cities {
		cityID, err := upsertArea(c.area)
		if err != nil {
			return nil, nil, fmt.Errorf("upsert area %s: %w", c.area.slug, err)
		}
		for _, s := range c.spots {
			if s.landmark != nil {
				if _, err := upsertArea(*s.landmark); err != nil {
					return nil, nil, fmt.Errorf("upsert area %s: %w", s.landmark.slug, err)
				}
			}
			placements = append(placements, placement{city: c, spot: s, areaID: cityID})
		}
	}
	return ownerIDs, placements, tx.Commit(ctx)
}

func pick[T any](r *rand.Rand, items []T) T {
	return items[r.IntN(len(items))]
}

func roundTo(value float64, step int64) int64 {
	return int64(math.Round(value/float64(step))) * step
}

func createProperty(ctx context.Context, deps Deps, r *rand.Rand, slug string, ownerID int64, p placement, photos map[photoKind][]media.Image) error {
	tx, err := deps.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := dbgen.New(tx)

	gender := pick(r, []string{"campur", "putra", "putri"})
	neighborhood := pick(r, p.spot.neighborhoods)
	discount := int64(0)
	flashSale := r.IntN(100) < 15
	if flashSale {
		discount = 100_000 + int64(r.IntN(5))*50_000
	}
	propertyID, err := q.DemoCreateProperty(ctx, dbgen.DemoCreatePropertyParams{
		OwnerID: ownerID, AreaID: p.areaID,
		Name:        fmt.Sprintf("%s %s %s", pick(r, namePrefixes), pick(r, nameWords), neighborhood),
		Slug:        slug,
		Gender:      gender,
		Description: description(r, gender, neighborhood, p),
		Address:     fmt.Sprintf("%s No. %d, %s, %s, %s", pick(r, p.spot.streets), 1+r.IntN(150), neighborhood, p.spot.district, p.city.area.name),
		Latitude:    p.spot.lat + (r.Float64()*2-1)*0.012,
		Longitude:   p.spot.lng + (r.Float64()*2-1)*0.012,
		IsFlashSale: flashSale, FlashSaleDiscount: discount,
		Views: int64(20 + r.IntN(900)),
	})
	if err != nil {
		return err
	}

	put := func(kind photoKind, index int, roomID pgtype.Int8, caption string) error {
		pool := photos[kind]
		image := pool[index%len(pool)]
		keyBase := fmt.Sprintf("properties/%d/%s", propertyID, uuid.NewString())
		largeKey, thumbKey := keyBase+".webp", keyBase+"_thumb.webp"
		if err := deps.Storage.Put(ctx, largeKey, image.Large, "image/webp", photoCacheControl); err != nil {
			return err
		}
		if err := deps.Storage.Put(ctx, thumbKey, image.Thumb, "image/webp", photoCacheControl); err != nil {
			return err
		}
		_, err := q.InsertPropertyPhoto(ctx, dbgen.InsertPropertyPhotoParams{
			PropertyID: propertyID, RoomID: roomID, ObjectKey: largeKey, ThumbKey: thumbKey,
			Width: int32(image.Width), Height: int32(image.Height), SizeBytes: int32(len(image.Large)),
			Caption: pgtype.Text{String: caption, Valid: caption != ""},
		})
		return err
	}

	noRoom := pgtype.Int8{}
	common := r.IntN(len(photos[kindCommon]))
	for _, shot := range []struct {
		kind  photoKind
		index int
	}{{kindExterior, r.IntN(len(photos[kindExterior]))}, {kindCommon, common}, {kindBathroom, r.IntN(len(photos[kindBathroom]))}, {kindCommon, common + 1}} {
		if err := put(shot.kind, shot.index, noRoom, photoCaptions[shot.kind]); err != nil {
			return err
		}
	}

	basePrice := p.city.priceMin + int64(r.Float64()*float64(p.city.priceMax-p.city.priceMin))
	bedroom := r.IntN(len(photos[kindBedroom]))
	for _, tier := range roomTiers[:1+r.IntN(len(roomTiers))] {
		price := roundTo(float64(basePrice)*tier.multiplier, 50_000)
		var weekly, daily pgtype.Int8
		if r.IntN(10) < 4 {
			weekly = pgtype.Int8{Int64: roundTo(float64(price)/3.3, 10_000), Valid: true}
		}
		if r.IntN(10) < 3 {
			daily = pgtype.Int8{Int64: roundTo(float64(price)/22, 5_000), Valid: true}
		}
		capacity := int16(1)
		if tier.bedrooms > 1 && r.IntN(2) == 0 {
			capacity = 2
		}
		facilities, err := json.Marshal(tier.facilities)
		if err != nil {
			return err
		}
		roomID, err := q.DemoCreateRoom(ctx, dbgen.DemoCreateRoomParams{
			PropertyID: propertyID, Name: tier.name, PriceMonthly: price, PriceWeekly: weekly, PriceDaily: daily,
			Available: int16(r.IntN(5)), Capacity: capacity,
			Facilities:             facilities,
			MonthlyDurationOptions: []byte(pick(r, durationChoices)),
		})
		if err != nil {
			return err
		}
		for range tier.bedrooms {
			if err := put(kindBedroom, bedroom, pgtype.Int8{Int64: roomID, Valid: true}, tier.name); err != nil {
				return err
			}
			bedroom++
		}
	}
	return tx.Commit(ctx)
}

func description(r *rand.Rand, gender, neighborhood string, p placement) string {
	kind := map[string]string{"campur": "Kost campur", "putra": "Kost khusus putra", "putri": "Kost khusus putri"}[gender]
	parts := []string{fmt.Sprintf("%s yang bersih dan nyaman di kawasan %s, %s.", kind, neighborhood, p.city.area.name)}
	if p.spot.landmark != nil {
		parts = append(parts, fmt.Sprintf("Hanya %d menit dari %s.", 3+r.IntN(10), p.spot.landmark.name))
	} else {
		parts = append(parts, "Dekat minimarket, tempat makan, dan akses transportasi umum.")
	}
	shared := make([]string, 0, 3)
	for _, i := range r.Perm(len(sharedFacilities))[:3] {
		shared = append(shared, sharedFacilities[i])
	}
	parts = append(parts, fmt.Sprintf("Fasilitas bersama: %s.", strings.Join(shared, ", ")))
	if gender == "campur" {
		parts = append(parts, "Penjaga kost tinggal di lokasi, akses 24 jam dengan kunci pribadi.")
	} else {
		parts = append(parts, "Lingkungan tenang dengan aturan jam tamu, penjaga kost tinggal di lokasi.")
	}
	parts = append(parts, "Cocok untuk mahasiswa dan karyawan.")
	return strings.Join(parts, " ")
}
