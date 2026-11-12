import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { Button } from "./button";

describe("Button", () => {
  it("defaults to the success variant with a border", () => {
    render(<Button>Simpan</Button>);
    const button = screen.getByRole("button", { name: "Simpan" });
    expect(button.className).toContain("bg-success");
    expect(button.className).toContain("border");
    expect(button.className).toContain("rounded-md");
  });

  it("uses danger for destructive actions", () => {
    render(<Button variant="danger">Hapus</Button>);
    expect(screen.getByRole("button", { name: "Hapus" }).className).toContain("bg-danger");
  });

  it("uses a grey neutral surface for secondary actions", () => {
    render(<Button variant="neutral">Batal</Button>);
    const button = screen.getByRole("button", { name: "Batal" });
    expect(button.className).toContain("bg-neutral-0");
    expect(button.className).toContain("border-neutral-400");
  });
});
