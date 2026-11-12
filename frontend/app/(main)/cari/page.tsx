import { AppHeader } from "@/components/features/app-header";
import { CariClient } from "@/components/features/cari-client";
import type { PopularArea } from "@/components/features/suggestion-list";
import { apiFetch } from "@/lib/api/client";
import { getNavRole } from "@/lib/auth/nav-role";
import { buildSearchBody, DEFAULT_SEARCH_FILTERS, type SearchResponseBody } from "@/lib/listings";

const RESULT_LIMIT = 20;

export default async function CariPage({
  searchParams,
}: {
  searchParams: Promise<{ area_id?: string; area_name?: string; q?: string }>;
}) {
  const params = await searchParams;
  const areaId = params.area_id ? Number(params.area_id) : undefined;
  const areaName = params.area_name;
  const keyword = params.q?.trim() ?? "";
  const activeKeyword = keyword.length >= 3 ? keyword : null;

  const [role, searchResult, popularAreasResult] = await Promise.all([
    getNavRole(),
    areaId || activeKeyword
      ? apiFetch<SearchResponseBody>("/listings/search", {
          method: "POST",
          auth: false,
          body: buildSearchBody({
            areaId,
            keyword: activeKeyword ?? undefined,
            filters: DEFAULT_SEARCH_FILTERS,
            sort: { field: "availability", direction: "-" },
            limit: RESULT_LIMIT,
            offset: 0,
          }),
        })
      : Promise.resolve(null),
    areaId || activeKeyword ? Promise.resolve(null) : apiFetch<{ areas: PopularArea[] }>("/area/city", { auth: false, searchParams: { top: "true" } }),
  ]);

  return (
    <>
      <AppHeader
        variant="inner"
        title={areaId ? (areaName ?? "Hasil pencarian") : activeKeyword ? "Hasil pencarian" : "Cari kost"}
        role={role}
      />
      <CariClient
        initialArea={areaId ? { id: areaId, name: areaName ?? "Hasil pencarian" } : null}
        initialKeyword={activeKeyword}
        initialResult={searchResult?.ok ? searchResult.data : null}
        popularAreas={popularAreasResult?.ok ? popularAreasResult.data.areas : []}
      />
    </>
  );
}
