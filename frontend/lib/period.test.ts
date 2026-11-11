import { describe, expect, it } from "vitest";
import { rentalEndDate } from "./period";

describe("rentalEndDate", () => {
  it("adds days and weeks", () => {
    expect(rentalEndDate("2026-12-30", "daily", 3)).toBe("2027-01-02");
    expect(rentalEndDate("2026-12-01", "weekly", 2)).toBe("2026-12-15");
  });

  it("adds calendar months clamped to the end of the target month", () => {
    expect(rentalEndDate("2026-12-01", "monthly", 3)).toBe("2027-03-01");
    expect(rentalEndDate("2027-01-31", "monthly", 1)).toBe("2027-02-28");
    expect(rentalEndDate("2028-01-31", "monthly", 1)).toBe("2028-02-29");
    expect(rentalEndDate("2027-01-31", "monthly", 2)).toBe("2027-03-31");
  });
});
