import { act, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { getToastsSnapshot, resetToastsForTests } from "@/lib/toast";
import { formatRemaining } from "@/lib/use-remaining";
import { Countdown } from "./payment-panel";

describe("formatRemaining", () => {
  it("uses hours and minutes from one hour up, mm:ss below", () => {
    expect(formatRemaining((23 * 60 + 10) * 60_000 + 59_000)).toBe("23 jam 10 menit");
    expect(formatRemaining(60 * 60_000)).toBe("1 jam 0 menit");
    expect(formatRemaining(14 * 60_000 + 32_000)).toBe("14:32");
    expect(formatRemaining(-5)).toBe("00:00");
  });
});

describe("Countdown", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-11-11T10:00:00Z"));
  });
  afterEach(() => {
    vi.useRealTimers();
    resetToastsForTests();
  });

  it("prefers the active payment window and warns once when five minutes are left", () => {
    render(
      <Countdown
        expiresAt="2026-11-12T10:00:00Z"
        activePaymentExpiresAt="2026-11-11T10:05:01Z"
        onExpire={vi.fn()}
      />,
    );
    expect(screen.getByText("05:01")).toBeInTheDocument();
    act(() => vi.advanceTimersByTime(1000));
    expect(screen.getByText("05:00")).toBeInTheDocument();
    act(() => vi.advanceTimersByTime(1000));
    act(() => vi.advanceTimersByTime(1000));
    expect(getToastsSnapshot().filter((item) => item.variant === "warning")).toHaveLength(1);
  });

  it("falls back to the booking deadline after the payment window closes, then reloads at expiry", () => {
    const onExpire = vi.fn();
    render(<Countdown expiresAt="2026-11-11T10:00:03Z" activePaymentExpiresAt="2026-11-11T10:00:01Z" onExpire={onExpire} />);
    act(() => vi.advanceTimersByTime(1000));
    expect(screen.getByText("00:02")).toBeInTheDocument();
    act(() => vi.advanceTimersByTime(1000));
    act(() => vi.advanceTimersByTime(1000));
    expect(onExpire).toHaveBeenCalledTimes(1);
    expect(screen.getByText("Batas waktu pembayaran sudah lewat.")).toBeInTheDocument();
  });
});
