import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { describe, expect, it } from "vitest";
import { TabPanel, Tabs } from "./tabs";

function Harness() {
  const [value, setValue] = useState("all");
  return (
    <>
      <Tabs
        idPrefix="booking"
        aria-label="Status booking"
        value={value}
        onChange={setValue}
        items={[
          { value: "all", label: "Semua" },
          { value: "paid", label: "Dibayar" },
          { value: "approved", label: "Aktif" },
        ]}
      />
      <TabPanel idPrefix="booking" value="all" activeValue={value}>
        Semua booking
      </TabPanel>
      <TabPanel idPrefix="booking" value="paid" activeValue={value}>
        Booking dibayar
      </TabPanel>
      <TabPanel idPrefix="booking" value="approved" activeValue={value}>
        Booking aktif
      </TabPanel>
    </>
  );
}

describe("Tabs", () => {
  it("exposes the tablist/tab/tabpanel ARIA pattern", () => {
    render(<Harness />);
    expect(screen.getByRole("tablist", { name: "Status booking" })).toBeInTheDocument();
    const tabs = screen.getAllByRole("tab");
    expect(tabs).toHaveLength(3);
    expect(screen.getByRole("tab", { name: "Semua" })).toHaveAttribute("aria-selected", "true");
    expect(screen.getByRole("tabpanel")).toHaveTextContent("Semua booking");
  });

  it("switches tabs and panel with the arrow keys", async () => {
    const user = userEvent.setup();
    render(<Harness />);

    screen.getByRole("tab", { name: "Semua" }).focus();
    await user.keyboard("{ArrowRight}");

    const paid = screen.getByRole("tab", { name: "Dibayar" });
    expect(paid).toHaveAttribute("aria-selected", "true");
    expect(paid).toHaveFocus();
    expect(screen.getByRole("tabpanel")).toHaveTextContent("Booking dibayar");
  });
});
