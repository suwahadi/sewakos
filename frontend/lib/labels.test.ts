import { describe, expect, it } from "vitest";
import {
  bookingStatusLabel,
  durationLabel,
  genderLabel,
  invoiceStatusLabel,
  invoiceTypeLabel,
  moderationStatusLabel,
  notificationCategoryLabel,
  paymentStatusLabel,
  payoutStatusLabel,
  refundStatusLabel,
  rentTypeLabel,
} from "./labels";

describe("bookingStatusLabel", () => {
  it.each([
    ["pending", "Menunggu pembayaran", "warning"],
    ["paid", "Menunggu konfirmasi pemilik", "info"],
    ["approved", "Aktif", "success"],
    ["completed", "Selesai", "neutral"],
    ["terminated", "Dihentikan", "danger"],
    ["rejected", "Ditolak", "danger"],
    ["cancelled", "Dibatalkan", "neutral"],
  ])("maps %s", (status, label, color) => {
    expect(bookingStatusLabel(status)).toEqual({ label, color });
  });

  it("falls back to the raw value for an unknown status", () => {
    expect(bookingStatusLabel("unknown")).toEqual({ label: "unknown", color: "neutral" });
  });
});

describe("invoiceStatusLabel", () => {
  it.each([
    ["unpaid", "Belum dibayar", "warning"],
    ["paid", "Lunas", "success"],
    ["void", "Dibatalkan", "neutral"],
  ])("maps %s", (status, label, color) => {
    expect(invoiceStatusLabel(status)).toEqual({ label, color });
  });
});

describe("paymentStatusLabel", () => {
  it.each([
    ["pending", "Menunggu pembayaran", "warning"],
    ["settlement", "Berhasil", "success"],
    ["expire", "Kedaluwarsa", "neutral"],
    ["cancel", "Dibatalkan", "danger"],
    ["deny", "Ditolak", "danger"],
    ["failure", "Gagal", "danger"],
  ])("maps %s", (status, label, color) => {
    expect(paymentStatusLabel(status)).toEqual({ label, color });
  });
});

describe("refundStatusLabel", () => {
  it.each([
    ["awaiting_account", "Perlu rekening", "warning"],
    ["ready", "Diproses", "info"],
    ["completed", "Selesai", "success"],
  ])("maps %s", (status, label, color) => {
    expect(refundStatusLabel(status)).toEqual({ label, color });
  });
});

describe("payoutStatusLabel", () => {
  it.each([
    ["pending", "Menunggu transfer", "warning"],
    ["completed", "Sudah ditransfer", "success"],
  ])("maps %s", (status, label, color) => {
    expect(payoutStatusLabel(status)).toEqual({ label, color });
  });
});

describe("moderationStatusLabel", () => {
  it.each([
    ["pending", "Sedang ditinjau", "info"],
    ["approved", "Tayang", "success"],
    ["rejected", "Ditolak", "danger"],
    ["suspended", "Dinonaktifkan", "danger"],
  ])("maps %s", (status, label, color) => {
    expect(moderationStatusLabel(status)).toEqual({ label, color });
  });
});

describe("genderLabel", () => {
  it.each([
    ["putra", "Putra"],
    ["putri", "Putri"],
    ["campur", "Campur"],
  ])("maps %s", (gender, label) => {
    expect(genderLabel(gender)).toBe(label);
  });
});

describe("notificationCategoryLabel", () => {
  it.each([
    ["booking", "Booking"],
    ["payment", "Pembayaran"],
    ["promo", "Promo"],
    ["chat", "Chat"],
  ])("maps %s", (category, label) => {
    expect(notificationCategoryLabel(category)).toBe(label);
  });
});

describe("rent type and invoice type labels", () => {
  it("names rent types and durations", () => {
    expect(rentTypeLabel("weekly")).toBe("Mingguan");
    expect(durationLabel("monthly", 3)).toBe("3 bulan");
    expect(durationLabel("daily", 5)).toBe("5 hari");
  });

  it("names invoice types", () => {
    expect(invoiceTypeLabel("initial")).toBe("Pembayaran pertama");
    expect(invoiceTypeLabel("renewal")).toBe("Tagihan bulanan");
  });
});
