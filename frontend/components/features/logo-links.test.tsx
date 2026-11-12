import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

vi.mock("next/navigation", () => ({ usePathname: () => "/admin/ulasan", useRouter: () => ({ push: vi.fn() }) }));

const { AdminNav } = await import("./admin-nav");
const { default: AuthLayout } = await import("@/app/(auth)/layout");
const { default: LegalLayout } = await import("@/app/(legal)/layout");

describe("text logo", () => {
  it.each([
    ["admin panel", <AdminNav key="admin" />, "Admin Sewa Kos"],
    ["auth pages", <AuthLayout key="auth">{null}</AuthLayout>, "Sewa Kos"],
    ["legal pages", <LegalLayout key="legal">{null}</LegalLayout>, "Sewa Kos"],
  ])("links to the home page on the %s", (_name, ui, label) => {
    render(ui);
    for (const link of screen.getAllByRole("link", { name: label })) {
      expect(link).toHaveAttribute("href", "/");
    }
  });
});
