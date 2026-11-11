import { beforeEach, describe, expect, it, vi } from "vitest";

const proxyMock = vi.fn<(...args: unknown[]) => Promise<Response>>(async () => new Response("{}"));
vi.mock("@/lib/api/bff", () => ({ proxyToApi: (...args: unknown[]) => proxyMock(...args) }));

const { GET, POST } = await import("./route");

function context(path: string[]) {
  return { params: Promise.resolve({ path }) };
}

describe("owner BFF route", () => {
  beforeEach(() => proxyMock.mockClear());

  it("forwards method, encoded path, query and body to /owner/*", async () => {
    await POST(
      new Request("http://localhost/api/owner/bookings/7/reject?x=1", { method: "POST", body: JSON.stringify({ reason: "Renovasi" }) }),
      context(["bookings", "7", "reject"]),
    );
    expect(proxyMock).toHaveBeenCalledWith("/owner/bookings/7/reject", {
      method: "POST",
      body: { reason: "Renovasi" },
      searchParams: { x: "1" },
    });
  });

  it("refuses dot segments that would escape /owner", async () => {
    const response = await GET(new Request("http://localhost/api/owner/x"), context(["..", "me"]));
    expect(response.status).toBe(404);
    expect(proxyMock).not.toHaveBeenCalled();
  });

  it("sends a body-less POST as an empty object", async () => {
    await POST(new Request("http://localhost/api/owner/bookings/7/approve", { method: "POST" }), context(["bookings", "7", "approve"]));
    expect(proxyMock).toHaveBeenCalledWith("/owner/bookings/7/approve", { method: "POST", body: {}, searchParams: {} });
  });
});
