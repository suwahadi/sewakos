import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ExpandableText } from "./expandable-text";

const TEXT = "Kost bersih dan nyaman.\nDekat stasiun.";

function stubOverflow(scrollHeight: number, clientHeight: number) {
  vi.spyOn(HTMLElement.prototype, "scrollHeight", "get").mockReturnValue(scrollHeight);
  vi.spyOn(HTMLElement.prototype, "clientHeight", "get").mockReturnValue(clientHeight);
}

afterEach(() => vi.restoreAllMocks());

describe("ExpandableText", () => {
  it("shows the whole text without a toggle when it fits", () => {
    stubOverflow(40, 40);
    render(<ExpandableText text={TEXT} />);

    expect(screen.getByText(/Dekat stasiun/)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Selengkapnya" })).not.toBeInTheDocument();
  });

  it("offers Selengkapnya and toggles the clamp when the text overflows", async () => {
    stubOverflow(90, 60);
    const user = userEvent.setup();
    render(<ExpandableText text={TEXT} />);

    const toggle = screen.getByRole("button", { name: "Selengkapnya" });
    expect(toggle).toHaveAttribute("aria-expanded", "false");
    expect(screen.getByText(/Dekat stasiun/)).toHaveClass("line-clamp-3");

    await user.click(toggle);
    expect(screen.getByRole("button", { name: "Tutup" })).toHaveAttribute("aria-expanded", "true");
    expect(screen.getByText(/Dekat stasiun/)).not.toHaveClass("line-clamp-3");
  });
});
