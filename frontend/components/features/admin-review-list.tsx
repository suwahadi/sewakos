"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import { DataTable } from "@/components/ui/data-table";
import { ConfirmDialog } from "@/components/ui/dialog";
import { LoadMore } from "@/components/ui/load-more";
import { RatingStarsRow } from "@/components/ui/rating";
import { Tabs, type TabItem } from "@/components/ui/tabs";
import { Textarea } from "@/components/ui/textarea";
import { callBff } from "@/lib/api/browser";
import { showApiError } from "@/lib/api/show-error";
import { formatDateShort } from "@/lib/format";
import type { AdminReview } from "@/lib/reviews";
import { toast } from "@/lib/toast";
import { useAdminList } from "@/lib/use-admin-list";

type HiddenTab = "false" | "true";

const TABS: TabItem<HiddenTab>[] = [
  { value: "false", label: "Tampil" },
  { value: "true", label: "Disembunyikan" },
];

export function AdminReviewList({ initial }: { initial: AdminReview[] }) {
  const router = useRouter();
  const list = useAdminList<AdminReview>("reviews", "reviews", initial, { is_hidden: "false" });
  const hiddenTab = (list.query.is_hidden ?? "false") as HiddenTab;
  const [hiding, setHiding] = useState<AdminReview | null>(null);
  const [reason, setReason] = useState("");
  const [reasonError, setReasonError] = useState<string>();
  const [busyId, setBusyId] = useState<number | null>(null);

  async function moderate(review: AdminReview, action: "hide" | "unhide") {
    if (action === "hide" && !reason.trim()) {
      setReasonError("Alasan wajib diisi.");
      return;
    }
    setBusyId(review.id);
    const result = await callBff<unknown>(`/api/admin/reviews/${review.id}/${action}`, {
      method: "POST",
      body: action === "hide" ? { reason: reason.trim() } : {},
    });
    setBusyId(null);
    if (!result.ok) {
      if (result.httpStatus === 422) {
        setReasonError(result.messages?.reason?.[0] ?? result.meta.message);
        return;
      }
      showApiError(result, (href) => router.push(href), () => moderate(review, action));
      return;
    }
    setHiding(null);
    toast.success(result.data.meta.message);
    list.reload();
  }

  return (
    <div className="flex flex-col gap-4">
      <h1 className="text-title-lg font-semibold text-neutral-900">Moderasi ulasan</h1>
      <div className="border-b border-neutral-200">
        <Tabs items={TABS} value={hiddenTab} onChange={(next) => list.filter({ is_hidden: next })} aria-label="Status ulasan" idPrefix="admin-review" />
      </div>
      {list.rows.length === 0 && !list.loading ? (
        <p className="text-body text-neutral-600">Tidak ada ulasan di sini.</p>
      ) : (
        <DataTable
          caption="Daftar ulasan"
          rows={list.rows}
          rowKey={(review) => review.id}
          columns={[
            {
              key: "property",
              header: "Properti",
              render: (review) => (
                <Link href={`/admin/properti/${review.property_id}`} className="text-brand-700 underline">
                  #{review.property_id}
                </Link>
              ),
            },
            { key: "tenant", header: "Penyewa", render: (review) => review.tenant_name },
            { key: "rating", header: "Rating", render: (review) => <RatingStarsRow rating={review.rating} /> },
            {
              key: "comment",
              header: "Ulasan",
              render: (review) => (
                <div className="flex max-w-md flex-col gap-1">
                  <span className="whitespace-pre-line break-words">{review.comment ?? "–"}</span>
                  {review.hidden_reason && <span className="text-caption text-neutral-600">Alasan: {review.hidden_reason}</span>}
                </div>
              ),
            },
            { key: "date", header: "Tanggal", render: (review) => <span className="whitespace-nowrap">{formatDateShort(review.created_at)}</span> },
            {
              key: "action",
              header: "Aksi",
              render: (review) =>
                review.is_hidden ? (
                  <Button size="sm" loading={busyId === review.id} onClick={() => moderate(review, "unhide")}>
                    Tampilkan
                  </Button>
                ) : (
                  <Button
                    size="sm"
                    variant="danger"
                    onClick={() => {
                      setReason("");
                      setReasonError(undefined);
                      setHiding(review);
                    }}
                  >
                    Sembunyikan
                  </Button>
                ),
            },
          ]}
        />
      )}
      {list.hasMore && <LoadMore loading={list.loading} onClick={list.loadMore} />}
      {hiding && (
        <ConfirmDialog
          open
          onClose={() => setHiding(null)}
          title="Sembunyikan ulasan ini?"
          description={
            <div className="flex flex-col gap-3">
              <p>Ulasan tidak tampil di halaman kost dan tidak dihitung dalam rating.</p>
              <Textarea label="Alasan" maxLength={500} value={reason} onChange={(event) => setReason(event.target.value)} error={reasonError} />
            </div>
          }
          confirmLabel="Sembunyikan"
          loading={busyId === hiding.id}
          onConfirm={() => moderate(hiding, "hide")}
        />
      )}
    </div>
  );
}
