import { AppHeader } from "@/components/features/app-header";
import { CariClient } from "@/components/features/cari-client";
import { DEFAULT_SEARCH_FILTERS } from "@/components/features/filter-sheet";
import type { PopularArea } from "@/components/features/suggestion-list";
import { apiFetch } from "@/lib/api/client";
import { getNavRole } from "@/lib/auth/nav-role";
import { buildSearchBody, type SearchResponseBody } from "@/lib/listings";

const RESULT_LIMIT = 20;

export default async function CariPage({
  searchParams,
}: {
  searchParams: Promise<{ area_id?: string; area_name?: string }>;
}) {
  const params = await searchParams;
  const areaId = params.area_id ? Number(params.area_id) : undefined;
  const areaName = params.area_name;

  const [role, searchResult, popularAreasResult] = await Promise.all([
    getNavRole(),
    areaId
      ? apiFetch<SearchResponseBody>("/listings/search", {
          method: "POST",
          auth: false,
          body: buildSearchBody({
            areaId,
            filters: DEFAULT_SEARCH_FILTERS,
            sort: { field: "availability", direction: "-" },
            limit: RESULT_LIMIT,
            offset: 0,
          }),
        })
      : Promise.resolve(null),
    areaId ? Promise.resolve(null) : apiFetch<{ areas: PopularArea[] }>("/area/city", { auth: false, searchParams: { top: "true" } }),
  ]);

  return (
    <>
      <AppHeader variant="inner" title={areaId ? (areaName ?? "Hasil pencarian") : "Cari kost"} role={role} />
      <CariClient
        initialArea={areaId ? { id: areaId, name: areaName ?? "Hasil pencarian" } : null}
        initialResult={searchResult?.ok ? searchResult.data : null}
        popularAreas={popularAreasResult?.ok ? popularAreasResult.data.areas : []}
      />
    </>
  );
}
