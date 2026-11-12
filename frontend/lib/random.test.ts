import { describe, expect, it } from "vitest";
import { pickRandom } from "./random";

describe("pickRandom", () => {
  it("returns the requested number of distinct items from the source", () => {
    const source = [1, 2, 3, 4, 5, 6, 7, 8];
    const picked = pickRandom(source, 6);

    expect(picked).toHaveLength(6);
    expect(new Set(picked).size).toBe(6);
    for (const item of picked) expect(source).toContain(item);
  });

  it("returns every item when there are fewer than requested", () => {
    expect(pickRandom([1, 2], 6).sort()).toEqual([1, 2]);
  });

  it("does not mutate the source list", () => {
    const source = [1, 2, 3];
    pickRandom(source, 2);
    expect(source).toEqual([1, 2, 3]);
  });
});
