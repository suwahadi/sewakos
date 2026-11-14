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

  it("opens the lightbox from Lihat semua foto at the first photo", () => {
    render(<PropertyGallery photos={photos} propertyName="Kost Melati" />);
    fireEvent.click(screen.getByRole("button", { name: "Lihat semua foto" }));
    expect(within(screen.getByLabelText("Foto Kost Melati")).getByRole("img", { name: "Foto Kost Melati 1" })).toBeInTheDocument();
  });
});
