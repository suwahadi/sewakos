import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { ForbiddenState, NotFoundState } from "./error-state";

describe("NotFoundState", () => {
  it("shows the 404 message and navigates home on click (DESIGN.md §6.19)", async () => {
    const user = userEvent.setup();
    const onGoHome = vi.fn();
    render(<NotFoundState onGoHome={onGoHome} />);

    expect(screen.getByText("Halaman tidak ditemukan")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Kembali ke beranda" }));
    expect(onGoHome).toHaveBeenCalledOnce();
  });
});

describe("ForbiddenState", () => {
  it("shows the wrong-role message", () => {
    render(<ForbiddenState />);
    expect(screen.getByText("Halaman ini tidak tersedia untuk akunmu.")).toBeInTheDocument();
  });
});
