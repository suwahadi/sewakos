import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { describe, expect, it } from "vitest";
import { SegmentedControl } from "./segmented-control";

function Harness() {
  const [value, setValue] = useState("monthly");
  return (
    <SegmentedControl
      aria-label="Tipe sewa"
      value={value}
      onChange={setValue}
      options={[
        { value: "monthly", label: "Bulanan" },
        { value: "weekly", label: "Mingguan" },
        { value: "daily", label: "Harian" },
      ]}
    />
  );
}

describe("SegmentedControl", () => {
  it("exposes the radiogroup/radio ARIA pattern with one checked option", () => {
    render(<Harness />);
    const group = screen.getByRole("radiogroup", { name: "Tipe sewa" });
    expect(group).toBeInTheDocument();

    const options = screen.getAllByRole("radio");
    expect(options).toHaveLength(3);
    expect(screen.getByRole("radio", { name: "Bulanan" })).toHaveAttribute("aria-checked", "true");
    expect(screen.getByRole("radio", { name: "Mingguan" })).toHaveAttribute("aria-checked", "false");
  });

  it("moves selection and focus with the arrow keys", async () => {
    const user = userEvent.setup();
    render(<Harness />);

    const monthly = screen.getByRole("radio", { name: "Bulanan" });
    monthly.focus();
    await user.keyboard("{ArrowRight}");

    const weekly = screen.getByRole("radio", { name: "Mingguan" });
    expect(weekly).toHaveAttribute("aria-checked", "true");
    expect(weekly).toHaveFocus();

    await user.keyboard("{ArrowLeft}");
    expect(monthly).toHaveAttribute("aria-checked", "true");
    expect(monthly).toHaveFocus();

    await user.keyboard("{ArrowLeft}");
    const daily = screen.getByRole("radio", { name: "Harian" });
    expect(daily).toHaveAttribute("aria-checked", "true");
    expect(daily).toHaveFocus();
  });
});
