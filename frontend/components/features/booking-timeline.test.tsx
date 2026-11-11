import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { BookingTimeline } from "./booking-timeline";

describe("BookingTimeline", () => {
  it("marks payment as the current step while the booking is pending", () => {
    render(<BookingTimeline booking={{ status: "pending", paid_at: null, reject_reason: null }} />);
    expect(screen.getAllByRole("listitem")).toHaveLength(4);
    expect(screen.getByText("Dibayar").closest("li")).toHaveAttribute("aria-current", "step");
  });

  it("stops at the last reached step and shows the rejection reason", () => {
    render(<BookingTimeline booking={{ status: "rejected", paid_at: "2026-11-10T10:05:00+00:00", reject_reason: "Kamar sedang direnovasi" }} />);
    expect(screen.queryByText("Disetujui pemilik")).not.toBeInTheDocument();
    expect(screen.getByText("Ditolak")).toBeInTheDocument();
    expect(screen.getByText("Kamar sedang direnovasi")).toBeInTheDocument();
  });

  it("completes every step once the stay is over", () => {
    render(<BookingTimeline booking={{ status: "completed", paid_at: "2026-11-10T10:05:00+00:00", reject_reason: null }} />);
    expect(screen.getAllByRole("listitem").every((item) => !item.hasAttribute("aria-current"))).toBe(true);
  });
});
