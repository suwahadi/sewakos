import { render, screen, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { DataTable } from "./data-table";

const rows = [
  { id: 1, name: "Kost Asti", total: "Rp1.000" },
  { id: 2, name: "Kost Melati", total: "Rp2.000" },
];

describe("DataTable", () => {
  it("renders a real table and a stacked list for small screens", () => {
    render(
      <DataTable
        caption="Daftar kost"
        rows={rows}
        rowKey={(row) => row.id}
        columns={[
          { key: "name", header: "Nama", render: (row) => row.name },
          { key: "total", header: "Total", render: (row) => row.total, align: "right" },
        ]}
      />,
    );
    const table = screen.getByRole("table", { name: "Daftar kost" });
    expect(within(table).getAllByRole("columnheader").map((cell) => cell.textContent)).toEqual(["Nama", "Total"]);
    expect(within(table).getAllByRole("row")).toHaveLength(3);
    expect(within(screen.getByRole("list", { name: "Daftar kost" })).getAllByRole("listitem")).toHaveLength(2);
  });
});
