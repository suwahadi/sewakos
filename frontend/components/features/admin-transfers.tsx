"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { ButtonPair } from "@/components/ui/button-pair";
import { DataTable } from "@/components/ui/data-table";
import { Dialog } from "@/components/ui/dialog";
import { LoadMore } from "@/components/ui/load-more";
import { Tabs, type TabItem } from "@/components/ui/tabs";
import { Textarea } from "@/components/ui/textarea";
import { TextField } from "@/components/ui/text-field";
import { refundReasonLabel, type AdminBankAccount, type AdminPayout, type AdminRefund } from "@/lib/admin";
import { callBff } from "@/lib/api/browser";
import { showApiError } from "@/lib/api/show-error";
import { formatDateShort } from "@/lib/format";
import { payoutStatusLabel, refundStatusLabel } from "@/lib/labels";
import { toast } from "@/lib/toast";
import { useAdminList } from "@/lib/use-admin-list";

function accountText(account: AdminBankAccount | null): string {
  return account ? `${account.bank_name} ${account.account_number} a.n. ${account.account_name}` : "Belum diisi";
}

function BookingLink({ id }: { id: number }) {
  return (
    <Link href={`/admin/booking/${id}`} className="text-brand-700 underline">
      #{id}
    </Link>
  );
}

function TransferDialog({ title, endpoint, onClose, onDone }: { title: string; endpoint: string | null; onClose: () => void; onDone: () => void }) {
  const router = useRouter();
  const [reference, setReference] = useState("");
  const [note, setNote] = useState("");
  const [error, setError] = useState<string>();
  const [saving, setSaving] = useState(false);

  async function submit() {
    if (!endpoint) return;
    if (!reference.trim()) return setError("Nomor referensi transfer wajib diisi.");
    setSaving(true);
    const result = await callBff(endpoint, { method: "POST", body: { transfer_reference: reference.trim(), note: note.trim() || undefined } });
    setSaving(false);
    if (!result.ok) {
      if (result.httpStatus === 422) return setError(result.messages?.transfer_reference?.[0] ?? result.meta.message);
      onClose();
      showApiError(result, (href) => router.push(href), submit);
      return;
    }
    toast.success(result.data.meta.message);
    setReference("");
    setNote("");
    setError(undefined);
    onDone();
  }

  return (
    <Dialog open={endpoint !== null} onClose={onClose} title={title}>
      <div className="flex flex-col gap-4 pt-2">
        <p>Catat transfer setelah dana benar-benar dikirim. Tindakan ini tidak bisa dibatalkan.</p>
        <TextField label="Nomor referensi transfer" maxLength={100} value={reference} onChange={(event) => setReference(event.target.value)} error={error} />
        <Textarea label="Catatan (opsional)" maxLength={500} value={note} onChange={(event) => setNote(event.target.value)} />
        <ButtonPair>
          <Button variant="outline" onClick={onClose}>
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

type RefundTab = "ready" | "awaiting_account" | "completed";
const REFUND_TABS: TabItem<RefundTab>[] = [
  { value: "ready", label: "Siap diproses" },
  { value: "awaiting_account", label: "Perlu rekening" },
  { value: "completed", label: "Selesai" },
];

export function AdminRefundList({ initial }: { initial: AdminRefund[] }) {
  const list = useAdminList<AdminRefund>("refunds", "refunds", initial, { status: "ready" });
  const [completing, setCompleting] = useState<AdminRefund | null>(null);

  return (
    <div className="flex flex-col gap-4">
      <h1 className="text-title-lg font-semibold text-neutral-900">Refund</h1>
      <div className="border-b border-neutral-200">
        <Tabs items={REFUND_TABS} value={(list.query.status ?? "ready") as RefundTab} onChange={(status) => list.filter({ status })} aria-label="Status refund" idPrefix="admin-refund" />
      </div>
      {list.rows.length === 0 && !list.loading ? (
        <p className="text-body text-neutral-600">Tidak ada refund dengan status ini.</p>
      ) : (
        <DataTable
          caption="Daftar refund"
          rows={list.rows}
          rowKey={(refund) => refund.id}
          columns={[
            { key: "booking", header: "Booking", render: (refund) => <BookingLink id={refund.booking_id} /> },
            { key: "tenant", header: "Penyewa", render: (refund) => refund.tenant.name },
            { key: "reason", header: "Alasan", render: (refund) => refundReasonLabel(refund.reason) },
            { key: "account", header: "Rekening", render: (refund) => accountText(refund.bank_account) },
            {
              key: "status",
              header: "Status",
              render: (refund) => {
                const label = refundStatusLabel(refund.status);
                return <Badge color={label.color}>{label.label}</Badge>;
              },
            },
            { key: "date", header: "Dibuat", render: (refund) => formatDateShort(refund.created_at) },
            { key: "amount", header: "Jumlah", render: (refund) => refund.amount_label, align: "right" },
            {
              key: "action",
              header: "Aksi",
              render: (refund) =>
                refund.status === "ready" ? (
                  <Button size="sm" onClick={() => setCompleting(refund)}>
                    Tandai selesai
                  </Button>
                ) : (
                  refund.transfer_reference ?? "–"
                ),
            },
          ]}
        />
      )}
      {list.hasMore && <LoadMore loading={list.loading} onClick={list.loadMore} />}
      <TransferDialog
        title={completing ? `Refund booking #${completing.booking_id}` : "Refund"}
        endpoint={completing ? `/api/admin/refunds/${completing.id}/complete` : null}
        onClose={() => setCompleting(null)}
        onDone={() => {
          setCompleting(null);
          list.reload();
        }}
      />
    </div>
  );
}

type PayoutTab = "pending" | "completed";
const PAYOUT_TABS: TabItem<PayoutTab>[] = [
  { value: "pending", label: payoutStatusLabel("pending").label },
  { value: "completed", label: payoutStatusLabel("completed").label },
];

export function AdminPayoutList({ initial }: { initial: AdminPayout[] }) {
  const list = useAdminList<AdminPayout>("payouts", "payouts", initial, { status: "pending" });
  const [completing, setCompleting] = useState<AdminPayout | null>(null);

  return (
    <div className="flex flex-col gap-4">
      <h1 className="text-title-lg font-semibold text-neutral-900">Payout</h1>
      <div className="border-b border-neutral-200">
        <Tabs items={PAYOUT_TABS} value={(list.query.status ?? "pending") as PayoutTab} onChange={(status) => list.filter({ status })} aria-label="Status payout" idPrefix="admin-payout" />
      </div>
      {list.rows.length === 0 && !list.loading ? (
        <p className="text-body text-neutral-600">Tidak ada payout dengan status ini.</p>
      ) : (
        <DataTable
          caption="Daftar payout"
          rows={list.rows}
          rowKey={(payout) => payout.id}
          columns={[
            { key: "booking", header: "Booking", render: (payout) => <BookingLink id={payout.booking_id} /> },
            { key: "owner", header: "Pemilik", render: (payout) => payout.owner.name },
            { key: "account", header: "Rekening", render: (payout) => accountText(payout.bank_account) },
            {
              key: "status",
              header: "Status",
              render: (payout) => {
                const label = payoutStatusLabel(payout.status);
                return <Badge color={label.color}>{label.label}</Badge>;
              },
            },
            { key: "date", header: "Dibuat", render: (payout) => formatDateShort(payout.created_at) },
            { key: "amount", header: "Jumlah", render: (payout) => payout.amount_label, align: "right" },
            {
              key: "action",
              header: "Aksi",
              render: (payout) =>
                payout.status === "pending" ? (
                  <Button size="sm" disabled={!payout.bank_account} onClick={() => setCompleting(payout)}>
                    Tandai selesai
                  </Button>
                ) : (
                  payout.transfer_reference ?? "–"
                ),
            },
          ]}
        />
      )}
      {list.hasMore && <LoadMore loading={list.loading} onClick={list.loadMore} />}
      <TransferDialog
        title={completing ? `Payout booking #${completing.booking_id}` : "Payout"}
        endpoint={completing ? `/api/admin/payouts/${completing.id}/complete` : null}
        onClose={() => setCompleting(null)}
        onDone={() => {
          setCompleting(null);
          list.reload();
        }}
      />
    </div>
  );
}
