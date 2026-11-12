import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

const pushMock = vi.fn();
vi.mock("next/navigation", () => ({ useRouter: () => ({ push: pushMock }) }));
vi.mock("@/lib/api/browser", () => ({ callBff: vi.fn().mockResolvedValue({ ok: true }) }));

const { KostDetailClient } = await import("./kost-detail-client");

const room = { id: 7, name: "Kamar A", price_label: "Rp2.250.000", available: 2, facilities: [] };

describe("KostDetailClient ajukan sewa", () => {
  it("asks for confirmation before leaving the page", async () => {
    pushMock.mockReset();
    const user = userEvent.setup();
    render(<KostDetailClient propertyId={3} description="" rooms={[room]} />);

    await user.click(screen.getByRole("button", { name: "Ajukan sewa" }));
    expect(screen.getByText("Yakin melanjutkan ajukan sewa properti ini?")).toBeInTheDocument();
    expect(pushMock).not.toHaveBeenCalled();

    await user.click(screen.getByRole("button", { name: "Lanjutkan" }));
    expect(pushMock).toHaveBeenCalledWith("/kost/3/sewa?room=7");
  });

  it("does not leave the page when the user cancels", async () => {
    pushMock.mockReset();
    const user = userEvent.setup();
    render(<KostDetailClient propertyId={3} description="" rooms={[room]} />);

    await user.click(screen.getByRole("button", { name: "Ajukan sewa" }));
    await user.click(screen.getByRole("button", { name: "Batal" }));
    expect(pushMock).not.toHaveBeenCalled();
  });
});
