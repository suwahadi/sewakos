import { render, screen } from "@testing-library/react";
import { beforeAll, describe, expect, it, vi } from "vitest";

vi.mock("next/navigation", () => ({ usePathname: () => "/admin/ulasan", useRouter: () => ({ push: vi.fn() }) }));
vi.mock("@/lib/auth/nav-role", () => ({ getNavRole: async () => "guest" }));
vi.mock("@/lib/notifications", () => ({ getUnreadCount: async () => 0 }));

beforeAll(() => {
  vi.stubGlobal(
    "IntersectionObserver",
    class {
      observe() {}
      disconnect() {}
      unobserve() {}
    },
  );
});

const { AdminNav } = await import("./admin-nav");
const { default: AuthLayout } = await import("@/app/(auth)/layout");
const { default: LegalLayout } = await import("@/app/(legal)/layout");

describe("text logo", () => {
  it.each([
    ["admin panel", async () => <AdminNav />, "Admin Sewa Kos"],
    ["auth pages", async () => <AuthLayout>{null}</AuthLayout>, "Sewa Kos"],
    ["legal pages", () => LegalLayout({ children: null }), "Sewa Kos"],
  ])("links to the home page on the %s", async (_name, ui, label) => {
    render(await ui());
    const links = screen.getAllByRole("link", { name: label });
    expect(links.length).toBeGreaterThan(0);
    for (const link of links) {
      expect(link).toHaveAttribute("href", "/");
    }
  });
});
