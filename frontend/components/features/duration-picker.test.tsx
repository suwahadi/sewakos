import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { describe, expect, it } from "vitest";
import type { RentType } from "@/lib/period";
import { DurationPicker } from "./duration-picker";

function Harness({ rentType, initial }: { rentType: RentType; initial: number }) {
  const [value, setValue] = useState(initial);
  return <DurationPicker rentType={rentType} value={value} onChange={setValue} monthlyOptions={[1, 3, 6]} endDate="2027-03-01" />;
}

describe("DurationPicker", () => {
  it("offers the room's monthly options as single-choice chips", async () => {
    const user = userEvent.setup();
    render(<Harness rentType="monthly" initial={1} />);
    await user.click(screen.getByRole("button", { name: "3 bulan" }));
    expect(screen.getByRole("button", { name: "3 bulan" })).toHaveAttribute("aria-pressed", "true");
    expect(screen.getByRole("button", { name: "1 bulan" })).toHaveAttribute("aria-pressed", "false");
  });

  it("shows the last day of stay, one day before the exclusive end date", () => {
    render(<Harness rentType="monthly" initial={3} />);
    expect(screen.getByText("Sampai Minggu, 28 Februari 2027")).toBeInTheDocument();
  });

  it("steps daily durations within 1-30", async () => {
    const user = userEvent.setup();
    render(<Harness rentType="daily" initial={1} />);
    expect(screen.getByRole("button", { name: "Kurangi" })).toBeDisabled();
    await user.click(screen.getByRole("button", { name: "Tambah" }));
    expect(screen.getByRole("spinbutton", { name: "Jumlah hari" })).toHaveValue(2);

    const input = screen.getByRole("spinbutton", { name: "Jumlah hari" });
    await user.clear(input);
    await user.type(input, "30");
    expect(screen.getByRole("button", { name: "Tambah" })).toBeDisabled();
  });
});
