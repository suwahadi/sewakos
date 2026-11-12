import { AppHeader } from "@/components/features/app-header";
import { AreaBannerGrid, type AreaBannerOption } from "@/components/features/area-banner-grid";
import { AreaChipRow } from "@/components/features/area-chip-row";
import { HomeSearchEntry } from "@/components/features/home-search-entry";
import { ListingCard } from "@/components/features/listing-card";
import { apiFetch } from "@/lib/api/client";
import { getNavRole } from "@/lib/auth/nav-role";
import { buildSearchBody, DEFAULT_SEARCH_FILTERS, type SearchResponseBody } from "@/lib/listings";
import { getUnreadCount } from "@/lib/notifications";

async function getTopAreas(type: "city" | "campus"): Promise<AreaBannerOption[]> {
  const result = await apiFetch<{ areas: AreaBannerOption[] }>(`/area/${type}`, { auth: false, searchParams: { top: "true" } });
  return result.ok ? result.data.areas : [];
}

async function getPromoRooms(): Promise<SearchResponseBody["rooms"]> {
  const result = await apiFetch<SearchResponseBody>("/listings/search", {
    method: "POST",
    auth: false,
    body: buildSearchBody({ filters: { ...DEFAULT_SEARCH_FILTERS, flashSale: true }, sort: null, limit: 6, offset: 0 }),
  });
  return result.ok ? result.data.rooms : [];
}

async function getLatestRooms(): Promise<SearchResponseBody["rooms"]> {
  const result = await apiFetch<Pick<SearchResponseBody, "rooms">>("/listings/latest", {
    auth: false,
    searchParams: { limit: "12" },
  });
  return result.ok ? result.data.rooms : [];
}

export default async function HomePage() {
  const [role, unreadCount, cities, campuses, promoRooms, latestRooms] = await Promise.all([
    getNavRole(),
    getUnreadCount(),
    getTopAreas("city"),
    getTopAreas("campus"),
    getPromoRooms(),
    getLatestRooms(),
  ]);

  return (
    <>
      <AppHeader variant="home" role={role} unreadCount={unreadCount} />
      <main className="mx-auto flex max-w-[1200px] flex-col gap-6 px-4 py-6">
        <div className="flex flex-col gap-3">
          <h1 className="text-display font-bold text-neutral-900">Cari kost yang pas</h1>
          <HomeSearchEntry />
        </div>

        {cities.length > 0 && (
          <section className="flex flex-col gap-2">
            <h2 className="text-title-sm font-semibold text-neutral-900">Kota populer</h2>
            <AreaBannerGrid areas={cities} />
          </section>
        )}

        {campuses.length > 0 && (
          <section className="-mx-4 flex flex-col gap-2">
            <h2 className="px-4 text-title-sm font-semibold text-neutral-900">Kampus terdekat</h2>
            <AreaChipRow areas={campuses} />
          </section>
        )}

        {promoRooms.length > 0 && (
          <section className="-mx-4 flex flex-col gap-2">
            <h2 className="px-4 text-title-sm font-semibold text-neutral-900">Promo minggu ini</h2>
            <div className="flex gap-3 overflow-x-auto px-4 [scrollbar-width:none] [-ms-overflow-style:none] [&::-webkit-scrollbar]:hidden">
              {promoRooms.map((room) => (
                <ListingCard key={room.id} room={room} vertical className="w-40 shrink-0 sm:w-48" />
              ))}
            </div>
          </section>
        )}

        {latestRooms.length > 0 && (
          <section className="flex flex-col gap-2">
            <h2 className="text-title-sm font-semibold text-neutral-900">Kost terbaru</h2>
            <div className="grid grid-cols-2 gap-x-3 gap-y-6 sm:grid-cols-3 lg:grid-cols-4">
              {latestRooms.map((room) => (
                <ListingCard key={room.id} room={room} vertical />
              ))}
            </div>
          </section>
        )}
      </main>
    </>
  );
}
