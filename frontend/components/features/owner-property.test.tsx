import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import type { OwnerProperty, OwnerRoom } from "@/lib/owner";
import { resetToastsForTests } from "@/lib/toast";

const pushMock = vi.fn();
const refreshMock = vi.fn();
vi.mock("next/navigation", () => ({ useRouter: () => ({ push: pushMock, refresh: refreshMock }) }));
const callBffMock = vi.fn();
vi.mock("@/lib/api/browser", () => ({ callBff: (...args: unknown[]) => callBffMock(...args) }));

const { OwnerPropertyForm } = await import("./owner-property-form");
const { OwnerPropertyDetail } = await import("./owner-property-detail");
const { OwnerRoomSheet } = await import("./owner-room-sheet");

const areas = [{ id: 3, name: "Kota Depok" }];

const property: OwnerProperty = {
  id: 12,
  name: "Kost Asti Beji",
  gender: "campur",
  description: "Dekat stasiun",
  address: "Jl. Contoh No. 42",
  latitude: -6.38,
  longitude: 106.81,
  area: { id: 3, name: "Kota Depok" },
  is_active: true,
  moderation_status: "approved",
  moderation_note: null,
  is_flash_sale: false,
  flash_sale_discount: 0,
  rooms_count: 1,
  photos: [],
};

const room: OwnerRoom = {
  id: 5,
  property_id: 12,
  name: "Kamar Tipe A",
  price_monthly: 1500000,
  price_weekly: 400000,
  price_daily: null,
  available: 2,
  held: 1,
  occupied: 0,
  capacity: 1,
  monthly_duration_options: [1, 3],
  facilities: ["WiFi", "AC"],
};

function ok(data: object, message = "OK") {
  return { ok: true, httpStatus: 200, data: { status: true, meta: { message }, ...data } };
}

afterEach(() => {
  callBffMock.mockReset();
  pushMock.mockReset();
  refreshMock.mockReset();
  resetToastsForTests();
});

describe("OwnerPropertyForm", () => {
  it("checks required fields and coordinates before calling the API", async () => {
    const user = userEvent.setup();
    render(<OwnerPropertyForm areas={areas} />);
    await user.type(screen.getByLabelText("Latitude"), "-120");
    await user.click(screen.getByRole("button", { name: "Simpan properti" }));
    expect(screen.getByText("Nama kost wajib diisi.")).toBeInTheDocument();
    expect(screen.getByText("Latitude harus berupa angka antara -90 dan 90.")).toBeInTheDocument();
    expect(screen.getByLabelText("Nama kost")).toHaveFocus();
    expect(callBffMock).not.toHaveBeenCalled();
  });

  it("creates the property and opens its management page", async () => {
    const user = userEvent.setup();
    callBffMock.mockResolvedValue(ok({ property: { id: 44 } }, "Properti dibuat dan menunggu persetujuan admin."));
    render(<OwnerPropertyForm areas={areas} />);
    await user.type(screen.getByLabelText("Nama kost"), "Kost Baru");
    await user.selectOptions(screen.getByLabelText("Kota"), "3");
    await user.click(screen.getByRole("radio", { name: "Putri" }));
    await user.type(screen.getByLabelText("Alamat"), "Jl. Margonda");
    await user.type(screen.getByLabelText("Deskripsi"), "Dekat kampus");
    await user.type(screen.getByLabelText("Latitude"), "-6.36");
    await user.type(screen.getByLabelText("Longitude"), "106.83");
    await user.click(screen.getByRole("button", { name: "Simpan properti" }));

    expect(callBffMock).toHaveBeenCalledWith("/api/owner/properties", {
      method: "POST",
      body: { name: "Kost Baru", area_id: 3, gender: "putri", address: "Jl. Margonda", description: "Dekat kampus", latitude: -6.36, longitude: 106.83 },
    });
    expect(pushMock).toHaveBeenCalledWith("/owner/properti/44");
  });
});

describe("OwnerRoomSheet", () => {
  it("sends cleared optional prices as null and keeps durations sorted", async () => {
    const user = userEvent.setup();
    callBffMock.mockResolvedValue(ok({ room }, "Kamar diperbarui."));
    render(<OwnerRoomSheet open onClose={vi.fn()} propertyId={12} room={room} />);

    expect(screen.getByLabelText("Harga per bulan")).toHaveValue("1.500.000");
    await user.clear(screen.getByLabelText("Per minggu"));
    await user.click(screen.getByRole("button", { name: "6 bulan" }));
    await user.click(screen.getByRole("button", { name: "Simpan kamar" }));

    expect(callBffMock).toHaveBeenCalledWith("/api/owner/rooms/5", {
      method: "PATCH",
      body: expect.objectContaining({ price_monthly: 1500000, price_weekly: null, price_daily: null, facilities: ["WiFi", "AC"], monthly_duration_options: [1, 3, 6] }),
    });
    expect(refreshMock).toHaveBeenCalled();
  });

  it("stops offering durations after six choices", async () => {
    const user = userEvent.setup();
    render(<OwnerRoomSheet open onClose={vi.fn()} propertyId={12} />);
    for (const months of [2, 3, 4, 5, 6]) {
      await user.click(screen.getByRole("button", { name: `${months} bulan` }));
    }
    expect(screen.getByRole("button", { name: "12 bulan" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "6 bulan" })).toBeEnabled();
  });
});

describe("OwnerPropertyDetail", () => {
  it("toggles visibility through PATCH and refreshes", async () => {
    const user = userEvent.setup();
    callBffMock.mockResolvedValue(ok({ property }, "Properti diperbarui."));
    render(<OwnerPropertyDetail property={property} rooms={[room]} areas={areas} />);
    await user.click(screen.getByRole("switch", { name: "Tampilkan di pencarian" }));
    expect(callBffMock).toHaveBeenCalledWith("/api/owner/properties/12", { method: "PATCH", body: { is_active: false } });
    expect(refreshMock).toHaveBeenCalled();
    expect(screen.getByText("2 kosong, 1 dikunci, 0 terisi")).toBeInTheDocument();
  });

  it("shows the rejection note and resubmits", async () => {
    const user = userEvent.setup();
    callBffMock.mockResolvedValue(ok({ property }, "Properti diajukan ulang ke admin."));
    render(<OwnerPropertyDetail property={{ ...property, moderation_status: "rejected", moderation_note: "Foto kurang jelas" }} rooms={[]} areas={areas} />);
    expect(screen.getByText("Foto kurang jelas")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Ajukan ulang" }));
    expect(callBffMock).toHaveBeenCalledWith("/api/owner/properties/12/submit", { method: "POST", body: undefined });
  });
});
