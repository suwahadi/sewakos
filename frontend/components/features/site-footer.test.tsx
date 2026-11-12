import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { SiteFooter } from "./site-footer";

describe("SiteFooter", () => {
  it("links only to pages that exist", () => {
    render(<SiteFooter />);
    expect(screen.getByRole("link", { name: "Sewa Kos" })).toHaveAttribute("href", "/");
    expect(screen.getByRole("link", { name: "Cari kost" })).toHaveAttribute("href", "/cari");
    expect(screen.getByRole("link", { name: "Syarat & Ketentuan" })).toHaveAttribute("href", "/syarat-ketentuan");
    expect(screen.getByRole("link", { name: "Kebijakan Privasi" })).toHaveAttribute("href", "/kebijakan-privasi");
  });

  it("ends with a small, light, centered copyright line", () => {
    render(<SiteFooter />);
    const copyright = screen.getByText("Copyright © 2026 - All Rights Reserved");
    expect(copyright).toHaveClass("text-center", "text-caption", "font-light");
  });
});
