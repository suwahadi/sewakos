import { act, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { resetToastsForTests, toast } from "@/lib/toast";
import { Toaster } from "./toaster";

describe("Toaster", () => {
  beforeEach(() => {
    resetToastsForTests();
  });

  afterEach(() => {
    resetToastsForTests();
    vi.useRealTimers();
  });

  it("shows at most 3 toasts, keeping the most recently pushed", () => {
    render(<Toaster />);

    act(() => {
      toast.info("Satu");
      toast.info("Dua");
      toast.info("Tiga");
      toast.info("Empat");
    });

    const statuses = screen.getAllByRole("status");
    expect(statuses).toHaveLength(3);
    expect(screen.queryByText("Satu")).not.toBeInTheDocument();
    expect(screen.getByText("Dua")).toBeInTheDocument();
    expect(screen.getByText("Tiga")).toBeInTheDocument();
    expect(screen.getByText("Empat")).toBeInTheDocument();
  });

  it("auto-dismisses after its duration and pauses the timer on hover", () => {
    vi.useFakeTimers();
    render(<Toaster />);

    act(() => {
      toast.success("Booking disetujui.", { duration: 1000 });
    });
    const card = screen.getByText("Booking disetujui.").closest('[role="status"]')!;

    fireEvent.mouseEnter(card);
    act(() => {
      vi.advanceTimersByTime(5000);
    });
    expect(screen.getByText("Booking disetujui.")).toBeInTheDocument();

    fireEvent.mouseLeave(card);
    act(() => {
      vi.advanceTimersByTime(1000);
    });
    expect(screen.queryByText("Booking disetujui.")).not.toBeInTheDocument();
  });

  it("uses a danger toast with role=alert for errors", () => {
    render(<Toaster />);
    act(() => {
      toast.error("Kamar tidak tersedia.");
    });
    expect(screen.getByRole("alert")).toHaveTextContent("Kamar tidak tersedia.");
  });
});
