import { fireEvent, render, screen, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

const push = vi.fn();
vi.mock("next/navigation", () => ({
  useRouter: () => ({ push, back: vi.fn() }),
  usePathname: () => "/cari",
}));

beforeEach(() => {
  // jsdom doesn't implement IntersectionObserver.
  vi.stubGlobal(
    "IntersectionObserver",
    class {
      observe() {}
      disconnect() {}
      unobserve() {}
    },
  );
});

const { AppHeader } = await import("./app-header");

describe("AppHeader", () => {
  it("home variant shows the logo and a notifications button with an unread badge", () => {
    render(<AppHeader variant="home" role="tenant" unreadCount={3} />);
    expect(screen.getByRole("link", { name: "Sewa Kos" })).toBeInTheDocument();
    const bell = screen.getByRole("button", { name: "Notifikasi" });
    expect(bell).toBeInTheDocument();
    expect(screen.getByText("3")).toBeInTheDocument();
  });

  it("inner variant shows a back button and the page title", () => {
    render(<AppHeader variant="inner" role="tenant" title="Detail booking" />);
    expect(screen.getByRole("button", { name: "Kembali" })).toBeInTheDocument();
    expect(screen.getByText("Detail booking")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Sewa Kos" })).toHaveClass("hidden", "lg:inline");
  });

  it("desktop nav marks the current page and leaves the account entry to its own button", () => {
    render(<AppHeader variant="inner" role="tenant" title="Cari" />);
    const nav = screen.getByRole("navigation", { name: "Navigasi utama" });
    expect(within(nav).getByRole("link", { name: "Cari" })).toHaveAttribute("aria-current", "page");
    expect(within(nav).queryByRole("link", { name: "Akun" })).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Akun" }));
    expect(push).toHaveBeenCalledWith("/akun");
  });

  it("guests get a sign-in button and are sent to sign in from protected nav links", () => {
    push.mockClear();
    render(<AppHeader variant="home" role="guest" />);
    fireEvent.click(screen.getByRole("button", { name: "Masuk" }));
    expect(push).toHaveBeenCalledWith("/masuk");
    fireEvent.click(screen.getByRole("link", { name: "Favorit" }));
    expect(push).toHaveBeenCalledWith("/masuk?next=%2Ffavorit");
  });
});
