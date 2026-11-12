import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import type { AdminReview, OwnerReview } from "@/lib/reviews";
import { resetToastsForTests } from "@/lib/toast";

vi.mock("next/navigation", () => ({ useRouter: () => ({ push: vi.fn(), refresh: vi.fn() }), usePathname: () => "/admin/ulasan" }));
const callBffMock = vi.fn();
vi.mock("@/lib/api/browser", () => ({ callBff: (...args: unknown[]) => callBffMock(...args) }));

const { RatingInput } = await import("@/components/ui/rating-input");
const { ReviewCard } = await import("./review-card");
const { OwnerReviewList } = await import("./owner-review-list");
const { AdminReviewList } = await import("./admin-review-list");

function ok(data: object = {}, message = "OK") {
  return { ok: true, httpStatus: 200, data: { status: true, meta: { message }, ...data } };
}

const review: OwnerReview = {
  id: 7, rating: 4, comment: "Bersih dan tenang.", tenant_name: "Budi S.", room_name: "Kamar Deluxe",
  created_at: "2026-10-01T03:00:00.000000Z", property_id: 3, is_hidden: false,
};

afterEach(() => {
  callBffMock.mockReset();
  resetToastsForTests();
});

function ControlledRating() {
  const [value, setValue] = useState(0);
  return <RatingInput value={value} onChange={setValue} />;
}

describe("RatingInput", () => {
  it("selects by click and moves with arrow keys", async () => {
    const user = userEvent.setup();
    render(<ControlledRating />);
    expect(screen.getByText("Pilih 1 sampai 5 bintang")).toBeInTheDocument();

    await user.click(screen.getByRole("radio", { name: "3 bintang" }));
    expect(screen.getByRole("radio", { name: "3 bintang" })).toHaveAttribute("aria-checked", "true");
    expect(screen.getByText("Cukup")).toBeInTheDocument();

    await user.keyboard("{ArrowRight}");
    expect(screen.getByRole("radio", { name: "4 bintang" })).toHaveFocus();
    expect(screen.getByText("Bagus")).toBeInTheDocument();
    await user.keyboard("{ArrowLeft}{ArrowLeft}{ArrowLeft}{ArrowLeft}");
    expect(screen.getByRole("radio", { name: "1 bintang" })).toHaveAttribute("aria-checked", "true");
  });
});

describe("ReviewCard", () => {
  it("shows the masked name, room, star rating and expands long comments", async () => {
    const user = userEvent.setup();
    const { rerender } = render(<ReviewCard review={review} />);
    expect(screen.getByText("Budi S.")).toBeInTheDocument();
    expect(screen.getByText("Kamar Deluxe")).toBeInTheDocument();
    expect(screen.getByRole("img", { name: "Rating 4 dari 5" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Selengkapnya" })).not.toBeInTheDocument();

    rerender(<ReviewCard review={{ ...review, comment: "Sangat nyaman. ".repeat(20) }} />);
    await user.click(screen.getByRole("button", { name: "Selengkapnya" }));
    expect(screen.getByRole("button", { name: "Tutup" })).toHaveAttribute("aria-expanded", "true");
  });
});

describe("OwnerReviewList", () => {
  it("labels reviews with the property name and refetches when filtering", async () => {
    const user = userEvent.setup();
    callBffMock.mockResolvedValue(ok({ reviews: [{ ...review, id: 8, is_hidden: true }] }));
    render(<OwnerReviewList properties={[{ id: 3, name: "Kost Melati" }]} initial={[review]} />);
    expect(screen.getByText("Kost Melati", { selector: "p" })).toBeInTheDocument();

    await user.selectOptions(screen.getByLabelText("Properti"), "3");
    expect(callBffMock).toHaveBeenCalledWith("/api/owner/reviews?page=1&limit=10&property_id=3");
    expect(await screen.findByText("Disembunyikan admin")).toBeInTheDocument();
  });
});

describe("AdminReviewList", () => {
  it("requires a reason before hiding a review", async () => {
    const user = userEvent.setup();
    const adminReview: AdminReview = { ...review, tenant_name: "Budi Santoso", hidden_reason: null };
    callBffMock.mockResolvedValueOnce(ok({}, "Ulasan disembunyikan.")).mockResolvedValue(ok({ reviews: [] }));
    render(<AdminReviewList initial={[adminReview]} />);

    const table = screen.getByRole("table", { name: "Daftar ulasan" });
    await user.click(within(table).getByRole("button", { name: "Sembunyikan" }));
    const dialog = screen.getByRole("dialog");
    await user.click(within(dialog).getByRole("button", { name: "Sembunyikan" }));
    expect(within(dialog).getByText("Alasan wajib diisi.")).toBeInTheDocument();
    expect(callBffMock).not.toHaveBeenCalled();

    await user.type(within(dialog).getByLabelText("Alasan"), "Mengandung kata kasar");
    await user.click(within(dialog).getByRole("button", { name: "Sembunyikan" }));
    expect(callBffMock).toHaveBeenCalledWith("/api/admin/reviews/7/hide", { method: "POST", body: { reason: "Mengandung kata kasar" } });
  });
});
