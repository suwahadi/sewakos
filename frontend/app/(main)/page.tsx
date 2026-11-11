import { AppHeader } from "@/components/features/app-header";
import { AreaChipRow, type AreaChipOption } from "@/components/features/area-chip-row";
import { HomeSearchEntry } from "@/components/features/home-search-entry";
import { ListingCard } from "@/components/features/listing-card";
import { apiFetch } from "@/lib/api/client";
import { getNavRole } from "@/lib/auth/nav-role";
import { getSessionToken } from "@/lib/auth/session";
import { buildSearchBody, DEFAULT_SEARCH_FILTERS, type SearchResponseBody } from "@/lib/listings";

async function getUnreadCount(): Promise<number> {
  const token = await getSessionToken();
  if (!token) return 0;
  const result = await apiFetch<{ unread: number }>("/notifications/counter");
  return result.ok ? result.data.unread : 0;
}

async function getTopAreas(type: "city" | "campus"): Promise<AreaChipOption[]> {
  const result = await apiFetch<{ areas: AreaChipOption[] }>(`/area/${type}`, { auth: false, searchParams: { top: "true" } });
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

export default async function HomePage() {
  const [role, unreadCount, cities, campuses, promoRooms] = await Promise.all([
    getNavRole(),
    getUnreadCount(),
    getTopAreas("city"),
    getTopAreas("campus"),
    getPromoRooms(),
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
            <p className="px-4 text-title-sm font-semibold text-neutral-900">Kota populer</p>
            <AreaChipRow areas={cities} />
          </section>
        )}

        {campuses.length > 0 && (
          <section className="flex flex-col gap-2">
            <p className="px-4 text-title-sm font-semibold text-neutral-900">Kampus terdekat</p>
            <AreaChipRow areas={campuses} />
          </section>
        )}

        {promoRooms.length > 0 && (
          <section className="flex flex-col gap-2">
            <p className="px-4 text-title-sm font-semibold text-neutral-900">Promo minggu ini</p>
            <div className="flex gap-3 overflow-x-auto px-4 [scrollbar-width:none] [-ms-overflow-style:none] [&::-webkit-scrollbar]:hidden">
              {promoRooms.map((room) => (
                <ListingCard key={room.id} room={room} className="w-40 shrink-0 sm:w-48" />
              ))}
            </div>
          </section>
        )}
      </main>
    </>
  );
}
