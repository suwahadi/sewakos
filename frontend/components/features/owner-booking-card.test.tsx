import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import type { OwnerBooking } from "@/lib/owner";

vi.mock("next/navigation", () => ({ useRouter: () => ({ push: vi.fn() }) }));
const callBffMock = vi.fn();
vi.mock("@/lib/api/browser", () => ({ callBff: (...args: unknown[]) => callBffMock(...args) }));

const { OwnerBookingCard } = await import("./owner-booking-card");
const { OwnerBookingList } = await import("./owner-booking-list");

function booking(overrides: Partial<OwnerBooking> = {}): OwnerBooking {
  return {
    id: 2,
    status: "paid",
    rent_type: "monthly",
    start_date: "2026-12-01",
    duration: 3,
    end_date: "2027-03-01",
    tenant: { id: 5, name: "Anak Kos Contoh", phone_number: null },
    property: { id: 1, name: "Kost Asti Beji" },
    room: { id: 1, name: "Kamar Tipe A" },
    price_label: "Rp2.000.000",
    deposit_label: "Rp2.250.000",
    payout_amount_label: "Rp4.250.000",
    paid_at: "2026-11-10T10:05:00+00:00",
    owner_deadline_at: new Date(Date.now() + 90 * 60_000).toISOString(),
    reject_reason: null,
    ...overrides,
  };
}

describe("OwnerBookingCard", () => {
  it("links to the booking and flags a response deadline under two hours", () => {
    render(<OwnerBookingCard booking={booking()} />);
    expect(screen.getByRole("link")).toHaveAttribute("href", "/owner/booking/2");
    expect(screen.getByText("Rp4.250.000")).toBeInTheDocument();
    expect(screen.getByText(/Respons dalam/)).toHaveClass("text-danger");
  });

  it("keeps a distant deadline neutral and hides it once approved", () => {
    const { rerender } = render(<OwnerBookingCard booking={booking({ owner_deadline_at: new Date(Date.now() + 20 * 3_600_000).toISOString() })} />);
    expect(screen.getByText(/Respons dalam/)).not.toHaveClass("text-danger");
    rerender(<OwnerBookingCard booking={booking({ status: "approved" })} />);
    expect(screen.queryByText(/Respons dalam/)).not.toBeInTheDocument();
  });
});

describe("OwnerBookingList", () => {
  afterEach(() => callBffMock.mockReset());

  it("reloads the first page for the chosen status tab", async () => {
    const user = userEvent.setup();
    callBffMock.mockResolvedValue({ ok: true, httpStatus: 200, data: { status: true, bookings: [booking({ id: 9, status: "approved" })] } });
    render(<OwnerBookingList initial={[booking()]} />);

    await user.click(screen.getByRole("tab", { name: "Aktif" }));
    expect(callBffMock).toHaveBeenCalledWith("/api/owner/bookings?status=approved&page=1&limit=20");
    expect(await screen.findByText("#9")).toBeInTheDocument();
    expect(screen.queryByText("#2")).not.toBeInTheDocument();
  });
});
