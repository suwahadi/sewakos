import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import type { AdminRefund, AdminUser } from "@/lib/admin";
import { resetToastsForTests } from "@/lib/toast";

const refreshMock = vi.fn();
vi.mock("next/navigation", () => ({ useRouter: () => ({ push: vi.fn(), refresh: refreshMock }), usePathname: () => "/admin/refund" }));
const callBffMock = vi.fn();
vi.mock("@/lib/api/browser", () => ({ callBff: (...args: unknown[]) => callBffMock(...args) }));

const { AdminPropertyActions } = await import("./admin-property-actions");
const { AdminRefundList } = await import("./admin-transfers");
const { AdminUserList } = await import("./admin-user-list");
const { AdminAreaList } = await import("./admin-area-list");
const { AdminSettings } = await import("./admin-settings");

function ok(data: object = {}, message = "OK") {
  return { ok: true, httpStatus: 200, data: { status: true, meta: { message }, ...data } };
}

afterEach(() => {
  callBffMock.mockReset();
  refreshMock.mockReset();
  resetToastsForTests();
});

describe("AdminPropertyActions", () => {
  it("requires a reason before rejecting", async () => {
    const user = userEvent.setup();
    callBffMock.mockResolvedValue(ok({}, "Properti ditolak."));
    render(<AdminPropertyActions propertyId={3} status="pending" />);
    await user.click(screen.getByRole("button", { name: "Tolak properti" }));
    const dialog = screen.getByRole("dialog", { hidden: true });
    await user.click(within(dialog).getByRole("button", { name: "Tolak properti", hidden: true }));
    expect(within(dialog).getByText("Alasan wajib diisi.")).toBeInTheDocument();
    expect(callBffMock).not.toHaveBeenCalled();

    await user.type(within(dialog).getByLabelText("Alasan"), "Foto kurang jelas");
    await user.click(within(dialog).getByRole("button", { name: "Tolak properti", hidden: true }));
    expect(callBffMock).toHaveBeenCalledWith("/api/admin/properties/3/reject", { method: "POST", body: { reason: "Foto kurang jelas" } });
    expect(refreshMock).toHaveBeenCalled();
  });

  it("only offers reactivation for a suspended property", () => {
    render(<AdminPropertyActions propertyId={3} status="suspended" />);
    expect(screen.getByRole("button", { name: "Aktifkan kembali" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Tolak properti" })).not.toBeInTheDocument();
  });
});

describe("AdminRefundList", () => {
  const refund: AdminRefund = {
    id: 8, booking_id: 2, tenant: { id: 5, name: "Anak Kos" }, amount_label: "Rp4.250.000", reason: "owner_rejected",
    status: "ready", bank_account: { bank_name: "BCA", account_number: "1234567890", account_name: "Anak Kos" },
    transfer_reference: null, completed_at: null, created_at: "2026-11-10T12:00:00.000000Z",
  };

  it("records the transfer reference and reloads the list", async () => {
    const user = userEvent.setup();
    callBffMock.mockImplementation(async (path: string) =>
      path.endsWith("/complete") ? ok({ refund: { ...refund, status: "completed" } }, "Refund ditandai selesai.") : ok({ refunds: [] }),
    );
    render(<AdminRefundList initial={[refund]} />);
    expect(screen.getAllByText("BCA 1234567890 a.n. Anak Kos").length).toBeGreaterThan(0);

    await user.click(screen.getAllByRole("button", { name: "Tandai selesai" })[0]!);
    const dialog = screen.getByRole("dialog", { hidden: true });
    await user.type(within(dialog).getByLabelText("Nomor referensi transfer"), "TRX-1");
    await user.click(within(dialog).getByRole("button", { name: "Tandai selesai", hidden: true }));
    expect(callBffMock).toHaveBeenCalledWith("/api/admin/refunds/8/complete", { method: "POST", body: { transfer_reference: "TRX-1", note: undefined } });
    expect(callBffMock).toHaveBeenLastCalledWith("/api/admin/refunds?status=ready&page=1&limit=20");
  });
});

describe("AdminUserList", () => {
  const tenant: AdminUser = {
    id: 4, name: "Budi", role: "tenant", phone_number: "081200000002", email: null, is_verified: true, suspended_at: null,
    created_at: "2026-11-10T10:00:00.000000Z",
  };

  it("asks for at least three characters before searching", async () => {
    const user = userEvent.setup();
    render(<AdminUserList initial={[tenant]} />);
    await user.type(screen.getByLabelText("Cari nama atau nomor HP"), "08");
    await user.click(screen.getByRole("button", { name: "Cari" }));
    expect(screen.getByText("Masukkan minimal 3 karakter.")).toBeInTheDocument();
    expect(callBffMock).not.toHaveBeenCalled();
  });

  it("suspends after confirmation", async () => {
    const user = userEvent.setup();
    callBffMock.mockResolvedValue(ok({ users: [] }, "Pengguna dinonaktifkan."));
    render(<AdminUserList initial={[tenant]} />);
    await user.click(screen.getAllByRole("button", { name: "Nonaktifkan" })[0]!);
    const dialog = screen.getByRole("dialog", { hidden: true });
    await user.click(within(dialog).getByRole("button", { name: "Nonaktifkan", hidden: true }));
    expect(callBffMock).toHaveBeenCalledWith("/api/admin/users/4/suspend", { method: "POST" });
  });
});

describe("AdminAreaList", () => {
  it("creates an area with numeric coordinates", async () => {
    const user = userEvent.setup();
    callBffMock.mockResolvedValue(ok({ area: {}, areas: [] }, "Area ditambahkan."));
    render(<AdminAreaList initial={[]} />);
    await user.click(screen.getByRole("button", { name: "Tambah area" }));
    const dialog = screen.getByRole("dialog", { hidden: true });
    await user.type(within(dialog).getByLabelText("Nama"), "Kota Bogor");
    await user.type(within(dialog).getByLabelText("Slug"), "kota-bogor");
    await user.type(within(dialog).getByLabelText("Latitude"), "-6.59");
    await user.type(within(dialog).getByLabelText("Longitude"), "106.8");
    await user.click(within(dialog).getByRole("button", { name: "Simpan area", hidden: true }));
    expect(callBffMock).toHaveBeenCalledWith("/api/admin/areas", {
      method: "POST",
      body: { name: "Kota Bogor", slug: "kota-bogor", type: "city", is_top: false, latitude: -6.59, longitude: 106.8, place_id: null },
    });
  });
});

describe("AdminSettings", () => {
  it("saves the admin fee as a plain number", async () => {
    const user = userEvent.setup();
    callBffMock.mockResolvedValue(ok({ settings: { admin_fee: 5000 } }, "Pengaturan disimpan."));
    render(<AdminSettings adminFee={0} />);
    const field = screen.getByLabelText("Biaya admin per tagihan");
    await user.clear(field);
    await user.type(field, "5000");
    expect(field).toHaveValue("5.000");
    await user.click(screen.getByRole("button", { name: "Simpan biaya admin" }));
    expect(callBffMock).toHaveBeenCalledWith("/api/admin/settings", { method: "PATCH", body: { admin_fee: 5000 } });
  });

  it("confirms before broadcasting", async () => {
    const user = userEvent.setup();
    callBffMock.mockResolvedValue(ok({ sent: 12 }, "Notifikasi dikirim."));
    render(<AdminSettings adminFee={0} />);
    await user.type(screen.getByLabelText("Judul"), "Promo");
    await user.type(screen.getByLabelText("Isi pesan"), "Diskon bulan ini");
    await user.click(screen.getByRole("button", { name: "Kirim notifikasi" }));
    expect(callBffMock).not.toHaveBeenCalled();
    const dialog = screen.getByRole("dialog", { hidden: true });
    await user.click(within(dialog).getByRole("button", { name: "Kirim", hidden: true }));
    expect(callBffMock).toHaveBeenCalledWith("/api/admin/notifications/broadcast", {
      method: "POST",
      body: { target: "tenant", title: "Promo", body: "Diskon bulan ini" },
    });
  });
});
