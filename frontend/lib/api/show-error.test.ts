import { afterEach, describe, expect, it, vi } from "vitest";
import { getToastsSnapshot, resetToastsForTests } from "@/lib/toast";
import type { ApiResult } from "./envelope";
import { showApiError } from "./show-error";

function failure(httpStatus: number, message: string): Extract<ApiResult<unknown>, { ok: false }> {
  return {
    ok: false,
    httpStatus,
    meta: { response_code: httpStatus, code: httpStatus, severity: "ERROR", message },
    messages: null,
    retryAfterSeconds: null,
  };
}

describe("showApiError", () => {
  afterEach(() => resetToastsForTests());

  it("sends an expired session to the login page with the current path as next", () => {
    window.history.pushState({}, "", "/booking/2?tab=1");
    const navigate = vi.fn();
    showApiError(failure(401, "Sesi berakhir."), navigate);
    expect(navigate).toHaveBeenCalledWith("/masuk?next=%2Fbooking%2F2%3Ftab%3D1");
    expect(getToastsSnapshot()[0]).toMatchObject({ variant: "info", title: "Sesi berakhir." });
  });

  it("offers a retry action on 502", () => {
    const retry = vi.fn();
    showApiError(failure(502, "Layanan pembayaran sedang bermasalah."), vi.fn(), retry);
    const [item] = getToastsSnapshot();
    expect(item).toMatchObject({ variant: "error", action: { label: "Coba lagi" } });
    item?.action?.onClick();
    expect(retry).toHaveBeenCalled();
  });

  it("shows rate limiting as a single warning toast", () => {
    showApiError(failure(429, "Terlalu banyak permintaan."), vi.fn());
    showApiError(failure(429, "Terlalu banyak permintaan."), vi.fn());
    expect(getToastsSnapshot()).toHaveLength(1);
    expect(getToastsSnapshot()[0]).toMatchObject({ id: "rate-limit", variant: "warning" });
  });
});
