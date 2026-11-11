import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import { getToastsSnapshot, resetToastsForTests } from "@/lib/toast";
import type { DraftPriceItem } from "./sewa-client";

const pushMock = vi.fn();
vi.mock("next/navigation", () => ({ useRouter: () => ({ push: pushMock }) }));

const callBffMock = vi.fn();
vi.mock("@/lib/api/browser", () => ({ callBff: (...args: unknown[]) => callBffMock(...args) }));

const { SewaClient } = await import("./sewa-client");

function price(type: DraftPriceItem["type"], overrides: Partial<DraftPriceItem> = {}): DraftPriceItem {
  return {
    type,
    price: 2250000,
    price_label: "Rp2.250.000",
    deposit_label: "Rp2.250.000",
    admin_fee: 0,
    admin_fee_label: "Rp0",
    flash_sale_discount: 0,
    flash_sale_discount_label: "Rp0",
    total_off_first_payment_label: "Rp4.500.000",
    remaining_payment_label: "Rp4.500.000",
    ...overrides,
  };
}

function meta(code: number, message: string) {
  return { response_code: code, code, severity: code < 300 ? "OK" : "ERROR", message };
}

function renderForm() {
  return render(
    <SewaClient
      roomId={1}
      isFlashSale={false}
      monthlyDurationOptions={[1, 3]}
      initialPrices={[price("monthly"), price("daily", { price_label: "Rp150.000" })]}
    />,
  );
}

describe("SewaClient", () => {
  afterEach(() => {
    callBffMock.mockReset();
    pushMock.mockReset();
    resetToastsForTests();
  });

  it("offers only the rent types the room has prices for", () => {
    renderForm();
    expect(screen.getAllByRole("radio").map((radio) => radio.textContent)).toEqual(["Bulanan", "Harian"]);
    expect(screen.getByText("Total bayar sekarang")).toBeInTheDocument();
  });

  it("switches to a day stepper and multiplies the daily price", async () => {
    const user = userEvent.setup();
    renderForm();
    await user.click(screen.getByRole("radio", { name: "Harian" }));
    expect(screen.getByRole("spinbutton", { name: "Jumlah hari" })).toHaveValue(1);
    expect(screen.getByText("Rp150.000 × 1 hari")).toBeInTheDocument();
  });

  it("submits the booking and opens its detail page", async () => {
    const user = userEvent.setup();
    callBffMock.mockResolvedValue({
      ok: true,
      httpStatus: 200,
      data: { status: true, meta: meta(200, "Pengajuan sewa dibuat. Silakan lakukan pembayaran."), booking: { id: 7 } },
    });
    renderForm();
    await user.click(screen.getByRole("button", { name: "Ajukan sewa" }));

    expect(callBffMock).toHaveBeenCalledWith("/api/booking/submit", {
      method: "POST",
      body: expect.objectContaining({ room_id: 1, rent_type: "monthly", duration: 1 }),
    });
    expect(pushMock).toHaveBeenCalledWith("/booking/7");
    expect(getToastsSnapshot()[0]).toMatchObject({ variant: "success" });
  });

  it("shows a full room as an inline alert on the form", async () => {
    const user = userEvent.setup();
    callBffMock.mockResolvedValue({
      ok: false,
      httpStatus: 400,
      meta: meta(400, "Kamar tidak tersedia."),
      messages: null,
      retryAfterSeconds: null,
    });
    renderForm();
    await user.click(screen.getByRole("button", { name: "Ajukan sewa" }));
    expect(screen.getByRole("alert")).toHaveTextContent("Kamar tidak tersedia.");
    expect(pushMock).not.toHaveBeenCalled();
  });
});
