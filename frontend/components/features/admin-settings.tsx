"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import { ConfirmDialog } from "@/components/ui/dialog";
import { Select } from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import { TextField } from "@/components/ui/text-field";
import { callBff } from "@/lib/api/browser";
import { showApiError } from "@/lib/api/show-error";
import { digitsOnly, formatThousands } from "@/lib/format";
import { toast } from "@/lib/toast";

const TARGET_LABEL = { all: "semua pengguna", tenant: "semua penyewa", owner: "semua pemilik" } as const;
type Target = keyof typeof TARGET_LABEL;

export function AdminSettings({ adminFee }: { adminFee: number }) {
  const router = useRouter();
  const navigate = (href: string) => router.push(href);
  const [fee, setFee] = useState(String(adminFee));
  const [feeError, setFeeError] = useState<string>();
  const [savingFee, setSavingFee] = useState(false);
  const [target, setTarget] = useState<Target>("tenant");
  const [title, setTitle] = useState("");
  const [body, setBody] = useState("");
  const [broadcastErrors, setBroadcastErrors] = useState<{ title?: string; body?: string }>({});
  const [confirming, setConfirming] = useState(false);
  const [sending, setSending] = useState(false);

  async function saveFee() {
    const value = Number(fee || 0);
    if (value > 1_000_000) return setFeeError("Biaya admin maksimal Rp1.000.000.");
    setFeeError(undefined);
    setSavingFee(true);
    const result = await callBff("/api/admin/settings", { method: "PATCH", body: { admin_fee: value } });
    setSavingFee(false);
    if (!result.ok) return showApiError(result, navigate, saveFee);
    toast.success(result.data.meta.message);
    router.refresh();
  }

  function askBroadcast() {
    const errors = { title: title.trim() ? undefined : "Judul wajib diisi.", body: body.trim() ? undefined : "Isi pesan wajib diisi." };
    setBroadcastErrors(errors);
    if (!errors.title && !errors.body) setConfirming(true);
  }

  async function send() {
    setSending(true);
    const result = await callBff<{ sent: number }>("/api/admin/notifications/broadcast", {
      method: "POST",
      body: { target, title: title.trim(), body: body.trim() },
    });
    setSending(false);
    setConfirming(false);
    if (!result.ok) return showApiError(result, navigate, send);
    toast.success(result.data.meta.message, { description: `Terkirim ke ${result.data.sent} pengguna.` });
    setTitle("");
    setBody("");
  }

  return (
    <div className="flex max-w-xl flex-col gap-8">
      <h1 className="text-title-lg font-semibold text-neutral-900">Pengaturan</h1>
      <section className="flex flex-col gap-4">
        <h2 className="text-title-md font-semibold text-neutral-900">Biaya admin</h2>
        <TextField
          label="Biaya admin per tagihan"
          prefix="Rp"
          inputMode="numeric"
          helper="Berlaku untuk booking yang diajukan setelah disimpan."
          value={formatThousands(fee)}
          onChange={(event) => setFee(digitsOnly(event.target.value))}
          error={feeError}
        />
        <Button loading={savingFee} onClick={saveFee}>
          Simpan biaya admin
        </Button>
      </section>
      <section className="flex flex-col gap-4">
        <h2 className="text-title-md font-semibold text-neutral-900">Kirim notifikasi promo</h2>
        <Select label="Penerima" value={target} onChange={(event) => setTarget(event.target.value as Target)}>
          <option value="tenant">Semua penyewa</option>
          <option value="owner">Semua pemilik</option>
          <option value="all">Semua pengguna</option>
        </Select>
        <TextField label="Judul" maxLength={255} value={title} onChange={(event) => setTitle(event.target.value)} error={broadcastErrors.title} />
        <Textarea label="Isi pesan" maxLength={1000} value={body} onChange={(event) => setBody(event.target.value)} error={broadcastErrors.body} />
        <Button variant="secondary" onClick={askBroadcast}>
          Kirim notifikasi
        </Button>
      </section>
      <ConfirmDialog
        open={confirming}
        onClose={() => setConfirming(false)}
        title="Kirim notifikasi promo?"
        description={`Notifikasi "${title.trim()}" dikirim ke ${TARGET_LABEL[target]} dan tidak bisa ditarik kembali.`}
        confirmLabel="Kirim"
        destructive={false}
        loading={sending}
        onConfirm={send}
      />
    </div>
  );
}
