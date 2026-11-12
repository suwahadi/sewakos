import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { AreaBannerGrid, areaPhotoUrl } from "./area-banner-grid";

describe("AreaBannerGrid", () => {
  it("links each city card to its search results", () => {
    render(<AreaBannerGrid areas={[{ id: 7, name: "Kota Depok", slug: "kota-depok" }]} />);
    expect(screen.getByRole("link", { name: "Kota Depok" })).toHaveAttribute(
      "href",
      "/cari?area_id=7&area_name=Kota%20Depok",
    );
  });

  it("uses the city photo when one is mapped and a plain brand card otherwise", () => {
    expect(areaPhotoUrl("kota-bandung")).toMatch(/^https:\/\/images\.unsplash\.com\/photo-/);
    expect(areaPhotoUrl("kota-baru")).toBeNull();

    const { container } = render(<AreaBannerGrid areas={[{ id: 9, name: "Kota Baru", slug: "kota-baru" }]} />);
    expect(container.querySelector("img")).toBeNull();
    expect(screen.getByRole("link", { name: "Kota Baru" })).toHaveClass("bg-brand-600");
  });

  it("renders nothing without areas", () => {
    const { container } = render(<AreaBannerGrid areas={[]} />);
    expect(container).toBeEmptyDOMElement();
  });
});
