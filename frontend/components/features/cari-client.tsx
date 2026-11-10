"use client";

import { Search } from "lucide-react";
import { useRouter } from "next/navigation";
import { useRef, useState } from "react";
import { ListingCard, ListingCardSkeleton } from "@/components/features/listing-card";
import {
  DEFAULT_SEARCH_FILTERS,
  FilterSheet,
  SortSheet,
  type SearchFilters,
  type SortOption,
} from "@/components/features/filter-sheet";
import {
  SuggestionList,
  type PopularArea,
  type SuggestionArea,
  type SuggestionLanding,
  type SuggestionRoomType,
} from "@/components/features/suggestion-list";
import { FilterBar, type ActiveFilterChip } from "@/components/features/filter-bar";
import { EmptyState } from "@/components/ui/empty-state";
import { LoadMore } from "@/components/ui/load-more";
import { callBff } from "@/lib/api/browser";
import { buildSearchBody, type SearchResponseBody } from "@/lib/listings";
import { toast } from "@/lib/toast";

const PAGE_LIMIT = 20;

interface AreaSelection {
  id: number;
  name: string;
}

export function CariClient({
  initialArea,
  initialResult,
  popularAreas,
}: {
  initialArea: AreaSelection | null;
  initialResult: SearchResponseBody | null;
  popularAreas: PopularArea[];
}) {
  const router = useRouter();
  const [area, setArea] = useState(initialArea);
  const [filters, setFilters] = useState<SearchFilters>(DEFAULT_SEARCH_FILTERS);
  const [sort, setSort] = useState<SortOption>({ field: "availability", direction: "-" });
  const [result, setResult] = useState(initialResult);
  const [loadingResults, setLoadingResults] = useState(false);
  const [loadingMore, setLoadingMore] = useState(false);
  const [filterSheetOpen, setFilterSheetOpen] = useState(false);
  const [sortSheetOpen, setSortSheetOpen] = useState(false);

  const [areas, setAreas] = useState<SuggestionArea[]>([]);
  const [landings, setLandings] = useState<SuggestionLanding[]>([]);
  const [roomTypes, setRoomTypes] = useState<SuggestionRoomType[]>([]);
  const suggestionAbortRef = useRef<AbortController | null>(null);

  async function handleQueryChange(query: string) {
    suggestionAbortRef.current?.abort();
    if (!query) {
      setAreas([]);
      setLandings([]);
      setRoomTypes([]);
      return;
    }
    const controller = new AbortController();
    suggestionAbortRef.current = controller;
    try {
      const response = await fetch(`/api/suggestion?keyword=${encodeURIComponent(query)}`, { signal: controller.signal });
      const body = await response.json();
      if (controller.signal.aborted) return;
      if (body?.status) {
        setAreas(body.suggestion.areas ?? []);
        setLandings(body.suggestion.landings ?? []);
        setRoomTypes(body.suggestion.room_types ?? []);
      }
    } catch {
      /* AbortController cancellation reaching here is expected; ignore. */
    }
  }

  async function runSearch(nextArea: AreaSelection, nextFilters: SearchFilters, nextSort: SortOption) {
    setLoadingResults(true);
    const response = await callBff<SearchResponseBody>("/api/listings/search", {
      method: "POST",
      body: buildSearchBody({ areaId: nextArea.id, filters: nextFilters, sort: nextSort, limit: PAGE_LIMIT, offset: 0 }),
    });
    setLoadingResults(false);
    if (!response.ok) {
      toast.error(response.meta.message);
      return;
    }
    setResult(response.data);
  }

  function selectArea(nextArea: AreaSelection) {
    setArea(nextArea);
    router.replace(`/cari?area_id=${nextArea.id}&area_name=${encodeURIComponent(nextArea.name)}`);
    runSearch(nextArea, filters, sort);
  }

  async function applyFilters(nextFilters: SearchFilters) {
    setFilters(nextFilters);
    if (area) await runSearch(area, nextFilters, sort);
  }

  async function applySort(nextSort: SortOption) {
    setSort(nextSort);
    if (area) await runSearch(area, filters, nextSort);
  }

  async function loadMore() {
    const nextOffset = result?.["next-offset"];
    if (!area || nextOffset === null || nextOffset === undefined) return;
    setLoadingMore(true);
    const response = await callBff<SearchResponseBody>("/api/listings/search", {
      method: "POST",
      body: buildSearchBody({ areaId: area.id, filters, sort, limit: PAGE_LIMIT, offset: nextOffset }),
    });
    setLoadingMore(false);
    if (!response.ok) {
      toast.error(response.meta.message);
      return;
    }
    setResult((previous) => (previous ? { ...response.data, rooms: [...previous.rooms, ...response.data.rooms] } : response.data));
  }

  function removeGenderFilter(gender: number) {
    applyFilters({ ...filters, gender: filters.gender.filter((value) => value !== gender) });
  }

  if (!area) {
    return (
      <div className="flex flex-col gap-4 px-4 py-4">
        <SuggestionList
          onQueryChange={handleQueryChange}
          areas={areas}
          landings={landings}
          roomTypes={roomTypes}
          popularAreas={popularAreas}
          onSelectArea={(selected) => selectArea({ id: selected.geocode_id, name: selected.title })}
          onSelectLanding={(landing) => router.push(`/kost/${landing.id}`)}
          onSelectRoomType={(roomType) => router.push(`/kost/${roomType.property_id}?room=${roomType.id}`)}
          onSelectPopularArea={(popular) => selectArea({ id: popular.id, name: popular.name })}
        />
      </div>
    );
  }

  const genderLabels: Record<number, string> = { 0: "Campur", 1: "Putra", 2: "Putri" };
  const activeChips: ActiveFilterChip[] = [
    ...filters.gender.map((gender) => ({
      key: `gender-${gender}`,
      label: genderLabels[gender] ?? String(gender),
      onRemove: () => removeGenderFilter(gender),
    })),
    ...(filters.flashSale ? [{ key: "flash-sale", label: "Promo", onRemove: () => applyFilters({ ...filters, flashSale: false }) }] : []),
    ...(filters.isAvailable
      ? [{ key: "is-available", label: "Tersedia", onRemove: () => applyFilters({ ...filters, isAvailable: false }) }]
      : []),
  ];
  const activeFilterCount =
    filters.gender.length +
    (filters.flashSale ? 1 : 0) +
    (filters.isAvailable ? 1 : 0) +
    (filters.priceMin !== undefined || filters.priceMax !== undefined ? 1 : 0);

  return (
    <div>
      <FilterBar
        activeFilterCount={activeFilterCount}
        onOpenFilter={() => setFilterSheetOpen(true)}
        onOpenSort={() => setSortSheetOpen(true)}
        activeChips={activeChips}
      />
      <div className="px-4 pb-3">
        {result && (
          <p className="text-caption text-neutral-600">
            {result.total} kamar di {result.property_total} kost
          </p>
        )}
      </div>
      <div className="flex flex-col gap-4 px-4 sm:grid sm:grid-cols-2 sm:gap-4 lg:grid-cols-3 xl:grid-cols-4">
        {loadingResults
          ? Array.from({ length: 6 }, (_, i) => <ListingCardSkeleton key={i} />)
          : result?.rooms.map((room) => <ListingCard key={room.id} room={room} rentType={filters.rentType} />)}
      </div>
      {!loadingResults && result?.rooms.length === 0 && (
        <EmptyState icon={Search} title="Tidak ada kost yang cocok" description="Coba ubah filter pencarianmu." />
      )}
      {!loadingResults && result && result["has-more"] && (
        <div className="p-4">
          <LoadMore loading={loadingMore} onClick={loadMore} />
        </div>
      )}
      <FilterSheet open={filterSheetOpen} onClose={() => setFilterSheetOpen(false)} filters={filters} onApply={applyFilters} />
      <SortSheet open={sortSheetOpen} onClose={() => setSortSheetOpen(false)} value={sort} onChange={applySort} />
    </div>
  );
}
