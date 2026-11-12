import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { OwnerSummary } from "./owner-summary";

describe("OwnerSummary", () => {
  it("shows initials, name, and active month without a link", () => {
    render(<OwnerSummary name="Renny Swarnasari" activeSince="2025-06-14T01:00:00.000000Z" />);

    expect(screen.getByText("RS")).toBeInTheDocument();
    expect(screen.getByText("Renny Swarnasari")).toBeInTheDocument();
    expect(screen.getByText("Pemilik Kos · Aktif sejak Juni 2025")).toBeInTheDocument();
    expect(screen.queryByRole("link")).not.toBeInTheDocument();
  });
});
