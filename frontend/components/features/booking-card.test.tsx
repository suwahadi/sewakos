import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import type { BookingSummary } from "@/lib/booking";
import { BookingCard } from "./booking-card";

const booking: BookingSummary = {
  id: 2,
  status: "pending",
  rent_type: "monthly",
  room: { id: 1, name: "Kamar Tipe A" },
  start_date: "2026-12-01",
  duration: 3,
  end_date: "2027-03-01",
  price_label: "Rp2.000.000",
  expires_at: "2026-11-11T10:00:00+00:00",
};

describe("BookingCard", () => {
  it("links to the booking and shows the last day of stay", () => {
    render(<BookingCard booking={booking} />);
    expect(screen.getByRole("link")).toHaveAttribute("href", "/booking/2");
    expect(screen.getByText("1 Des 2026 – 28 Feb 2027")).toBeInTheDocument();
  });

  it("reminds a pending booking to pay before the deadline", () => {
    render(<BookingCard booking={booking} />);
    expect(screen.getByRole("status")).toHaveTextContent("Bayar sebelum 11 Nov 2026, 17.00 WIB");
  });

  it("drops the payment reminder once paid", () => {
    render(<BookingCard booking={{ ...booking, status: "paid" }} />);
    expect(screen.getByText("Menunggu konfirmasi pemilik")).toBeInTheDocument();
    expect(screen.queryByRole("status")).not.toBeInTheDocument();
  });
});
