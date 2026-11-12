import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { OwnerPhoto } from "@/lib/owner";
import { getToastsSnapshot, resetToastsForTests } from "@/lib/toast";

vi.mock("next/navigation", () => ({ useRouter: () => ({ push: vi.fn(), refresh: vi.fn() }) }));
const callBffMock = vi.fn();
vi.mock("@/lib/api/browser", () => ({ callBff: (...args: unknown[]) => callBffMock(...args) }));

const { PhotoUploader } = await import("./photo-uploader");

class FakeXHR {
  static sent: { url: string; headers: Record<string, string> }[] = [];
  upload: { onprogress: ((event: { lengthComputable: boolean; loaded: number; total: number }) => void) | null } = { onprogress: null };
  onload: (() => void) | null = null;
  onerror: (() => void) | null = null;
  status = 200;
  private url = "";
  private headers: Record<string, string> = {};
  open(_method: string, url: string) {
    this.url = url;
  }
  setRequestHeader(name: string, value: string) {
    this.headers[name] = value;
  }
  send() {
    FakeXHR.sent.push({ url: this.url, headers: this.headers });
    this.upload.onprogress?.({ lengthComputable: true, loaded: 5, total: 10 });
    queueMicrotask(() => this.onload?.());
  }
}

function photo(id: number, sortOrder: number): OwnerPhoto {
  return { id, url: `https://cdn.example.com/${id}.webp`, thumbnail_url: `https://cdn.example.com/${id}_thumb.webp`, caption: null, room_id: null, sort_order: sortOrder };
}

function ok(data: object, message = "OK") {
  return { ok: true, httpStatus: 200, data: { status: true, meta: { message }, ...data } };
}

beforeEach(() => {
  vi.stubGlobal("XMLHttpRequest", FakeXHR);
  URL.createObjectURL = vi.fn(() => "blob:preview");
  URL.revokeObjectURL = vi.fn();
  FakeXHR.sent = [];
});

afterEach(() => {
  vi.unstubAllGlobals();
  callBffMock.mockReset();
  resetToastsForTests();
});

describe("PhotoUploader", () => {
  it("rejects unsupported and oversized files before uploading", async () => {
    const user = userEvent.setup({ applyAccept: false });
    render(<PhotoUploader propertyId={12} initialPhotos={[]} />);
    const gif = new File(["x"], "anim.gif", { type: "image/gif" });
    const huge = new File([new Uint8Array(10 * 1024 * 1024 + 1)], "besar.jpg", { type: "image/jpeg" });
    await user.upload(screen.getByLabelText("Pilih foto"), [gif, huge]);

    expect(getToastsSnapshot().map((item) => item.title)).toEqual([
      "anim.gif: format harus JPEG, PNG, atau WebP.",
      "besar.jpg: ukuran maksimal 10 MB.",
    ]);
    expect(callBffMock).not.toHaveBeenCalled();
  });

  it("presigns, uploads with the signed headers, then registers the photo", async () => {
    const user = userEvent.setup();
    callBffMock.mockImplementation(async (path: string) =>
      path === "/api/owner/uploads/presign"
        ? ok({ upload: { url: "http://storage.test/up", headers: { "Content-Type": "image/jpeg" }, object_key: "uploads/3/a.jpg" } })
        : ok({ photo: photo(41, 1) }, "Foto ditambahkan."),
    );
    render(<PhotoUploader propertyId={12} initialPhotos={[]} />);
    await user.upload(screen.getByLabelText("Pilih foto"), new File(["jpeg"], "depan.jpg", { type: "image/jpeg" }));

    expect(await screen.findByRole("button", { name: "Kelola foto 1" })).toBeInTheDocument();
    expect(callBffMock).toHaveBeenCalledWith("/api/owner/uploads/presign", { method: "POST", body: { content_type: "image/jpeg", size: 4 } });
    expect(FakeXHR.sent).toEqual([{ url: "http://storage.test/up", headers: { "Content-Type": "image/jpeg" } }]);
    expect(callBffMock).toHaveBeenCalledWith("/api/owner/properties/12/photos", { method: "POST", body: { object_key: "uploads/3/a.jpg" } });
    expect(screen.getByText("1/30")).toBeInTheDocument();
  });

  it("shows a retryable failure when processing is rejected", async () => {
    const user = userEvent.setup();
    callBffMock.mockImplementation(async (path: string) =>
      path === "/api/owner/uploads/presign"
        ? ok({ upload: { url: "http://storage.test/up", headers: {}, object_key: "uploads/3/a.jpg" } })
        : { ok: false, httpStatus: 400, meta: { message: "File foto tidak dapat diproses." }, messages: null, retryAfterSeconds: null },
    );
    render(<PhotoUploader propertyId={12} initialPhotos={[]} />);
    await user.upload(screen.getByLabelText("Pilih foto"), new File(["x"], "rusak.jpg", { type: "image/jpeg" }));

    expect(await screen.findByRole("alert")).toHaveTextContent("File foto tidak dapat diproses.");
    expect(screen.getByRole("button", { name: "Ulangi" })).toBeInTheDocument();
  });

  it("moves a photo right with Alt+ArrowRight and renumbers sort_order", async () => {
    const user = userEvent.setup();
    callBffMock.mockResolvedValue(ok({}, "Foto diperbarui."));
    render(<PhotoUploader propertyId={12} initialPhotos={[photo(1, 0), photo(2, 0)]} />);

    screen.getByRole("button", { name: "Geser foto 1" }).focus();
    await user.keyboard("{Alt>}{ArrowRight}{/Alt}");

    await waitFor(() => expect(callBffMock).toHaveBeenCalledTimes(2));
    expect(callBffMock).toHaveBeenNthCalledWith(1, "/api/owner/photos/2", { method: "PATCH", body: { sort_order: 1 } });
    expect(callBffMock).toHaveBeenNthCalledWith(2, "/api/owner/photos/1", { method: "PATCH", body: { sort_order: 2 } });
  });
});
