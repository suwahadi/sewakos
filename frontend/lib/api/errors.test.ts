import { describe, expect, it } from "vitest";
import type { ApiMeta, ApiResult } from "./envelope";
import { mapApiError } from "./errors";

function errorResult(
  httpStatus: number,
  message: string,
  overrides: Partial<{ messages: Record<string, string[]> | null; retryAfterSeconds: number | null }> = {},
): Extract<ApiResult<unknown>, { ok: false }> {
  const meta: ApiMeta = { response_code: httpStatus, code: httpStatus, severity: "WARNING", message };
  return {
    ok: false,
    httpStatus,
    meta,
    messages: overrides.messages ?? null,
    retryAfterSeconds: overrides.retryAfterSeconds ?? null,
  };
}

describe("mapApiError", () => {
  it("422 maps to validation with field messages, no toast", () => {
    const fieldMessages = { phone_number: ["The phone number field is required."] };
    const result = mapApiError(errorResult(422, "Validasi gagal.", { messages: fieldMessages }));
    expect(result).toEqual({ kind: "validation", message: "Validasi gagal.", fieldMessages });
  });

  it("401 maps to session_expired (token invalid case)", () => {
    const result = mapApiError(errorResult(401, "Sesi tidak valid. Silakan login kembali."));
    expect(result).toEqual({ kind: "session_expired", message: "Sesi tidak valid. Silakan login kembali." });
  });

  it("403 with the suspended message forces logout", () => {
    const result = mapApiError(errorResult(403, "Akun dinonaktifkan."));
    expect(result).toEqual({ kind: "forced_logout", message: "Akun dinonaktifkan." });
  });

  it("403 otherwise is a generic toast", () => {
    const result = mapApiError(errorResult(403, "Akses ditolak."));
    expect(result).toEqual({ kind: "generic", message: "Akses ditolak." });
  });

  it("404 maps to not_found", () => {
    const result = mapApiError(errorResult(404, "Data tidak ditemukan."));
    expect(result).toEqual({ kind: "not_found", message: "Data tidak ditemukan." });
  });

  it("429 maps to rate_limited and carries Retry-After", () => {
    const result = mapApiError(
      errorResult(429, "Terlalu banyak permintaan. Silakan coba lagi nanti.", { retryAfterSeconds: 42 }),
    );
    expect(result).toEqual({
      kind: "rate_limited",
      message: "Terlalu banyak permintaan. Silakan coba lagi nanti.",
      retryAfterSeconds: 42,
    });
  });

  it.each([500, 502])("%d maps to retryable_error", (status) => {
    const result = mapApiError(errorResult(status, "Terjadi kesalahan pada server."));
    expect(result.kind).toBe("retryable_error");
  });

  it("400 maps to generic", () => {
    const result = mapApiError(errorResult(400, "Kamar tidak tersedia."));
    expect(result).toEqual({ kind: "generic", message: "Kamar tidak tersedia." });
  });

  it("network failure (httpStatus 0) maps to generic with the offline message", () => {
    const result = mapApiError(errorResult(0, "Tidak ada koneksi internet. Periksa jaringan lalu coba lagi."));
    expect(result).toEqual({
      kind: "generic",
      message: "Tidak ada koneksi internet. Periksa jaringan lalu coba lagi.",
    });
  });
});
