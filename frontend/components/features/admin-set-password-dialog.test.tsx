import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import type { AdminUser } from "@/lib/admin";
import { getToastsSnapshot, resetToastsForTests } from "@/lib/toast";

vi.mock("next/navigation", () => ({ useRouter: () => ({ push: vi.fn() }) }));
const callBffMock = vi.fn();
vi.mock("@/lib/api/browser", () => ({ callBff: (...args: unknown[]) => callBffMock(...args) }));

const { AdminSetPasswordDialog } = await import("./admin-set-password-dialog");

const tenant: AdminUser = {
  id: 7,
  name: "Penyewa Contoh",
  role: "tenant",
  phone_number: "081200000002",
  email: null,
  is_verified: true,
  suspended_at: null,
  created_at: "2026-11-01T00:00:00.000000Z",
};

afterEach(() => {
  callBffMock.mockReset();
  resetToastsForTests();
});

function dialog() {
  return within(screen.getByRole("dialog", { hidden: true }));
}

describe("AdminSetPasswordDialog", () => {
  it("shows inline errors and sends nothing while the fields are invalid", async () => {
    const user = userEvent.setup();
    render(<AdminSetPasswordDialog user={tenant} onClose={vi.fn()} />);

    await user.click(dialog().getByRole("button", { name: "Simpan", hidden: true }));
    expect(dialog().getByText("Password wajib diisi.")).toBeInTheDocument();
    expect(dialog().getByText("Ulangi password wajib diisi.")).toBeInTheDocument();

    await user.type(dialog().getByLabelText("Password baru"), "pendek");
    await user.type(dialog().getByLabelText("Ulangi password"), "berbeda1");
    await user.click(dialog().getByRole("button", { name: "Simpan", hidden: true }));
    expect(dialog().getByText("Password minimal 8 karakter.")).toBeInTheDocument();
    expect(dialog().getByText("Konfirmasi password tidak sama.")).toBeInTheDocument();
    expect(callBffMock).not.toHaveBeenCalled();
  });

  it("sets the password and closes", async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();
    callBffMock.mockResolvedValue({ ok: true, httpStatus: 200, data: { status: true, meta: { message: "Password pengguna diperbarui." } } });
    render(<AdminSetPasswordDialog user={tenant} onClose={onClose} />);

    await user.type(dialog().getByLabelText("Password baru"), "Rahasia123");
    await user.type(dialog().getByLabelText("Ulangi password"), "Rahasia123");
    await user.click(dialog().getByRole("button", { name: "Simpan", hidden: true }));

    expect(callBffMock).toHaveBeenCalledWith("/api/admin/users/7/password", {
      method: "POST",
      body: { password: "Rahasia123", password_confirmation: "Rahasia123" },
    });
    expect(onClose).toHaveBeenCalled();
    expect(getToastsSnapshot().map((toast) => toast.title)).toContain("Password pengguna diperbarui.");
  });
});
