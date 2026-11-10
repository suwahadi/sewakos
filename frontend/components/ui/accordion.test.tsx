import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";
import { AccordionItem } from "./accordion";

describe("AccordionItem", () => {
  it("toggles aria-expanded and inert on the panel when clicked", async () => {
    const user = userEvent.setup();
    render(
      <AccordionItem title="Lihat semua fasilitas (10)">
        <p>Isi fasilitas</p>
      </AccordionItem>,
    );

    const trigger = screen.getByRole("button", { name: "Lihat semua fasilitas (10)" });
    expect(trigger).toHaveAttribute("aria-expanded", "false");

    const panel = screen.getByRole("region", { hidden: true });
    expect(panel).toHaveAttribute("inert");

    await user.click(trigger);
    expect(trigger).toHaveAttribute("aria-expanded", "true");
    expect(panel).not.toHaveAttribute("inert");
  });

  it("starts open when defaultOpen is true", () => {
    render(
      <AccordionItem title="Tentang kost" defaultOpen>
        <p>Deskripsi</p>
      </AccordionItem>,
    );
    expect(screen.getByRole("button", { name: "Tentang kost" })).toHaveAttribute("aria-expanded", "true");
  });
});
