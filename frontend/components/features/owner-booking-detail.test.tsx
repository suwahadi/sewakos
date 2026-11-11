import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import type { OwnerBooking, Payout } from "@/lib/owner";
import { resetToastsForTests } from "@/lib/toast";

const refreshMock = vi.fn();
vi.mock("next/navigation", () => ({ useRouter: () => ({ push: vi.fn(), refresh: refreshMock }) }));
const callBffMock = vi.fn();
vi.mock("@/lib/api/browser", () => ({ callBff: (...args: unknown[]) => callBffMock(...args) }));

const { OwnerBookingDetail } = await import("./owner-booking-detail");
const { OwnerFinance } = await import("./owner-finance");

const booking: OwnerBooking = {
  id: 2,
  status: "paid",
  rent_type: "monthly",
  start_date: "2026-12-01",
  duration: 3,
  end_date: "2027-03-01",
  tenant: { id: 5, name: "Anak Kos Contoh", phone_number: null },
  property: { id: 1, name: "Kost Asti Beji" },
  room: { id: 1, name: "Kamar Tipe A" },
  price_label: "Rp2.000.000",
  deposit_label: "Rp2.250.000",
  payout_amount_label: "Rp4.250.000",
  paid_at: "2026-11-10T10:05:00+00:00",
  owner_deadline_at: new Date(Date.now() + 5 * 3_600_000).toISOString(),
  reject_reason: null,
};

function ok(message: string) {
  return { ok: true, httpStatus: 200, data: { status: true, meta: { message } } };
}

afterEach(() => {
  callBffMock.mockReset();
  refreshMock.mockReset();
  resetToastsForTests();
});

describe("OwnerBookingDetail", () => {
  it("hides the tenant phone until approval and approves directly", async () => {
    const user = userEvent.setup();
    callBffMock.mockResolvedValue(ok("Booking disetujui."));
    render(<OwnerBookingDetail booking={booking} />);
    expect(screen.getByText("Tampil setelah disetujui")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Setujui booking" }));
    expect(callBffMock).toHaveBeenCalledWith("/api/owner/bookings/2/approve", { method: "POST", body: {} });
    expect(refreshMock).toHaveBeenCalled();
  });

  it("rejects with the optional reason from the dialog", async () => {
    const user = userEvent.setup();
    callBffMock.mockResolvedValue(ok("Booking ditolak. Dana akan dikembalikan ke penyewa."));
    render(<OwnerBookingDetail booking={booking} />);

    await user.click(screen.getAllByRole("button", { name: "Tolak booking" })[0]!);
    const dialog = screen.getByRole("dialog", { hidden: true });
    await user.type(within(dialog).getByLabelText("Alasan (opsional)"), "Kamar sedang renovasi.");
    await user.click(within(dialog).getByRole("button", { name: "Tolak booking", hidden: true }));
    expect(callBffMock).toHaveBeenCalledWith("/api/owner/bookings/2/reject", { method: "POST", body: { reason: "Kamar sedang renovasi." } });
  });

  it("shows the phone link and no actions once approved", () => {
    render(<OwnerBookingDetail booking={{ ...booking, status: "approved", tenant: { ...booking.tenant, phone_number: "081200000002" } }} />);
    expect(screen.getByRole("link", { name: "0812 0000 0002" })).toHaveAttribute("href", "tel:081200000002");
    expect(screen.queryByRole("button", { name: "Setujui booking" })).not.toBeInTheDocument();
  });
});

describe("OwnerFinance", () => {
  const payout: Payout = {
    id: 1, booking_id: 2, amount_label: "Rp4.250.000", status: "pending", transfer_reference: null, completed_at: null,
    created_at: "2026-11-10T11:00:00.000000Z",
  };

  it("prefills the bank account and saves it to the owner endpoint", async () => {
    const user = userEvent.setup();
    callBffMock.mockResolvedValue(ok("Rekening disimpan."));
    render(<OwnerFinance bankAccount={{ bank_name: "BCA", account_number: "1234567890", account_name: "Pemilik" }} initialPayouts={[payout]} />);

    expect(screen.getByLabelText("Nomor rekening")).toHaveValue("1234567890");
    await user.click(screen.getByRole("button", { name: "Simpan rekening" }));
    expect(callBffMock).toHaveBeenCalledWith("/api/owner/bank-account", {
      method: "PUT",
      body: { bank_name: "BCA", account_number: "1234567890", account_name: "Pemilik" },
    });
    expect(screen.getAllByText("Rp4.250.000").length).toBeGreaterThan(0);
    expect(screen.getAllByText("Menunggu transfer").length).toBeGreaterThan(0);
  });

  it("asks for an account when none is on file", () => {
    render(<OwnerFinance bankAccount={null} initialPayouts={[]} />);
    expect(screen.getByText("Isi rekening agar dana booking yang disetujui bisa ditransfer admin.")).toBeInTheDocument();
  });
});
