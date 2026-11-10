// Command seed loads the dev/test fixture data described in SPEC-GO.md §10.
// It refuses to run against production and is idempotent: if the seed's
// marker area (slug "kota-depok") already exists, it exits without making
// any changes rather than reconciling partial state (docs/DECISIONS.md).
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"

	"github.com/suwahadi/sewakos/backend/internal/config"
	"github.com/suwahadi/sewakos/backend/internal/platform/postgres"
	"github.com/suwahadi/sewakos/backend/internal/platform/postgres/dbgen"
)

func pgText(s string) pgtype.Text {
	return pgtype.Text{String: s, Valid: true}
}

const seedPassword = "password123"
const markerAreaSlug = "kota-depok"

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("load config", "error", err)
		os.Exit(1)
	}
	if cfg.AppEnv == "production" {
		logger.Error("refusing to seed: APP_ENV=production")
		os.Exit(1)
	}

	ctx := context.Background()
	pool, err := postgres.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("connect postgres", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	q := dbgen.New(pool)

	already, err := q.CountAreasBySlug(ctx, markerAreaSlug)
	if err != nil {
		logger.Error("check existing seed", "error", err)
		os.Exit(1)
	}
	if already > 0 {
		logger.Info("already seeded, nothing to do")
		return
	}

	if err := seed(ctx, q); err != nil {
		logger.Error("seed", "error", err)
		os.Exit(1)
	}
	logger.Info("seed complete")
}

func seed(ctx context.Context, q *dbgen.Queries) error {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(seedPassword), 12)
	if err != nil {
		return fmt.Errorf("hash seed password: %w", err)
	}
	hash := pgText(string(passwordHash))

	ownerID, err := q.SeedCreateUser(ctx, dbgen.SeedCreateUserParams{
		Name: "Pemilik Contoh", Role: "owner", Phone: "081200000001", Password: hash,
	})
	if err != nil {
		return fmt.Errorf("seed owner: %w", err)
	}
	tenantID, err := q.SeedCreateUser(ctx, dbgen.SeedCreateUserParams{
		Name: "Anak Kos Contoh", Role: "tenant", Phone: "081200000002", Password: hash,
	})
	if err != nil {
		return fmt.Errorf("seed tenant: %w", err)
	}
	if _, err := q.SeedCreateUser(ctx, dbgen.SeedCreateUserParams{
		Name: "Admin Contoh", Role: "admin", Phone: "081200000000", Password: hash,
	}); err != nil {
		return fmt.Errorf("seed admin: %w", err)
	}

	if err := q.SeedUpsertSetting(ctx, dbgen.SeedUpsertSettingParams{Key: "admin_fee", Value: []byte("0")}); err != nil {
		return fmt.Errorf("seed settings: %w", err)
	}

	if err := q.SeedCreateOwnerBankAccount(ctx, dbgen.SeedCreateOwnerBankAccountParams{
		UserID: ownerID, BankName: "BCA", AccountNumber: "1234567890", AccountName: "Pemilik Contoh",
	}); err != nil {
		return fmt.Errorf("seed owner bank account: %w", err)
	}

	areaID, err := q.SeedCreateArea(ctx, dbgen.SeedCreateAreaParams{
		Name: "Kota Depok", Slug: markerAreaSlug, Type: "city", IsTop: true,
		Latitude: -6.3878486, Longitude: 106.8177948,
	})
	if err != nil {
		return fmt.Errorf("seed area: %w", err)
	}

	property1ID, err := q.SeedCreateProperty(ctx, dbgen.SeedCreatePropertyParams{
		OwnerID: ownerID, AreaID: areaID, Name: "Kost Asti Beji", Slug: "kost-asti-beji-123",
		Gender: "campur", Description: "Kost dekat stasiun UI Depok", Address: "Jl. Contoh No. 42",
		Latitude: -6.38, Longitude: 106.81, IsFlashSale: true, FlashSaleDiscount: 250000,
	})
	if err != nil {
		return fmt.Errorf("seed property 1: %w", err)
	}
	if _, err := q.SeedCreateRoom(ctx, dbgen.SeedCreateRoomParams{
		PropertyID: property1ID, Name: "Kamar Tipe A", PriceMonthly: 2250000, Available: 2, Capacity: 2,
		Facilities: []byte(`["WiFi", "AC", "K. Mandi Dalam", "Kasur", "Akses 24 Jam"]`),
	}); err != nil {
		return fmt.Errorf("seed room 1: %w", err)
	}

	// Three more active & approved properties in the same area, one room each,
	// so property_total matches API.md §3.3's search example (4 total).
	others := []struct {
		name, slug, gender string
	}{
		{"Kost Melati Margonda", "kost-melati-margonda", "putra"},
		{"Kost Anggrek Beji", "kost-anggrek-beji", "putri"},
		{"Kost Cendana Depok", "kost-cendana-depok", "campur"},
	}
	for _, p := range others {
		propertyID, err := q.SeedCreateProperty(ctx, dbgen.SeedCreatePropertyParams{
			OwnerID: ownerID, AreaID: areaID, Name: p.name, Slug: p.slug, Gender: p.gender,
			Description: "Kost nyaman di Kota Depok", Address: "Jl. Contoh No. 1",
			Latitude: -6.38, Longitude: 106.81,
		})
		if err != nil {
			return fmt.Errorf("seed property %q: %w", p.slug, err)
		}
		if _, err := q.SeedCreateRoom(ctx, dbgen.SeedCreateRoomParams{
			PropertyID: propertyID, Name: "Kamar Standar", PriceMonthly: 1500000, Available: 3, Capacity: 1,
			Facilities: []byte(`["WiFi", "Kasur"]`),
		}); err != nil {
			return fmt.Errorf("seed room for %q: %w", p.slug, err)
		}
	}

	if _, err := q.CreateUserNotification(ctx, dbgen.CreateUserNotificationParams{
		UserID: tenantID, Category: "promo", Title: "Promo Ngebut!",
		Body: "Diskon s.d. Rp250.000 untuk bulan pertama.",
	}); err != nil {
		return fmt.Errorf("seed promo notification: %w", err)
	}
	if _, err := q.CreateUserNotification(ctx, dbgen.CreateUserNotificationParams{
		UserID: tenantID, Category: "booking", Title: "Booking disetujui",
		Body: "Pemilik menyetujui booking Kamar Tipe A di Kost Asti Beji.", Link: pgText("/booking/1"),
	}); err != nil {
		return fmt.Errorf("seed booking notification: %w", err)
	}

	return nil
}
