import { afterEach, describe, expect, it, vi } from "vitest";

const incoming = new Headers();
vi.mock("next/headers", () => ({ headers: async () => incoming }));
vi.mock("@/lib/auth/session", () => ({ getSessionToken: async () => null }));

describe("apiFetch", () => {
  afterEach(() => {
    vi.unstubAllEnvs();
    vi.unstubAllGlobals();
  });

  it("forwards the incoming X-Forwarded-For so the API rate-limits per visitor", async () => {
    vi.stubEnv("API_BASE_URL", "http://api.test/api");
    vi.stubEnv("API_CLIENT_ID", "KOSAPI WEB:WEB");
    const fetchMock = vi.fn<(input: string, init: RequestInit) => Promise<Response>>(async () =>
      Response.json({ status: true, meta: { response_code: 200, code: 200, severity: "OK", message: "" } }),
    );
    vi.stubGlobal("fetch", fetchMock);
    incoming.set("x-forwarded-for", "198.51.100.7");

    vi.resetModules();
    const { apiFetch } = await import("./client");
    await apiFetch("/area/city", { auth: false });

    const sent = fetchMock.mock.calls[0]?.[1].headers as Record<string, string>;
    expect(sent["X-Forwarded-For"]).toBe("198.51.100.7");
    expect(sent["x-api-client"]).toBe("KOSAPI WEB:WEB");
  });
});
