import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { InvoiceCard } from "./invoice-card";

const invoice = {
  id: 7,
  number: "INV/2026/12/000007",
  type: "renewal",
  status: "unpaid",
  period_start: "2026-12-31",
  period_end: "2027-01-31",
  total_label: "Rp2.250.000",
  due_date: "2026-12-31",
};

describe("InvoiceCard", () => {
  it("shows the period ending the day before period_end and a pay action while unpaid", () => {
    render(<InvoiceCard invoice={invoice} paying={false} onPay={vi.fn()} />);
    expect(screen.getByText("Tagihan bulanan")).toBeInTheDocument();
    expect(screen.getByText("31 Des 2026 – 30 Jan 2027")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Bayar" })).toBeInTheDocument();
  });

  it("hides the pay action once paid", () => {
    render(<InvoiceCard invoice={{ ...invoice, status: "paid" }} paying={false} onPay={vi.fn()} />);
    expect(screen.getByText("Lunas")).toBeInTheDocument();
    expect(screen.queryByRole("button")).not.toBeInTheDocument();
  });

  it("offers no pay action when the booking can no longer be paid", () => {
    render(<InvoiceCard invoice={invoice} paying={false} />);
    expect(screen.getByText("Belum dibayar")).toBeInTheDocument();
    expect(screen.queryByRole("button")).not.toBeInTheDocument();
  });
});
