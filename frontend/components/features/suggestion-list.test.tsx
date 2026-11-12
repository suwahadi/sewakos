import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { SuggestionList } from "./suggestion-list";

function renderList(onSubmitKeyword = vi.fn()) {
  render(
    <SuggestionList
      onQueryChange={vi.fn()}
      areas={[]}
      landings={[]}
      roomTypes={[]}
      popularAreas={[]}
      onSelectArea={vi.fn()}
      onSelectLanding={vi.fn()}
      onSelectRoomType={vi.fn()}
      onSelectPopularArea={vi.fn()}
      onSubmitKeyword={onSubmitKeyword}
    />,
  );
  return screen.getByRole("combobox");
}

describe("SuggestionList keyword search", () => {
  it("submits typed text on Enter even when nothing matches", async () => {
    const onSubmitKeyword = vi.fn();
    const user = userEvent.setup();
    const input = renderList(onSubmitKeyword);

    await user.type(input, "abcdefg");
    await user.keyboard("{Enter}");

    expect(onSubmitKeyword).toHaveBeenCalledWith("abcdefg");
  });

  it("offers the typed text as the first option and submits it on click", async () => {
    const onSubmitKeyword = vi.fn();
    const user = userEvent.setup();
    renderList(onSubmitKeyword);

    await user.type(screen.getByRole("combobox"), "kost depok");
    await user.click(screen.getByRole("option", { name: "Cari “kost depok”" }));

    expect(onSubmitKeyword).toHaveBeenCalledWith("kost depok");
  });

  it("does not submit a keyword shorter than 3 characters", async () => {
    const onSubmitKeyword = vi.fn();
    const user = userEvent.setup();
    const input = renderList(onSubmitKeyword);

    await user.type(input, "ab");
    await user.keyboard("{Enter}");

    expect(onSubmitKeyword).not.toHaveBeenCalled();
  });
});
