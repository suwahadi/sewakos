import type { ListingCardRoom } from "@/components/features/listing-card";
import type { SearchFilters, SortOption } from "@/components/features/filter-sheet";

export interface SearchResponseBody {
  property_total: number;
  total: number;
  limit: number;
  "has-more": boolean;
  "next-offset": number | null;
  rooms: ListingCardRoom[];
}

export interface SearchRequestBody {
  filters: {
    gender?: number[];
    price_range?: [number, number];
    rent_type?: number;
    flash_sale?: boolean;
    is_available?: boolean;
    area_id?: number;
  };
  sorting?: { field: string; direction: string };
  limit?: number;
  offset?: number;
}

export function buildSearchBody(options: {
  areaId?: number;
  filters: SearchFilters;
  sort: SortOption | null;
  limit: number;
  offset: number;
}): SearchRequestBody {
  const { areaId, filters, sort, limit, offset } = options;
  return {
    filters: {
      area_id: areaId,
      rent_type: filters.rentType,
      gender: filters.gender.length > 0 ? filters.gender : undefined,
      price_range: filters.priceMin !== undefined || filters.priceMax !== undefined ? [filters.priceMin ?? 0, filters.priceMax ?? 999999999] : undefined,
      flash_sale: filters.flashSale,
      is_available: filters.isAvailable,
    },
    sorting: sort ? { field: sort.field, direction: sort.direction } : undefined,
    limit,
    offset,
  };
}
