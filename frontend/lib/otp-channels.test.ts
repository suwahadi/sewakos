import { renderHook, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { useOtpChannels } from "./otp-channels";

function respond(body: unknown, status = 200) {
  vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify(body), { status })));
}

describe("useOtpChannels", () => {
  afterEach(() => vi.unstubAllGlobals());

  it("lists SMS when the server offers it", async () => {
    respond({ status: true, meta: { response_code: 200, code: 200, severity: "OK", message: "" }, channels: ["whatsapp", "sms"] });
    const { result } = renderHook(() => useOtpChannels());

    await waitFor(() => expect(result.current).toEqual(["whatsapp", "sms"]));
  });

  it("falls back to WhatsApp only when the list cannot be loaded", async () => {
    respond({ status: false, meta: { response_code: 503, code: 503, severity: "ERROR", message: "x" }, messages: null }, 503);
    const { result } = renderHook(() => useOtpChannels());

    await waitFor(() => expect(vi.mocked(fetch)).toHaveBeenCalled());
    expect(result.current).toEqual(["whatsapp"]);
  });

  it("ignores unknown channel names", async () => {
    respond({ status: true, meta: { response_code: 200, code: 200, severity: "OK", message: "" }, channels: ["pigeon", "sms"] });
    const { result } = renderHook(() => useOtpChannels());

    await waitFor(() => expect(result.current).toEqual(["sms"]));
  });
});
