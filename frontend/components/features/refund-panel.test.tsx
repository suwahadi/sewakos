import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";

vi.mock("next/navigation", () => ({ useRouter: () => ({ push: vi.fn() }) }));

const callBffMock = vi.fn();
vi.mock("@/lib/api/browser", () => ({ callBff: (...args: unknown[]) => callBffMock(...args) }));

const { RefundPanel } = await import("./refund-panel");

const awaiting = { amount_label: "Rp4.250.000", status: "awaiting_account" as const, completed_at: null };

describe("RefundPanel", () => {
  afterEach(() => callBffMock.mockReset());

  it("validates the account in the browser before saving", async () => {
    const user = userEvent.setup();
    render(<RefundPanel refund={awaiting} bookingId={2} onSaved={vi.fn()} />);
    expect(screen.getByText("Isi rekening untuk menerima pengembalian dana Rp4.250.000.")).toBeInTheDocument();

    await user.type(screen.getByLabelText("Nomor rekening"), "12ab");
    await user.click(screen.getByRole("button", { name: "Simpan rekening" }));
    expect(screen.getByText("Nama bank wajib diisi.")).toBeInTheDocument();
    expect(screen.getByText("Nomor rekening harus berisi 5–20 digit angka.")).toBeInTheDocument();
    expect(screen.getByLabelText("Nama bank")).toHaveFocus();
    expect(callBffMock).not.toHaveBeenCalled();
  });

  it("saves a valid account and reloads the booking", async () => {
    const user = userEvent.setup();
    const onSaved = vi.fn();
    callBffMock.mockResolvedValue({
      ok: true,
      httpStatus: 200,
      data: { status: true, meta: { message: "Rekening pengembalian dana disimpan." } },
    });
    render(<RefundPanel refund={awaiting} bookingId={2} onSaved={onSaved} />);

    await user.type(screen.getByLabelText("Nama bank"), "BCA");
    await user.type(screen.getByLabelText("Nomor rekening"), "1234567890");
    await user.type(screen.getByLabelText("Nama pemilik rekening"), "Anak Kos Contoh");
    await user.click(screen.getByRole("button", { name: "Simpan rekening" }));

    expect(callBffMock).toHaveBeenCalledWith("/api/booking/2/refund-account", {
      method: "PUT",
      body: { bank_name: "BCA", account_number: "1234567890", account_name: "Anak Kos Contoh" },
    });
    expect(onSaved).toHaveBeenCalled();
  });

  it("describes processed and completed refunds without a form", () => {
    const { rerender } = render(
      <RefundPanel refund={{ amount_label: "Rp1.000", status: "ready", completed_at: null }} bookingId={2} onSaved={vi.fn()} />,
    );
    expect(screen.getByText("Pengembalian dana Rp1.000 sedang diproses admin.")).toBeInTheDocument();

    rerender(
      <RefundPanel
        refund={{ amount_label: "Rp1.000", status: "completed", completed_at: "2026-12-01T03:00:00+00:00" }}
        bookingId={2}
        onSaved={vi.fn()}
      />,
    );
    expect(screen.getByText("Dana Rp1.000 sudah dikembalikan pada 1 Des 2026.")).toBeInTheDocument();
    expect(screen.queryByRole("textbox")).not.toBeInTheDocument();
  });
});
