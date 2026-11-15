import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

vi.mock("next/navigation", () => ({ useRouter: () => ({ push: vi.fn(), refresh: vi.fn() }) }));
vi.mock("@/lib/api/browser", () => ({ callBff: vi.fn() }));

const { AkunClient } = await import("./akun-client");

const user = { id: 2, name: "Penyewa Contoh", role: "tenant", phone_number: "081200000002", email: null, is_verified: true };

describe("AkunClient", () => {
  it("opens on the profile tab with the logout button below it", () => {
    render(<AkunClient user={user} />);

    expect(screen.getByRole("tab", { name: "Profil" })).toHaveAttribute("aria-selected", "true");
    expect(screen.getByLabelText("Nama")).toBeVisible();
    expect(screen.queryByLabelText("Password saat ini")).not.toBeVisible();
    expect(screen.getByRole("button", { name: "Keluar" })).toBeVisible();
  });

  it("moves password and phone changes to the settings tab and keeps the logout button", async () => {
    const userEvents = userEvent.setup();
    render(<AkunClient user={user} />);

    await userEvents.type(screen.getByLabelText("Nama"), " Baru");
    await userEvents.click(screen.getByRole("tab", { name: "Pengaturan" }));

    expect(screen.getByLabelText("Password saat ini")).toBeVisible();
    expect(screen.getByLabelText("Nomor HP baru")).toBeVisible();
    expect(screen.queryByLabelText("Nama")).not.toBeVisible();
    expect(screen.getByRole("button", { name: "Keluar" })).toBeVisible();

    await userEvents.click(screen.getByRole("tab", { name: "Profil" }));
    expect(screen.getByLabelText("Nama")).toHaveValue("Penyewa Contoh Baru");
  });
});
