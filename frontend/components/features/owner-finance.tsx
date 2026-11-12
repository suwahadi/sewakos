"use client";

import { Download } from "lucide-react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { BankAccountForm } from "@/components/features/bank-account-form";
import { Badge } from "@/components/ui/badge";
import { InlineAlert } from "@/components/ui/inline-alert";
import { LoadMore } from "@/components/ui/load-more";
import { MoneyRow } from "@/components/ui/money-row";
import { StatCard } from "@/components/ui/stat-card";
import { Tabs } from "@/components/ui/tabs";
import { callBff } from "@/lib/api/browser";
import { showApiError } from "@/lib/api/show-error";
import { formatDateShort } from "@/lib/format";
import { ledgerTypeLabel, payoutStatusLabel } from "@/lib/labels";
import { OWNER_PAGE_SIZE, type BankAccount, type OwnerFinanceSummary, type OwnerLedgerEntry, type Payout } from "@/lib/owner";

const SECTION_TABS = [
  { value: "ledger", label: "Buku besar" },
  { value: "payout", label: "Payout" },
  { value: "bank", label: "Rekening" },
] as const;

type Section = (typeof SECTION_TABS)[number]["value"];

const LEDGER_TYPE_TABS = [
  { value: "", label: "Semua" },
  { value: "owner_earning", label: "Pendapatan" },
  { value: "payout_completed", label: "Payout" },
  { value: "deposit_returned", label: "Deposit" },
] as const;

type LedgerType = (typeof LEDGER_TYPE_TABS)[number]["value"];

export interface OwnerFinanceProps {
  bankAccount: BankAccount | null;
  initialPayouts: Payout[];
  financeSummary: OwnerFinanceSummary | null;
  initialLedger: OwnerLedgerEntry[];
}

export function OwnerFinance({ bankAccount, initialPayouts, financeSummary, initialLedger }: OwnerFinanceProps) {
  const router = useRouter();
  const [section, setSection] = useState<Section>("ledger");
  const [payouts, setPayouts] = useState(initialPayouts);
  const [page, setPage] = useState(1);
  const [hasMore, setHasMore] = useState(initialPayouts.length >= OWNER_PAGE_SIZE);
  const [loading, setLoading] = useState(false);

  const [ledgerType, setLedgerType] = useState<LedgerType>("");
  const [ledger, setLedger] = useState(initialLedger);
  const [ledgerPage, setLedgerPage] = useState(1);
  const [ledgerHasMore, setLedgerHasMore] = useState(initialLedger.length >= OWNER_PAGE_SIZE);
  const [ledgerLoading, setLedgerLoading] = useState(false);

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

  async function loadLedger(type: LedgerType, nextPage: number, append: boolean) {
    setLedgerLoading(true);
    const typeParam = type ? `&type=${type}` : "";
    const result = await callBff<{ entries: OwnerLedgerEntry[] }>(
      `/api/owner/finance/ledger?page=${nextPage}&limit=${OWNER_PAGE_SIZE}${typeParam}`,
    );
    setLedgerLoading(false);
    if (!result.ok) return showApiError(result, (href) => router.push(href), () => loadLedger(type, nextPage, append));
    setLedger((current) => (append ? [...current, ...result.data.entries] : result.data.entries));
    setLedgerPage(nextPage);
    setLedgerHasMore(result.data.entries.length >= OWNER_PAGE_SIZE);
  }

  function changeLedgerType(type: LedgerType) {
    setLedgerType(type);
    loadLedger(type, 1, false);
  }

  const ledgerCsvHref = ledgerType ? `/api/owner/finance/ledger.csv?type=${ledgerType}` : "/api/owner/finance/ledger.csv";

  return (
    <main className="mx-auto flex max-w-[960px] flex-col gap-8 px-4 py-4">
      {financeSummary && (
        <section className="grid grid-cols-2 gap-3 lg:grid-cols-4">
          <StatCard label="Pendapatan" value={financeSummary.earnings_label} />
          <StatCard label="Payout tertransfer" value={financeSummary.payouts_completed_label} />
          <StatCard label="Saldo belum ditransfer" value={financeSummary.pending_balance_label} />
          <StatCard label="Deposit dipegang" value={financeSummary.deposits_held_label} />
        </section>
      )}

      <Tabs
        aria-label="Bagian keuangan"
        idPrefix="owner-section"
        items={SECTION_TABS.map((tab) => ({ value: tab.value, label: tab.label }))}
        value={section}
        onChange={setSection}
      />

      <div role="tabpanel" id="owner-section-panel-ledger" aria-labelledby="owner-section-tab-ledger" hidden={section !== "ledger"}>
        {section === "ledger" && (
          <section className="flex flex-col gap-3">
            <div className="flex items-center justify-between gap-3">
              <h2 className="text-title-md font-semibold text-neutral-900">Buku besar</h2>
              <a
                href={ledgerCsvHref}
                className="inline-flex h-9 items-center gap-1.5 rounded-md border border-neutral-300 px-3 text-label font-medium text-neutral-700 hover:bg-neutral-50"
              >
                <Download aria-hidden="true" size={16} strokeWidth={1.75} />
                CSV
              </a>
            </div>
            <Tabs
              aria-label="Jenis entri"
              idPrefix="owner-ledger"
              items={LEDGER_TYPE_TABS.map((tab) => ({ value: tab.value, label: tab.label }))}
              value={ledgerType}
              onChange={changeLedgerType}
            />
            {ledger.length === 0 ? (
              <p className="text-body text-neutral-600">Belum ada entri buku besar.</p>
            ) : (
              <>
                <div className="flex flex-col divide-y divide-neutral-200 rounded-lg border border-neutral-200 px-3">
                  {ledger.map((entry) => (
                    <MoneyRow
                      key={entry.id}
                      description={ledgerTypeLabel(entry.type)}
                      date={entry.occurred_at}
                      amount={entry.amount}
                      isNote={entry.type === "deposit_returned"}
                    />
                  ))}
                </div>
                {ledgerHasMore && <LoadMore loading={ledgerLoading} onClick={() => loadLedger(ledgerType, ledgerPage + 1, true)} />}
              </>
          )}
        </section>
        )}
      </div>

      <div role="tabpanel" id="owner-section-panel-payout" aria-labelledby="owner-section-tab-payout" hidden={section !== "payout"}>
        {section === "payout" && (
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
        )}
      </div>

      <div role="tabpanel" id="owner-section-panel-bank" aria-labelledby="owner-section-tab-bank" hidden={section !== "bank"}>
        {section === "bank" && (
          <section className="flex flex-col gap-4">
            <h2 className="text-title-md font-semibold text-neutral-900">Rekening penerima</h2>
            {!bankAccount && <InlineAlert variant="warning">Isi rekening agar dana booking yang disetujui bisa ditransfer admin.</InlineAlert>}
            <BankAccountForm endpoint="/api/owner/bank-account" idPrefix="owner-bank" initial={bankAccount} onSaved={() => router.refresh()} />
          </section>
        )}
      </div>

      <Link href="/owner/riwayat" className="text-label font-semibold text-brand-700">
        Lihat riwayat aktivitas
      </Link>
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
