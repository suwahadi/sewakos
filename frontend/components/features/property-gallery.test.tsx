import { fireEvent, render, screen, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { PropertyGallery } from "./property-gallery";

const photos = [
  { url: "https://cdn.example.com/1.webp", thumbnail_url: "https://cdn.example.com/1_t.webp", caption: null },
  { url: "https://cdn.example.com/2.webp", thumbnail_url: "https://cdn.example.com/2_t.webp", caption: null },
  { url: "https://cdn.example.com/3.webp", thumbnail_url: "https://cdn.example.com/3_t.webp", caption: null },
];

describe("PropertyGallery", () => {
  it("opens the lightbox on the photo that was clicked", () => {
    render(<PropertyGallery photos={photos} propertyName="Kost Melati" />);
    const lightbox = screen.getByLabelText("Foto Kost Melati");
    expect(within(lightbox).queryByRole("img")).not.toBeInTheDocument();

    fireEvent.click(screen.getAllByRole("button", { name: "Perbesar foto 2" })[0]!);
    expect(within(lightbox).getByRole("img", { name: "Foto Kost Melati 2" })).toBeInTheDocument();
  });

  it("shows no extra-photo count when every photo already fits", () => {
    render(<PropertyGallery photos={photos} propertyName="Kost Melati" />);
    expect(screen.queryByText(/foto$/)).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Lihat semua foto" })).not.toBeInTheDocument();
  });

  it("counts the photos beyond the three on the last tile and opens the lightbox there", () => {
    const five = [
      ...photos,
      { url: "https://cdn.example.com/4.webp", thumbnail_url: "https://cdn.example.com/4_t.webp", caption: null },
      { url: "https://cdn.example.com/5.webp", thumbnail_url: "https://cdn.example.com/5_t.webp", caption: null },
    ];
    render(<PropertyGallery photos={five} propertyName="Kost Melati" />);
    expect(screen.getByText("+2 foto")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Lihat 2 foto lainnya" }));
    expect(within(screen.getByLabelText("Foto Kost Melati")).getByRole("img", { name: "Foto Kost Melati 3" })).toBeInTheDocument();
  });
});
