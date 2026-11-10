import { describe, expect, it } from "vitest";
import { networkFailureResult, parseEnvelope } from "./envelope";

function jsonResponse(body: unknown, init?: ResponseInit) {
  return new Response(JSON.stringify(body), {
    status: 200,
    headers: { "Content-Type": "application/json" },
    ...init,
  });
}

describe("parseEnvelope", () => {
  it("parses a success envelope", async () => {
    const response = jsonResponse(
      { status: true, meta: { response_code: 200, code: 200, severity: "OK", message: "OK" }, user: { id: 1 } },
      { status: 200 },
    );
    const result = await parseEnvelope<{ user: { id: number } }>(response);
    expect(result.ok).toBe(true);
    if (result.ok) {
      expect(result.data.user).toEqual({ id: 1 });
      expect(result.httpStatus).toBe(200);
    }
  });

  it("parses an error envelope with field messages", async () => {
    const response = jsonResponse(
      {
        status: false,
        meta: { response_code: 422, code: 422, severity: "WARNING", message: "Validasi gagal." },
        messages: { name: ["The name field is required."] },
      },
      { status: 422 },
    );
    const result = await parseEnvelope(response);
    expect(result.ok).toBe(false);
    if (!result.ok) {
      expect(result.meta.message).toBe("Validasi gagal.");
      expect(result.messages).toEqual({ name: ["The name field is required."] });
      expect(result.retryAfterSeconds).toBeNull();
    }
  });

  it("extracts Retry-After for 429 responses", async () => {
    const response = jsonResponse(
      {
        status: false,
        meta: { response_code: 429, code: 429, severity: "WARNING", message: "Terlalu banyak permintaan." },
        messages: null,
      },
      { status: 429, headers: { "Content-Type": "application/json", "Retry-After": "30" } },
    );
    const result = await parseEnvelope(response);
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.retryAfterSeconds).toBe(30);
  });

  it("treats unparseable JSON as a generic server error", async () => {
    const response = new Response("not json", { status: 500 });
    const result = await parseEnvelope(response);
    expect(result.ok).toBe(false);
    if (!result.ok) {
      expect(result.httpStatus).toBe(500);
      expect(result.meta.message).toBe("Terjadi kesalahan pada server.");
    }
  });
});

describe("networkFailureResult", () => {
  it("reports the offline message with httpStatus 0", () => {
    const result = networkFailureResult();
    expect(result.ok).toBe(false);
    if (!result.ok) {
      expect(result.httpStatus).toBe(0);
      expect(result.meta.message).toBe("Tidak ada koneksi internet. Periksa jaringan lalu coba lagi.");
    }
  });
});
