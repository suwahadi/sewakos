import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { ListingCard, type ListingCardRoom } from "./listing-card";

const baseRoom: ListingCardRoom = {
  id: 1,
  property_id: 1,
  name: "Kamar Tipe A",
  property_name: "Kost Asti Beji",
  area_name: "Kota Depok",
  thumbnail_url: null,
  gender: "campur",
  price_label: "Rp2.250.000",
  available: 2,
  is_flash_sale: true,
  rating_average: 4.5,
  rating_count: 12,
};

describe("ListingCard", () => {
  it("links to the property detail page and shows the rent-type price suffix", () => {
    render(<ListingCard room={baseRoom} rentType={3} />);
    expect(screen.getByRole("link")).toHaveAttribute("href", "/kost/1");
    expect(screen.getByText("/bulan")).toBeInTheDocument();
  });

  it("hides the rating when rating_count is 0", () => {
    render(<ListingCard room={{ ...baseRoom, rating_count: 0, rating_average: null }} />);
    expect(screen.queryByText(/ulasan|\(\d+\)/)).not.toBeInTheDocument();
  });

  it("shows a remaining-rooms label for 1-3 available, and Penuh for 0", () => {
    const { rerender } = render(<ListingCard room={{ ...baseRoom, available: 2 }} />);
    expect(screen.getByText("Sisa 2 kamar")).toBeInTheDocument();

    rerender(<ListingCard room={{ ...baseRoom, available: 0 }} />);
    expect(screen.getByText("Penuh")).toBeInTheDocument();
  });

  it("shows no availability label for 4 or more available rooms", () => {
    render(<ListingCard room={{ ...baseRoom, available: 4 }} />);
    expect(screen.queryByText(/kamar$/)).not.toBeInTheDocument();
    expect(screen.queryByText("Penuh")).not.toBeInTheDocument();
  });
});
