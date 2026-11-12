import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { FacilitySelect } from "./facility-select";

describe("FacilitySelect", () => {
  it("lists selected facilities as removable chips", async () => {
    const onChange = vi.fn();
    const user = userEvent.setup();
    render(<FacilitySelect value={["WiFi", "Kasur"]} onChange={onChange} />);

    const chips = screen.getByRole("list", { name: "Fasilitas terpilih" });
    expect(within(chips).getAllByRole("listitem")).toHaveLength(2);
    await user.click(screen.getByRole("button", { name: "Hapus WiFi" }));
    expect(onChange).toHaveBeenCalledWith(["Kasur"]);
  });

  it("adds a facility from the catalog through the picker sheet", async () => {
    const onChange = vi.fn();
    const user = userEvent.setup();
    render(<FacilitySelect value={["WiFi"]} onChange={onChange} />);

    await user.click(screen.getByRole("button", { name: "Pilih fasilitas" }));
    await user.click(screen.getByRole("checkbox", { name: "AC" }));
    expect(onChange).toHaveBeenCalledWith(["WiFi", "AC"]);
  });

  it("keeps a legacy facility that is not in the catalog visible and checked", async () => {
    const onChange = vi.fn();
    const user = userEvent.setup();
    render(<FacilitySelect value={["Gym lama"]} onChange={onChange} />);

    await user.click(screen.getByRole("button", { name: "Pilih fasilitas" }));
    const legacy = screen.getByRole("checkbox", { name: "Gym lama" });
    expect(legacy).toBeChecked();
    await user.click(legacy);
    expect(onChange).toHaveBeenCalledWith([]);
  });
});
