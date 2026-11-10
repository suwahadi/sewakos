import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { ConfirmDialog } from "./dialog";

describe("ConfirmDialog", () => {
  it("renders the question, consequence, and a 50:50 Batal/action pair", () => {
    render(
      <ConfirmDialog
        open
        onClose={() => {}}
        title="Batalkan booking?"
        description="Dana akan dikembalikan setelah dikurangi biaya admin."
        confirmLabel="Batalkan booking"
        onConfirm={() => {}}
      />,
    );

    expect(screen.getByRole("dialog", { name: "Batalkan booking?" })).toBeInTheDocument();
    expect(screen.getByText("Dana akan dikembalikan setelah dikurangi biaya admin.")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Batal" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Batalkan booking" })).toBeInTheDocument();
  });

  it("calls onConfirm and onClose from their respective buttons", async () => {
    const user = userEvent.setup();
    const onConfirm = vi.fn();
    const onClose = vi.fn();
    render(
      <ConfirmDialog
        open
        onClose={onClose}
        title="Hapus foto?"
        description="Tindakan ini tidak bisa dibatalkan."
        confirmLabel="Hapus"
        onConfirm={onConfirm}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Hapus" }));
    expect(onConfirm).toHaveBeenCalledOnce();

    await user.click(screen.getByRole("button", { name: "Batal" }));
    expect(onClose).toHaveBeenCalledOnce();
  });

  it("does not render open when closed", () => {
    render(
      <ConfirmDialog
        open={false}
        onClose={() => {}}
        title="Keluar dari akun?"
        description="Kamu perlu masuk lagi untuk melanjutkan."
        confirmLabel="Keluar"
        onConfirm={() => {}}
      />,
    );
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });
});
