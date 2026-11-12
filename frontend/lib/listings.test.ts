import { describe, expect, it } from "vitest";
import { buildSearchBody, DEFAULT_SEARCH_FILTERS } from "./listings";

describe("buildSearchBody", () => {
  it("sends the keyword as a filter without an area", () => {
    const body = buildSearchBody({
      keyword: "kost depok",
      filters: DEFAULT_SEARCH_FILTERS,
      sort: null,
      limit: 20,
      offset: 0,
    });

    expect(body.filters.keyword).toBe("kost depok");
    expect(body.filters.area_id).toBeUndefined();
  });
});
