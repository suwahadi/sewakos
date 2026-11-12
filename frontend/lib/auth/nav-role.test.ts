import { beforeEach, describe, expect, it, vi } from "vitest";

const apiFetchMock = vi.fn();
const sessionMock = { token: null as string | null, role: null as string | null };

vi.mock("@/lib/api/client", () => ({ apiFetch: (...args: unknown[]) => apiFetchMock(...args) }));
vi.mock("./session", () => ({
  getSessionToken: async () => sessionMock.token,
  getSessionRole: async () => sessionMock.role,
}));

const { getNavRole, NavRoleUnavailableError } = await import("./nav-role");

beforeEach(() => {
  apiFetchMock.mockReset();
  sessionMock.token = null;
  sessionMock.role = null;
});

describe("getNavRole", () => {
  it("is a guest without a session token", async () => {
    expect(await getNavRole()).toBe("guest");
    expect(apiFetchMock).not.toHaveBeenCalled();
  });

  it("uses the stored role without calling /me", async () => {
    sessionMock.token = "t";
    sessionMock.role = "owner";
    expect(await getNavRole()).toBe("owner");
    expect(apiFetchMock).not.toHaveBeenCalled();
  });

  it("falls back to /me when no role is stored", async () => {
    sessionMock.token = "t";
    apiFetchMock.mockResolvedValue({ ok: true, httpStatus: 200, data: { user: { role: "owner" } } });
    expect(await getNavRole()).toBe("owner");
  });

  it("treats a rejected token (401) as a guest", async () => {
    sessionMock.token = "t";
    apiFetchMock.mockResolvedValue({ ok: false, httpStatus: 401, meta: { message: "x" } });
    expect(await getNavRole()).toBe("guest");
  });

  it("does not treat a rate limit as a guest", async () => {
    sessionMock.token = "t";
    apiFetchMock.mockResolvedValue({ ok: false, httpStatus: 429, meta: { message: "x" } });
    await expect(getNavRole()).rejects.toBeInstanceOf(NavRoleUnavailableError);
  });
});
