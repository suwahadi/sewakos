import { act, renderHook } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import type { ApiResult } from "@/lib/api/envelope";
import { MAX_OTP_RESENDS, useOtpRequest, type OtpRequestResponse } from "./use-otp-request";

function success(canResendAt: number): ApiResult<OtpRequestResponse> {
  return {
    ok: true,
    httpStatus: 200,
    data: {
      status: true,
      meta: { response_code: 200, code: 200, severity: "OK", message: "Kode verifikasi telah dikirim." },
      verification_code: { can_resend_at: canResendAt },
    },
  };
}

function failure(httpStatus: number, message: string): ApiResult<OtpRequestResponse> {
  return {
    ok: false,
    httpStatus,
    meta: { response_code: httpStatus, code: httpStatus, severity: "WARNING", message },
    messages: null,
    retryAfterSeconds: null,
  };
}

const past = () => Math.floor(Date.now() / 1000) - 1;

describe("useOtpRequest", () => {
  afterEach(() => vi.useRealTimers());

  it("ignores a second click while the first request is still in flight", async () => {
    let resolve: (value: ApiResult<OtpRequestResponse>) => void = () => {};
    const send = vi.fn(() => new Promise<ApiResult<OtpRequestResponse>>((r) => (resolve = r)));
    const { result } = renderHook(() => useOtpRequest(send));

    let first: Promise<boolean> = Promise.resolve(false);
    let second = true;
    await act(async () => {
      first = result.current.request();
      second = await result.current.request();
    });
    await act(async () => {
      resolve(success(past()));
      await first;
    });

    expect(second).toBe(false);
    expect(send).toHaveBeenCalledTimes(1);
  });

  it("refuses to resend before can_resend_at", async () => {
    const send = vi.fn().mockResolvedValue(success(Math.floor(Date.now() / 1000) + 60));
    const { result } = renderHook(() => useOtpRequest(send));

    await act(async () => {
      await result.current.request();
    });
    let resent = true;
    await act(async () => {
      resent = await result.current.request();
    });

    expect(resent).toBe(false);
    expect(send).toHaveBeenCalledTimes(1);
  });

  it(`allows at most ${MAX_OTP_RESENDS} resends per visit`, async () => {
    const send = vi.fn().mockResolvedValue(success(past()));
    const { result } = renderHook(() => useOtpRequest(send));

    for (let i = 0; i < MAX_OTP_RESENDS + 3; i++) {
      await act(async () => {
        await result.current.request();
      });
    }

    expect(send).toHaveBeenCalledTimes(1 + MAX_OTP_RESENDS);
    expect(result.current.resendsLeft).toBe(0);
  });

  it("stops sending for good after a 429", async () => {
    const send = vi.fn().mockResolvedValue(failure(429, "Batas pengiriman kode untuk nomor ini sudah tercapai."));
    const { result } = renderHook(() => useOtpRequest(send));

    await act(async () => {
      await result.current.request();
    });
    await act(async () => {
      await result.current.request();
    });

    expect(send).toHaveBeenCalledTimes(1);
    expect(result.current.blockedMessage).toBe("Batas pengiriman kode untuk nomor ini sudah tercapai.");
  });

  it("waits a minute after the provider fails before trying again", async () => {
    const send = vi.fn().mockResolvedValue(failure(503, "Kode verifikasi gagal dikirim."));
    const { result } = renderHook(() => useOtpRequest(send));

    await act(async () => {
      await result.current.request();
    });
    await act(async () => {
      await result.current.request();
    });

    expect(send).toHaveBeenCalledTimes(1);
    expect(result.current.canResendAt).toBeGreaterThan(Math.floor(Date.now() / 1000));
  });
});
