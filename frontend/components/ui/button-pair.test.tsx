import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { Button } from "./button";
import { ButtonPair } from "./button-pair";

describe("ButtonPair", () => {
  it("renders exactly two buttons in a 50:50 grid", () => {
    render(
      <ButtonPair>
        <Button variant="outline">Batal</Button>
        <Button variant="primary">Lanjutkan</Button>
      </ButtonPair>,
    );

    const batal = screen.getByRole("button", { name: "Batal" });
    const lanjutkan = screen.getByRole("button", { name: "Lanjutkan" });
    expect(batal).toBeInTheDocument();
    expect(lanjutkan).toBeInTheDocument();

    const wrapper = batal.parentElement;
    expect(wrapper).toBe(lanjutkan.parentElement);
    expect(wrapper?.className).toContain("grid-cols-2");
  });

  it("puts the secondary action on the left and the primary on the right", () => {
    render(
      <ButtonPair>
        <Button variant="outline">Batal</Button>
        <Button variant="primary">Lanjutkan</Button>
      </ButtonPair>,
    );
    const buttons = screen.getAllByRole("button");
    expect(buttons[0]).toHaveTextContent("Batal");
    expect(buttons[1]).toHaveTextContent("Lanjutkan");
  });
});
