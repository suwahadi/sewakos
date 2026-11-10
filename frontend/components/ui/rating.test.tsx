import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { RatingSummary } from "./rating";

describe("RatingSummary", () => {
  it("shows the average, star, and review count with an ARIA label", () => {
    render(<RatingSummary average={4.5} count={12} />);
    expect(screen.getByLabelText("Rating 4,5 dari 5, 12 ulasan")).toBeInTheDocument();
    expect(screen.getByText("4,5")).toBeInTheDocument();
    expect(screen.getByText("(12 ulasan)")).toBeInTheDocument();
  });

  it("shows a no-reviews message when average is null", () => {
    render(<RatingSummary average={null} count={0} />);
    expect(screen.getByText("Belum ada ulasan")).toBeInTheDocument();
  });
});
