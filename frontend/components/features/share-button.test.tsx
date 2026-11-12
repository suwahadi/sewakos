import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ShareButton } from "./share-button";

describe("ShareButton", () => {
  afterEach(() => {
    vi.restoreAllMocks();
    Reflect.deleteProperty(navigator, "share");
  });

  it("copies the page link when the Web Share API is unavailable", async () => {
    const user = userEvent.setup();
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(navigator, "clipboard", { value: { writeText }, configurable: true });
    render(<ShareButton title="Kost Asti Beji" />);

    await user.click(screen.getByRole("button", { name: "Bagikan" }));

    expect(writeText).toHaveBeenCalledWith(window.location.href);
  });

  it("opens the native share sheet when available", async () => {
    const share = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(navigator, "share", { value: share, configurable: true });
    const user = userEvent.setup();
    render(<ShareButton title="Kost Asti Beji" />);

    await user.click(screen.getByRole("button", { name: "Bagikan" }));

    expect(share).toHaveBeenCalledWith({ title: "Kost Asti Beji", url: window.location.href });
  });
});
