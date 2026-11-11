"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";
import { BankAccountForm } from "@/components/features/bank-account-form";
import { Badge } from "@/components/ui/badge";
import { InlineAlert } from "@/components/ui/inline-alert";
import { LoadMore } from "@/components/ui/load-more";
import { callBff } from "@/lib/api/browser";
import { showApiError } from "@/lib/api/show-error";
import { formatDateShort } from "@/lib/format";
import { payoutStatusLabel } from "@/lib/labels";
import { OWNER_PAGE_SIZE, type BankAccount, type Payout } from "@/lib/owner";

export function OwnerFinance({ bankAccount, initialPayouts }: { bankAccount: BankAccount | null; initialPayouts: Payout[] }) {
  const router = useRouter();
  const [payouts, setPayouts] = useState(initialPayouts);
  const [page, setPage] = useState(1);
  const [hasMore, setHasMore] = useState(initialPayouts.length >= OWNER_PAGE_SIZE);
  const [loading, setLoading] = useState(false);

  async function loadMore() {
    setLoading(true);
    const nextPage = page + 1;
    const result = await callBff<{ payouts: Payout[] }>(`/api/owner/payouts?page=${nextPage}&limit=${OWNER_PAGE_SIZE}`);
    setLoading(false);
    if (!result.ok) return showApiError(result, (href) => router.push(href), loadMore);
    setPayouts((current) => [...current, ...result.data.payouts]);
    setPage(nextPage);
    setHasMore(result.data.payouts.length >= OWNER_PAGE_SIZE);
  }

  return (
    <main className="mx-auto flex max-w-[960px] flex-col gap-8 px-4 py-4">
      <section className="flex flex-col gap-4">
        <h2 className="text-title-md font-semibold text-neutral-900">Rekening penerima</h2>
        {!bankAccount && <InlineAlert variant="warning">Isi rekening agar dana booking yang disetujui bisa ditransfer admin.</InlineAlert>}
        <BankAccountForm endpoint="/api/owner/bank-account" idPrefix="owner-bank" initial={bankAccount} onSaved={() => router.refresh()} />
      </section>

      <section className="flex flex-col gap-3">
        <h2 className="text-title-md font-semibold text-neutral-900">Payout</h2>
        {payouts.length === 0 ? (
          <p className="text-body text-neutral-600">Dana dari booking yang kamu setujui akan tercatat di sini.</p>
        ) : (
          <>
            <ul className="flex flex-col divide-y divide-neutral-200 rounded-lg border border-neutral-200 lg:hidden">
              {payouts.map((payout) => (
                <li key={payout.id} className="flex flex-col gap-1 p-3">
                  <div className="flex items-center justify-between gap-2">
                    <PayoutBadge status={payout.status} />
                    <span className="text-caption text-neutral-600">Booking #{payout.booking_id}</span>
                  </div>
                  <p className="text-price font-bold tabular-nums text-neutral-900">{payout.amount_label}</p>
                  <p className="text-caption text-neutral-600">{payoutDateText(payout)}</p>
                  {payout.transfer_reference && <p className="break-words text-caption text-neutral-600">Ref. {payout.transfer_reference}</p>}
                </li>
              ))}
            </ul>
            <div className="hidden overflow-x-auto rounded-lg border border-neutral-200 lg:block">
              <table className="w-full text-left text-body">
                <thead className="bg-neutral-50 text-caption-strong text-neutral-600">
                  <tr>
                    <th scope="col" className="px-4 py-3">Booking</th>
                    <th scope="col" className="px-4 py-3">Status</th>
                    <th scope="col" className="px-4 py-3">Tanggal</th>
                    <th scope="col" className="px-4 py-3">Referensi</th>
                    <th scope="col" className="px-4 py-3 text-right">Jumlah</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-neutral-200">
                  {payouts.map((payout) => (
                    <tr key={payout.id} className="h-[52px]">
                      <td className="px-4">#{payout.booking_id}</td>
                      <td className="px-4">
                        <PayoutBadge status={payout.status} />
                      </td>
                      <td className="px-4">{payoutDateText(payout)}</td>
                      <td className="px-4">{payout.transfer_reference ?? "–"}</td>
                      <td className="px-4 text-right font-semibold tabular-nums">{payout.amount_label}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            {hasMore && <LoadMore loading={loading} onClick={loadMore} />}
          </>
        )}
      </section>
    </main>
  );
}

function payoutDateText(payout: Payout): string {
  return payout.completed_at ? `Ditransfer ${formatDateShort(payout.completed_at)}` : `Dicatat ${formatDateShort(payout.created_at)}`;
}

function PayoutBadge({ status }: { status: string }) {
  const label = payoutStatusLabel(status);
  return <Badge color={label.color}>{label.label}</Badge>;
}
