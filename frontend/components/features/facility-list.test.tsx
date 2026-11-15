import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";
import { FacilityList } from "./facility-list";

describe("FacilityList", () => {
  it("renders nothing for an empty facility list", () => {
    const { container } = render(<FacilityList facilities={[]} />);
    expect(container).toBeEmptyDOMElement();
  });

  it("shows the first 6 facilities and opens all of them in a popup from a plain text link", async () => {
    const user = userEvent.setup();
    const facilities = ["WiFi", "AC", "Kamar mandi dalam", "Kasur", "Akses 24 jam", "Lemari", "Meja belajar"];
    render(<FacilityList facilities={facilities} />);

    const outsidePopup = (text: string) => screen.queryAllByText(text).filter((element) => element.closest("dialog") === null);
    for (const facility of facilities.slice(0, 6)) {
      expect(outsidePopup(facility)).toHaveLength(1);
    }
    expect(outsidePopup("Meja belajar")).toHaveLength(0);
    const link = screen.getByRole("button", { name: "Lihat semua fasilitas" });
    expect(link).not.toHaveTextContent(/\(\d+\)/);

    await user.click(link);
    const sheet = within(screen.getByRole("dialog", { hidden: true }));
    expect(sheet.getByText("Fasilitas")).toBeInTheDocument();
    for (const facility of facilities) {
      expect(sheet.getByText(facility)).toBeInTheDocument();
    }
  });

  it("omits the link when 6 or fewer facilities are given", () => {
    render(<FacilityList facilities={["WiFi", "AC"]} />);
    expect(screen.queryByRole("button", { name: "Lihat semua fasilitas" })).not.toBeInTheDocument();
  });
});
