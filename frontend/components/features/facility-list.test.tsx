import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";
import { FacilityList } from "./facility-list";

describe("FacilityList", () => {
  it("renders nothing for an empty facility list", () => {
    const { container } = render(<FacilityList facilities={[]} />);
    expect(container).toBeEmptyDOMElement();
  });

  it("shows the first 6 facilities directly and the rest behind a collapsed accordion", async () => {
    const user = userEvent.setup();
    const facilities = ["WiFi", "AC", "Kamar mandi dalam", "Kasur", "Akses 24 jam", "Lemari", "Meja belajar"];
    render(<FacilityList facilities={facilities} />);

    for (const facility of facilities.slice(0, 6)) {
      expect(screen.getByText(facility)).toBeInTheDocument();
    }
    const trigger = screen.getByRole("button", { name: "Lihat semua fasilitas (7)" });
    expect(trigger).toHaveAttribute("aria-expanded", "false");

    await user.click(trigger);
    expect(trigger).toHaveAttribute("aria-expanded", "true");
    expect(screen.getByText("Meja belajar")).toBeInTheDocument();
  });

  it("omits the accordion when 6 or fewer facilities are given", () => {
    render(<FacilityList facilities={["WiFi", "AC"]} />);
    expect(screen.queryByRole("button")).not.toBeInTheDocument();
  });
});
