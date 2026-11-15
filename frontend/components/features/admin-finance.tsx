"use client";

import { Download, X } from "lucide-react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { ButtonPair } from "@/components/ui/button-pair";
import { DataTable } from "@/components/ui/data-table";
import { Dialog } from "@/components/ui/dialog";
import { LoadMore } from "@/components/ui/load-more";
import { MoneyRow } from "@/components/ui/money-row";
import { StatCard } from "@/components/ui/stat-card";
import { Tabs, type TabItem } from "@/components/ui/tabs";
import { Textarea } from "@/components/ui/textarea";
import { TextField } from "@/components/ui/text-field";
import {
  adminOwnerFinanceHref,
  type AdminFinanceSummary,
  type AdminLedgerEntry,
  type AdminReconciliationIssue,
  type AdminUser,
} from "@/lib/admin";
import { callBff } from "@/lib/api/browser";
import { showApiError } from "@/lib/api/show-error";
import { formatDateShort, formatRupiah, formatTimeWIB } from "@/lib/format";
import { ledgerTypeLabel, reconciliationKindLabel, reconciliationStatusLabel } from "@/lib/labels";
import { toast } from "@/lib/toast";
import { useAdminList } from "@/lib/use-admin-list";

const LEDGER_TYPES = [
  "payment_received",
  "admin_fee_earned",
  "owner_earning",
  "payout_completed",
  "refund_due",
  "refund_completed",
  "admin_fee_reversed",
  "deposit_returned",
] as const;

const LEDGER_TABS: TabItem<string>[] = [
  { value: "", label: "Semua" },
  ...LEDGER_TYPES.map((type) => ({ value: type, label: ledgerTypeLabel(type) })),
];

export interface FinanceOwner {
  id: number;
  name: string;
}

export function AdminFinanceSummaryAndLedger({
  summary,
  initialLedger,
  owner,
}: {
  summary: AdminFinanceSummary | null;
  initialLedger: AdminLedgerEntry[];
  owner: FinanceOwner | null;
}) {
  const list = useAdminList<AdminLedgerEntry>("finance/ledger", "entries", initialLedger, {
    type: "",
    owner_id: owner ? String(owner.id) : "",
  });
  const csvParams = new URLSearchParams(Object.entries(list.query).filter(([, value]) => value !== ""));
  const csvHref = `/api/admin/finance/ledger.csv${csvParams.size > 0 ? `?${csvParams}` : ""}`;

  return (
    <div className="flex flex-col gap-6">
      <h1 className="text-title-lg font-semibold text-neutral-900">Keuangan platform</h1>
      <OwnerFilter owner={owner} />
      {summary && (
        <section className="grid grid-cols-2 gap-3 lg:grid-cols-4">
          <StatCard label="Tagihan dibayar" value={summary.payments_received_label} />
          <StatCard label="Pendapatan biaya admin" value={summary.admin_fee_revenue_label} />
          <StatCard label="Refund tertransfer" value={summary.refunds_completed_label} caption={`Belum ditransfer ${summary.refunds_outstanding_label}`} />
          <StatCard label="Payout tertransfer" value={summary.payouts_completed_label} caption={`Belum ditransfer ${summary.payouts_outstanding_label}`} />
        </section>
      )}

      <section className="flex flex-col gap-3">
        <div className="flex items-center justify-between gap-3">
          <h2 className="text-title-md font-semibold text-neutral-900">Buku besar</h2>
          <a
            href={csvHref}
            className="inline-flex h-9 items-center gap-1.5 rounded-md border border-neutral-300 px-3 text-label font-medium text-neutral-700 hover:bg-neutral-50"
          >
            <Download aria-hidden="true" size={16} strokeWidth={1.75} />
            CSV
          </a>
        </div>
        <div className="border-b border-neutral-200">
          <Tabs items={LEDGER_TABS} value={list.query.type ?? ""} onChange={(type) => list.filter({ ...list.query, type })} aria-label="Jenis entri" idPrefix="admin-ledger" />
        </div>
        {list.rows.length === 0 && !list.loading ? (
          <p className="text-body text-neutral-600">Tidak ada entri buku besar untuk jenis ini.</p>
        ) : (
          <div className="flex flex-col divide-y divide-neutral-200 rounded-lg border border-neutral-200 px-3">
            {list.rows.map((entry) => (
              <MoneyRow
                key={entry.id}
                description={ledgerTypeLabel(entry.type)}
                detail={
                  entry.owner_id && !owner ? (
                    <Link href={adminOwnerFinanceHref(entry.owner_id)} className="text-brand-700 underline">
                      {entry.owner_name ?? `Pemilik #${entry.owner_id}`}
                    </Link>
                  ) : undefined
                }
                date={entry.occurred_at}
                amount={entry.amount}
                isNote={entry.type === "deposit_returned"}
              />
            ))}
          </div>
        )}
        {list.hasMore && <LoadMore loading={list.loading} onClick={list.loadMore} />}
      </section>
    </div>
  );
}

function OwnerFilter({ owner }: { owner: FinanceOwner | null }) {
  const router = useRouter();
  const [search, setSearch] = useState("");
  const [error, setError] = useState<string>();
  const [results, setResults] = useState<AdminUser[] | null>(null);
  const [searching, setSearching] = useState(false);

  if (owner) {
    return (
      <div className="flex flex-wrap items-center gap-2">
        <span className="text-label text-neutral-600">Difilter:</span>
        <span className="inline-flex h-8 items-center gap-1 rounded-full border border-brand-600 bg-brand-50 pl-3 pr-1 text-label font-medium text-brand-800">
          Pemilik: {owner.name}
          <button
            type="button"
            aria-label={`Hapus filter pemilik ${owner.name}`}
            onClick={() => router.push("/admin/keuangan")}
            className="inline-flex size-6 items-center justify-center rounded-full hover:bg-brand-100"
          >
            <X aria-hidden="true" size={14} strokeWidth={2} />
          </button>
        </span>
      </div>
    );
  }

  async function findOwners() {
    const q = search.trim();
    if (q.length < 3) return setError("Masukkan minimal 3 karakter.");
    setError(undefined);
    setSearching(true);
    const result = await callBff<{ users: AdminUser[] }>(`/api/admin/users?${new URLSearchParams({ role: "owner", q, limit: "5" })}`);
    setSearching(false);
    if (!result.ok) return showApiError(result, (href) => router.push(href), findOwners);
    setResults(result.data.users);
  }

  return (
    <section className="flex flex-col gap-2">
      <form
        className="grid gap-3 sm:grid-cols-[minmax(0,1fr)_auto] sm:items-start"
        onSubmit={(event) => {
          event.preventDefault();
          findOwners();
        }}
      >
        <TextField label="Filter per pemilik" placeholder="Nama atau nomor HP pemilik" value={search} onChange={(event) => setSearch(event.target.value)} error={error} />
        <Button type="submit" variant="neutral" loading={searching} className="sm:mt-[26px]">
          Cari
        </Button>
      </form>
      {results && results.length === 0 && <p className="text-body text-neutral-600">Pemilik tidak ditemukan.</p>}
      {results && results.length > 0 && (
        <ul aria-label="Hasil pencarian pemilik" className="flex flex-col divide-y divide-neutral-200 rounded-lg border border-neutral-200">
          {results.map((user) => (
            <li key={user.id}>
              <Link href={adminOwnerFinanceHref(user.id)} className="flex min-h-11 items-center px-3 text-body text-neutral-900 hover:bg-neutral-50">
                {user.name}
              </Link>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}

const RECONCILIATION_TABS: TabItem<"open" | "resolved">[] = [
  { value: "open", label: "Terbuka" },
  { value: "resolved", label: "Selesai" },
];

function ResolveDialog({ issue, onClose, onDone }: { issue: AdminReconciliationIssue | null; onClose: () => void; onDone: () => void }) {
  const router = useRouter();
  const [note, setNote] = useState("");
  const [error, setError] = useState<string>();
  const [saving, setSaving] = useState(false);

  async function submit() {
    if (!issue) return;
    if (!note.trim()) return setError("Catatan wajib diisi.");
    setSaving(true);
    const result = await callBff(`/api/admin/finance/reconciliation/${issue.id}/resolve`, { method: "POST", body: { note: note.trim() } });
    setSaving(false);
    if (!result.ok) {
      if (result.httpStatus === 422) return setError(result.messages?.note?.[0] ?? result.meta.message);
      onClose();
      showApiError(result, (href) => router.push(href), submit);
      return;
    }
    toast.success(result.data.meta.message);
    setNote("");
    setError(undefined);
    onDone();
  }

  return (
    <Dialog open={issue !== null} onClose={onClose} title={issue ? `Selisih ${issue.order_id}` : "Selisih"}>
      <div className="flex flex-col gap-4 pt-2">
        <p>Tandai selisih ini sudah ditangani. Perubahan data tetap lewat alur normal, ini hanya mencatat.</p>
        <Textarea label="Catatan" maxLength={500} value={note} onChange={(event) => setNote(event.target.value)} error={error} />
        <ButtonPair>
          <Button variant="neutral" onClick={onClose}>
            Batal
          </Button>
          <Button loading={saving} onClick={submit}>
            Tandai selesai
          </Button>
        </ButtonPair>
      </div>
    </Dialog>
  );
}

export function AdminReconciliationList({ initial }: { initial: AdminReconciliationIssue[] }) {
  const list = useAdminList<AdminReconciliationIssue>("finance/reconciliation", "issues", initial, { status: "open" });
  const [resolving, setResolving] = useState<AdminReconciliationIssue | null>(null);

  return (
    <div className="flex flex-col gap-4">
      <h1 className="text-title-lg font-semibold text-neutral-900">Rekonsiliasi</h1>
      <div className="border-b border-neutral-200">
        <Tabs
          items={RECONCILIATION_TABS}
          value={(list.query.status ?? "open") as "open" | "resolved"}
          onChange={(status) => list.filter({ status })}
          aria-label="Status selisih"
          idPrefix="admin-reconciliation"
        />
      </div>
      {list.rows.length === 0 && !list.loading ? (
        <p className="text-body text-neutral-600">Tidak ada selisih dengan status ini.</p>
      ) : (
        <DataTable
          caption="Daftar selisih rekonsiliasi"
          rows={list.rows}
          rowKey={(issue) => issue.id}
          columns={[
            { key: "order", header: "Order", render: (issue) => issue.order_id },
            { key: "kind", header: "Jenis", render: (issue) => reconciliationKindLabel(issue.kind) },
            { key: "local", header: "Status sistem", render: (issue) => issue.local_status },
            { key: "midtrans", header: "Status pembayaran", render: (issue) => issue.midtrans_status ?? "–" },
            {
              key: "amount",
              header: "Nominal",
              render: (issue) => `${formatRupiah(issue.local_amount)} / ${issue.midtrans_amount !== null ? formatRupiah(issue.midtrans_amount) : "–"}`,
              align: "right",
            },
            { key: "detected", header: "Terdeteksi", render: (issue) => `${formatDateShort(issue.detected_at)}, ${formatTimeWIB(issue.detected_at)}` },
            {
              key: "status",
              header: "Status",
              render: (issue) => {
                const label = reconciliationStatusLabel(issue.status);
                return <Badge color={label.color}>{label.label}</Badge>;
              },
            },
            {
              key: "action",
              header: "Aksi",
              render: (issue) =>
                issue.status === "open" ? (
                  <Button size="sm" onClick={() => setResolving(issue)}>
                    Tandai selesai
                  </Button>
                ) : (
                  issue.note ?? "–"
                ),
            },
          ]}
        />
      )}
      {list.hasMore && <LoadMore loading={list.loading} onClick={list.loadMore} />}
      <ResolveDialog
        issue={resolving}
        onClose={() => setResolving(null)}
        onDone={() => {
          setResolving(null);
          list.reload();
        }}
      />
    </div>
  );
}
