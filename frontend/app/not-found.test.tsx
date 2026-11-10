import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

const pushMock = vi.fn();
vi.mock("next/navigation", () => ({
  useRouter: () => ({ push: pushMock }),
}));

const { default: NotFound } = await import("./not-found");

describe("NotFound page", () => {
  it("renders the 404 ErrorState and goes home on click", async () => {
    const user = userEvent.setup();
    render(<NotFound />);

    expect(screen.getByText("Halaman tidak ditemukan")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Kembali ke beranda" }));
    expect(pushMock).toHaveBeenCalledWith("/");
  });
});
