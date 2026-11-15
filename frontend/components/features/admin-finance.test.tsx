import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import type { AdminFinanceSummary, AdminLedgerEntry, AdminReconciliationIssue } from "@/lib/admin";
import { resetToastsForTests } from "@/lib/toast";

vi.mock("next/navigation", () => ({ useRouter: () => ({ push: vi.fn() }) }));
const callBffMock = vi.fn();
vi.mock("@/lib/api/browser", () => ({ callBff: (...args: unknown[]) => callBffMock(...args) }));

const { AdminFinanceSummaryAndLedger, AdminReconciliationList } = await import("./admin-finance");

function ok(data: object = {}, message = "OK") {
  return { ok: true, httpStatus: 200, data: { status: true, meta: { message }, ...data } };
}

afterEach(() => {
  callBffMock.mockReset();
  resetToastsForTests();
});

const summary: AdminFinanceSummary = {
  from: "2026-11-01",
  to: "2026-11-30",
  payments_received_label: "Rp12.750.000",
  admin_fee_revenue_label: "Rp15.000",
  refunds_completed_label: "Rp4.250.000",
  refunds_outstanding_label: "Rp0",
  payouts_completed_label: "Rp4.250.000",
  payouts_outstanding_label: "Rp4.250.000",
  open_reconciliation_issues: 0,
};

const ledgerEntry: AdminLedgerEntry = {
  id: 1,
  occurred_at: "2026-11-10T11:00:00+00:00",
  type: "payment_received",
  amount: 4250000,
  amount_label: "Rp4.250.000",
  owner_id: 1,
  owner_name: "Pemilik Contoh",
  booking_id: 2,
  invoice_number: "INV/2026/11/000002",
  description: "Tagihan dibayar",
};

describe("AdminFinanceSummaryAndLedger", () => {
  it("shows the summary cards, the ledger, and a CSV link that follows the type filter", async () => {
    const user = userEvent.setup();
    callBffMock.mockResolvedValue(ok({ entries: [{ ...ledgerEntry, type: "owner_earning" }] }));
    render(<AdminFinanceSummaryAndLedger summary={summary} initialLedger={[ledgerEntry]} owner={null} />);

    expect(screen.getByText("Rp12.750.000")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "CSV" })).toHaveAttribute("href", "/api/admin/finance/ledger.csv");

    await user.click(screen.getByRole("tab", { name: "Pendapatan pemilik" }));
    expect(callBffMock).toHaveBeenCalledWith("/api/admin/finance/ledger?type=owner_earning&page=1&limit=20");
    expect(await screen.findByRole("link", { name: "CSV" })).toHaveAttribute("href", "/api/admin/finance/ledger.csv?type=owner_earning");
  });
});

describe("AdminFinanceSummaryAndLedger per owner", () => {
  it("links each entry to its owner's finance view", () => {
    render(<AdminFinanceSummaryAndLedger summary={summary} initialLedger={[ledgerEntry]} owner={null} />);

    expect(screen.getByRole("link", { name: "Pemilik Contoh" })).toHaveAttribute("href", "/admin/keuangan?user=1");
  });

  it("keeps the owner filter on the ledger, its tabs, and the CSV link", async () => {
    const user = userEvent.setup();
    callBffMock.mockResolvedValue(ok({ entries: [] }));
    render(<AdminFinanceSummaryAndLedger summary={summary} initialLedger={[ledgerEntry]} owner={{ id: 1, name: "Pemilik Contoh" }} />);

    expect(screen.getByText("Pemilik: Pemilik Contoh")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Hapus filter pemilik Pemilik Contoh" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "CSV" })).toHaveAttribute("href", "/api/admin/finance/ledger.csv?owner_id=1");
    expect(screen.queryByRole("link", { name: "Pemilik Contoh" })).not.toBeInTheDocument();

    await user.click(screen.getByRole("tab", { name: "Pendapatan pemilik" }));
    expect(callBffMock).toHaveBeenCalledWith("/api/admin/finance/ledger?type=owner_earning&owner_id=1&page=1&limit=20");
  });

  it("searches owners by name and links to their finance view", async () => {
    const user = userEvent.setup();
    callBffMock.mockResolvedValue(ok({ users: [{ id: 7, name: "Budi Pemilik", role: "owner" }] }));
    render(<AdminFinanceSummaryAndLedger summary={summary} initialLedger={[ledgerEntry]} owner={null} />);

    await user.type(screen.getByLabelText("Filter per pemilik"), "Bu");
    await user.click(screen.getByRole("button", { name: "Cari" }));
    expect(screen.getByText("Masukkan minimal 3 karakter.")).toBeInTheDocument();
    expect(callBffMock).not.toHaveBeenCalled();

    await user.type(screen.getByLabelText("Filter per pemilik"), "di");
    await user.click(screen.getByRole("button", { name: "Cari" }));
    expect(callBffMock).toHaveBeenCalledWith("/api/admin/users?role=owner&q=Budi&limit=5");
    expect(await screen.findByRole("link", { name: "Budi Pemilik" })).toHaveAttribute("href", "/admin/keuangan?user=7");
  });
});

describe("AdminReconciliationList", () => {
  const issue: AdminReconciliationIssue = {
    id: 3,
    order_id: "KOS-7-12",
    kind: "status_mismatch",
    local_status: "pending",
    midtrans_status: "settlement",
    local_amount: 2250000,
    midtrans_amount: 2250000,
    status: "open",
    note: null,
    detected_at: "2026-12-02T19:00:00+00:00",
  };

  it("resolves an open issue with a note", async () => {
    const user = userEvent.setup();
    callBffMock.mockResolvedValue(ok({}, "Selisih ditandai selesai."));
    render(<AdminReconciliationList initial={[issue]} />);

    await user.click(screen.getAllByRole("button", { name: "Tandai selesai" })[0]!);
    const dialog = screen.getByRole("dialog", { hidden: true });
    await user.click(within(dialog).getByRole("button", { name: "Tandai selesai", hidden: true }));
    expect(within(dialog).getByText("Catatan wajib diisi.")).toBeInTheDocument();
    expect(callBffMock).not.toHaveBeenCalled();

    await user.type(within(dialog).getByLabelText("Catatan"), "Sudah dicek manual, cocok.");
    await user.click(within(dialog).getByRole("button", { name: "Tandai selesai", hidden: true }));
    expect(callBffMock).toHaveBeenCalledWith("/api/admin/finance/reconciliation/3/resolve", {
      method: "POST",
      body: { note: "Sudah dicek manual, cocok." },
    });
  });
});
