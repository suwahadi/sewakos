import { describe, expect, it } from "vitest";
import {
  formatDateLong,
  formatDateShort,
  formatPercent,
  formatPhoneDisplay,
  formatRating,
  formatRelativeTime,
  formatRupiah,
  formatTimeWIB,
  previousDay,
} from "./format";

describe("formatRupiah", () => {
  it("formats with thousands dots and no spaces", () => {
    expect(formatRupiah(2250000)).toBe("Rp2.250.000");
  });

  it("formats zero", () => {
    expect(formatRupiah(0)).toBe("Rp0");
  });
});

describe("formatDateShort", () => {
  it("renders day/short-month/year in WIB", () => {
    expect(formatDateShort("2026-12-01T00:00:00.000000Z")).toBe("1 Des 2026");
  });
});

describe("formatDateLong", () => {
  it("renders weekday + long month in WIB", () => {
    // 2026-12-01 falls on a Tuesday.
    expect(formatDateLong("2026-12-01T00:00:00.000000Z")).toBe("Selasa, 1 Desember 2026");
  });
});

describe("formatTimeWIB", () => {
  it("renders hour:minute suffixed with WIB", () => {
    expect(formatTimeWIB("2026-11-10T10:05:00+00:00")).toBe("17.05 WIB");
  });
});

describe("formatRelativeTime", () => {
  const now = new Date("2026-11-10T12:00:00Z");

  it("renders minutes in the past", () => {
    expect(formatRelativeTime("2026-11-10T11:55:00Z", now)).toBe("5 menit yang lalu");
  });

  it("falls back to a date beyond 7 days", () => {
    expect(formatRelativeTime("2026-10-01T00:00:00Z", now)).toBe(formatDateShort("2026-10-01T00:00:00Z"));
  });
});

describe("formatRating", () => {
  it("renders one decimal with a comma", () => {
    expect(formatRating(4.5)).toBe("4,5");
  });
});

describe("formatPercent", () => {
  it("rounds to the nearest whole percent", () => {
    expect(formatPercent(0.4)).toBe("40%");
  });
});

describe("formatPhoneDisplay", () => {
  it("groups digits 4-4-rest", () => {
    expect(formatPhoneDisplay("081200000002")).toBe("0812 0000 0002");
  });
});

describe("previousDay", () => {
  it("subtracts one calendar day", () => {
    expect(previousDay("2027-03-01")).toBe("2027-02-28");
  });

  it("crosses a month/year boundary", () => {
    expect(previousDay("2027-01-01")).toBe("2026-12-31");
  });
});
