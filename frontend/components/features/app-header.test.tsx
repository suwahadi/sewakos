import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("next/navigation", () => ({
  useRouter: () => ({ push: vi.fn(), back: vi.fn() }),
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
    expect(screen.queryByRole("link", { name: "Sewa Kos" })).not.toBeInTheDocument();
  });
});
