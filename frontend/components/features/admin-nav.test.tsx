import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

vi.mock("next/navigation", () => ({ usePathname: () => "/admin/refund" }));

const { AdminNav } = await import("./admin-nav");

describe("AdminNav", () => {
  it("renders every menu item with an icon in the sidebar and marks the current page", () => {
    const { container } = render(<AdminNav />);
    const sidebar = container.querySelector("aside") as HTMLElement;
    const links = within(sidebar).getAllByRole("link", { hidden: true }).filter((link) => link.closest("nav"));

    expect(links.map((link) => link.textContent)).toEqual([
      "Moderasi properti",
      "Moderasi ulasan",
      "Booking",
      "Tagihan",
      "Refund",
      "Payout",
      "Keuangan",
      "Rekonsiliasi",
      "Riwayat",
      "Pengguna",
      "Area",
      "Pengaturan",
      "Akun",
    ]);
    for (const link of links) {
      expect(link.querySelector("svg[aria-hidden='true']")).not.toBeNull();
    }
    expect(within(sidebar).getByRole("link", { name: "Refund", hidden: true })).toHaveAttribute("aria-current", "page");
    expect(within(sidebar).getByRole("link", { name: "Payout", hidden: true })).not.toHaveAttribute("aria-current");
  });

  it("opens the drawer from the menu button and closes it with the close button", async () => {
    const user = userEvent.setup();
    const { container } = render(<AdminNav />);
    const drawer = container.querySelector("dialog") as HTMLDialogElement;
    const menuButton = screen.getByRole("button", { name: "Buka menu" });

    expect(drawer).not.toHaveAttribute("open");
    expect(menuButton).toHaveAttribute("aria-expanded", "false");
    expect(menuButton).toHaveAttribute("aria-controls", drawer.id);

    await user.click(menuButton);
    expect(drawer).toHaveAttribute("open");
    expect(menuButton).toHaveAttribute("aria-expanded", "true");

    await user.click(within(drawer).getByRole("button", { name: "Tutup menu" }));
    expect(drawer).not.toHaveAttribute("open");
  });

  it("closes the drawer when a menu item is chosen", async () => {
    const user = userEvent.setup();
    const { container } = render(<AdminNav />);
    const drawer = container.querySelector("dialog") as HTMLDialogElement;

    await user.click(screen.getByRole("button", { name: "Buka menu" }));
    const keuangan = within(drawer).getByRole("link", { name: "Keuangan" });
    keuangan.addEventListener("click", (event) => event.preventDefault());
    await user.click(keuangan);

    expect(drawer).not.toHaveAttribute("open");
  });
});
