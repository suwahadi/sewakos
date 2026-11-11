import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

const pushMock = vi.fn();
vi.mock("next/navigation", () => ({
  usePathname: () => "/",
  useRouter: () => ({ push: pushMock, back: vi.fn() }),
}));

const { BottomNav, hasOwnStickyActionBar } = await import("./bottom-nav");

describe("BottomNav", () => {
  it("marks the current page with aria-current", () => {
    render(<BottomNav role="tenant" />);
    expect(screen.getByRole("link", { name: "Beranda" })).toHaveAttribute("aria-current", "page");
    expect(screen.getByRole("link", { name: "Cari" })).not.toHaveAttribute("aria-current");
  });

  it("shows the owner item set for role=owner", () => {
    render(<BottomNav role="owner" />);
    expect(screen.getByRole("link", { name: "Properti" })).toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Favorit" })).not.toBeInTheDocument();
  });

  it("redirects a guest to /masuk?next= for an auth-only item instead of navigating", async () => {
    const user = userEvent.setup();
    render(<BottomNav role="guest" />);

    await user.click(screen.getByRole("link", { name: "Favorit" }));
    expect(pushMock).toHaveBeenCalledWith("/masuk?next=%2Ffavorit");
  });

  it("hides itself on the listing detail and rent form pages", () => {
    expect(hasOwnStickyActionBar("/kost/1")).toBe(true);
    expect(hasOwnStickyActionBar("/kost/1/sewa")).toBe(true);
    expect(hasOwnStickyActionBar("/kost/1/ulasan")).toBe(false);
  });
});
