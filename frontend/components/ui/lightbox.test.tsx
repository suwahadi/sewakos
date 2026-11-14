import { fireEvent, render, screen } from "@testing-library/react";
import { useState } from "react";
import { describe, expect, it, vi } from "vitest";
import { Lightbox } from "./lightbox";

const photos = [
  { src: "https://cdn.example.com/a.webp", alt: "Foto A" },
  { src: "https://cdn.example.com/b.webp", alt: "Foto B" },
];

function Harness({ start, onClose = () => {} }: { start: number | null; onClose?: () => void }) {
  const [index, setIndex] = useState<number | null>(start);
  return <Lightbox photos={photos} index={index} onIndexChange={setIndex} onClose={onClose} label="Foto Kost" />;
}

describe("Lightbox", () => {
  it("shows only the photo, with no title or caption text", () => {
    const { container } = render(<Harness start={0} />);
    expect(screen.getByRole("img", { name: "Foto A" })).toBeInTheDocument();
    expect(container.querySelector("dialog")?.textContent).toBe("");
  });

  it("moves between photos with the buttons and arrow keys, wrapping around", () => {
    render(<Harness start={0} />);
    fireEvent.click(screen.getByRole("button", { name: "Foto berikutnya" }));
    expect(screen.getByRole("img", { name: "Foto B" })).toBeInTheDocument();
    fireEvent.keyDown(screen.getByLabelText("Foto Kost"), { key: "ArrowRight" });
    expect(screen.getByRole("img", { name: "Foto A" })).toBeInTheDocument();
    fireEvent.keyDown(screen.getByLabelText("Foto Kost"), { key: "ArrowLeft" });
    expect(screen.getByRole("img", { name: "Foto B" })).toBeInTheDocument();
  });

  it("closes from the close button and from Escape", () => {
    const onClose = vi.fn();
    render(<Harness start={1} onClose={onClose} />);
    fireEvent.click(screen.getByRole("button", { name: "Tutup" }));
    fireEvent(screen.getByLabelText("Foto Kost"), new Event("cancel", { cancelable: true }));
    expect(onClose).toHaveBeenCalledTimes(2);
  });

  it("renders no photo while closed", () => {
    render(<Harness start={null} />);
    expect(screen.queryByRole("img")).not.toBeInTheDocument();
  });
});
