import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import { resetToastsForTests } from "@/lib/toast";

const pushMock = vi.fn();
vi.mock("next/navigation", () => ({ useRouter: () => ({ push: pushMock, refresh: vi.fn() }), usePathname: () => "/kost/5" }));
vi.mock("next/image", () => ({ default: () => null }));
const callBffMock = vi.fn();
vi.mock("@/lib/api/browser", () => ({ callBff: (...args: unknown[]) => callBffMock(...args) }));

const { FavoriteButton } = await import("./favorite-button");
const { FavoriteList } = await import("./favorite-list");

const ok = { ok: true, httpStatus: 200, data: { status: true, meta: { message: "OK" } } };
const failure = { ok: false, httpStatus: 500, meta: { message: "Terjadi kesalahan." }, messages: null };

afterEach(() => {
  callBffMock.mockReset();
  pushMock.mockReset();
  resetToastsForTests();
});

describe("FavoriteButton", () => {
  it("toggles optimistically and reverts when the request fails", async () => {
    const user = userEvent.setup();
    callBffMock.mockResolvedValueOnce(ok).mockResolvedValueOnce(failure);
    render(<FavoriteButton propertyId={5} initial={false} guest={false} />);

    await user.click(screen.getByRole("button", { name: "Simpan ke favorit" }));
    expect(callBffMock).toHaveBeenCalledWith("/api/favorites/5", { method: "PUT" });
    const active = screen.getByRole("button", { name: "Hapus dari favorit" });
    expect(active).toHaveAttribute("aria-pressed", "true");

    await user.click(active);
    expect(callBffMock).toHaveBeenLastCalledWith("/api/favorites/5", { method: "DELETE" });
    expect(await screen.findByRole("button", { name: "Hapus dari favorit" })).toHaveAttribute("aria-pressed", "true");
  });

  it("sends guests to the login page instead of calling the API", async () => {
    const user = userEvent.setup();
    render(<FavoriteButton propertyId={5} initial={false} guest />);
    await user.click(screen.getByRole("button", { name: "Simpan ke favorit" }));
    expect(pushMock).toHaveBeenCalledWith("/masuk?next=%2Fkost%2F5");
    expect(callBffMock).not.toHaveBeenCalled();
  });
});

describe("FavoriteList", () => {
  const favorite = {
    id: 5, name: "Kost Melati", gender: "putri", area: { name: "Kota Depok" }, thumbnail_url: null,
    min_price_monthly_label: "Rp1.500.000", rating_average: 4.5, is_available: true,
  };

  it("removes an item after unfavoriting and links only available listings", async () => {
    const user = userEvent.setup();
    callBffMock.mockResolvedValue(ok);
    render(<FavoriteList initial={[favorite, { ...favorite, id: 6, name: "Kost Mawar", is_available: false }]} />);

    expect(screen.getByRole("link", { name: /Kost Melati/ })).toHaveAttribute("href", "/kost/5");
    expect(screen.queryByRole("link", { name: /Kost Mawar/ })).not.toBeInTheDocument();
    expect(screen.getByText("Tidak tersedia")).toBeInTheDocument();

    await user.click(screen.getAllByRole("button", { name: "Hapus dari favorit" })[0]!);
    expect(screen.queryByText("Kost Melati")).not.toBeInTheDocument();
  });

  it("shows the empty state with a search action", async () => {
    const user = userEvent.setup();
    render(<FavoriteList initial={[]} />);
    expect(screen.getByText("Belum ada kost favorit")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Cari kost" }));
    expect(pushMock).toHaveBeenCalledWith("/cari");
  });
});
