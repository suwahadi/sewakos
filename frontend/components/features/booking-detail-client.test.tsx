import { act, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import type { BookingDetail, Invoice } from "@/lib/booking";
import { resetToastsForTests } from "@/lib/toast";

const refreshMock = vi.fn();
vi.mock("next/navigation", () => ({ useRouter: () => ({ push: vi.fn(), refresh: refreshMock }) }));
vi.mock("next/script", () => ({ default: () => null }));

const callBffMock = vi.fn();
vi.mock("@/lib/api/browser", () => ({ callBff: (...args: unknown[]) => callBffMock(...args) }));

const { BookingDetailClient } = await import("./booking-detail-client");

const booking: BookingDetail = {
  id: 2,
  status: "pending",
  rent_type: "monthly",
  room: { id: 1, name: "Kamar Tipe A" },
  property: { id: 1, name: "Kost Asti Beji" },
  start_date: "2026-12-01",
  duration: 3,
  end_date: "2027-03-01",
  price_label: "Rp2.000.000",
  deposit_label: "Rp2.250.000",
  admin_fee: 5000,
  admin_fee_label: "Rp5.000",
  total_payment_label: "Rp4.255.000",
  expires_at: new Date(Date.now() + 23 * 3_600_000).toISOString(),
  paid_at: null,
  reject_reason: null,
};

const invoice: Invoice = {
  id: 7,
  number: "INV/2026/11/000007",
  type: "initial",
  status: "unpaid",
  period_start: "2026-12-01",
  period_end: "2027-01-01",
  total_label: "Rp4.255.000",
  due_date: "2026-11-12",
};

function ok(data: object) {
  return { ok: true, httpStatus: 200, data: { status: true, meta: { message: "OK" }, ...data } };
}

describe("BookingDetailClient", () => {
  afterEach(() => {
    callBffMock.mockReset();
    refreshMock.mockReset();
    resetToastsForTests();
    delete window.snap;
    vi.useRealTimers();
  });

  it("opens Snap with the checkout token and polls the booking after it closes", async () => {
    const user = userEvent.setup();
    const pay = vi.fn();
    window.snap = { pay };
    callBffMock.mockResolvedValue(ok({ payment: { snap_token: "snap-token", redirect_url: "https://example.test/pay" } }));
    render(<BookingDetailClient booking={booking} payment={null} refund={null} invoices={[invoice]} />);

    await user.click(screen.getByRole("button", { name: "Bayar sekarang" }));
    expect(callBffMock).toHaveBeenCalledWith("/api/booking/2/checkout", { method: "POST" });
    expect(pay).toHaveBeenCalledWith("snap-token", expect.any(Object));

    vi.useFakeTimers();
    act(() => pay.mock.calls[0]![1].onClose());
    expect(screen.getByText("Memeriksa status pembayaran…")).toBeInTheDocument();
    act(() => vi.advanceTimersByTime(3000));
    expect(refreshMock).toHaveBeenCalledTimes(1);
  });

  it("pays an unpaid invoice through its own checkout", async () => {
    const user = userEvent.setup();
    window.snap = { pay: vi.fn() };
    callBffMock.mockResolvedValue(ok({ payment: { snap_token: "t", redirect_url: "https://example.test/pay" } }));
    render(<BookingDetailClient booking={{ ...booking, status: "approved" }} payment={null} refund={null} invoices={[invoice]} />);

    expect(screen.queryByRole("button", { name: "Bayar sekarang" })).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Bayar" }));
    expect(callBffMock).toHaveBeenCalledWith("/api/invoices/7/checkout", { method: "POST" });
  });

  it("warns about the admin fee before cancelling a paid booking", async () => {
    const user = userEvent.setup();
    callBffMock.mockResolvedValue(ok({ booking: { id: 2, status: "cancelled" } }));
    render(
      <BookingDetailClient
        booking={{ ...booking, status: "paid", paid_at: "2026-11-10T10:05:00+00:00" }}
        payment={null}
        refund={null}
        invoices={[]}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Batalkan booking" }));
    expect(
      screen.getByText("Dana yang sudah kamu bayar dikembalikan setelah dikurangi biaya admin Rp5.000."),
    ).toBeInTheDocument();
    const buttons = screen.getAllByRole("button", { name: "Batalkan booking" });
    await user.click(buttons[buttons.length - 1]!);
    expect(callBffMock).toHaveBeenCalledWith("/api/booking/2/cancel", { method: "POST" });
    expect(refreshMock).toHaveBeenCalled();
  });
});
