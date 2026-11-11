"use client";

import { BankAccountForm } from "@/components/features/bank-account-form";
import { InlineAlert } from "@/components/ui/inline-alert";
import type { Refund } from "@/lib/booking";
import { formatDateShort } from "@/lib/format";

export function RefundPanel({ refund, bookingId, onSaved }: { refund: Refund; bookingId: number; onSaved: () => void }) {
  return (
    <section aria-label="Pengembalian dana" className="flex flex-col gap-4 rounded-lg border border-neutral-200 p-4">
      <p className="text-title-sm font-semibold text-neutral-900">Pengembalian dana</p>
      {refund.status === "awaiting_account" && (
        <>
          <InlineAlert variant="warning">Isi rekening untuk menerima pengembalian dana {refund.amount_label}.</InlineAlert>
          <BankAccountForm endpoint={`/api/booking/${bookingId}/refund-account`} idPrefix="refund" onSaved={onSaved} onRejected={onSaved} />
        </>
      )}
      {refund.status === "ready" && (
        <InlineAlert variant="info">Pengembalian dana {refund.amount_label} sedang diproses admin.</InlineAlert>
      )}
      {refund.status === "completed" && (
        <InlineAlert variant="success">
          Dana {refund.amount_label} sudah dikembalikan
          {refund.completed_at ? ` pada ${formatDateShort(refund.completed_at)}` : ""}.
        </InlineAlert>
      )}
    </section>
  );
}
