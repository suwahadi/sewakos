"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import { ButtonPair } from "@/components/ui/button-pair";
import { ConfirmDialog } from "@/components/ui/dialog";
import { Textarea } from "@/components/ui/textarea";
import { callBff } from "@/lib/api/browser";
import { showApiError } from "@/lib/api/show-error";
import { toast } from "@/lib/toast";

type Action = "approve" | "reject" | "suspend";

const REASON_DIALOG: Record<Exclude<Action, "approve">, { title: string; description: string; confirm: string }> = {
  reject: { title: "Tolak properti ini?", description: "Alasan dikirim ke pemilik agar bisa memperbaiki dan mengajukan ulang.", confirm: "Tolak properti" },
  suspend: { title: "Nonaktifkan properti ini?", description: "Properti berhenti tayang. Booking yang sudah dibayar tetap diproses.", confirm: "Nonaktifkan" },
};

export function AdminPropertyActions({ propertyId, status }: { propertyId: number; status: string }) {
  const router = useRouter();
  const [busy, setBusy] = useState<Action | null>(null);
  const [dialog, setDialog] = useState<Exclude<Action, "approve"> | null>(null);
  const [reason, setReason] = useState("");
  const [reasonError, setReasonError] = useState<string>();

  async function run(action: Action) {
    if (action !== "approve" && !reason.trim()) return setReasonError("Alasan wajib diisi.");
    setBusy(action);
    const result = await callBff(`/api/admin/properties/${propertyId}/${action}`, {
      method: "POST",
      body: action === "approve" ? {} : { reason: reason.trim() },
    });
    setBusy(null);
    if (!result.ok) {
      if (result.httpStatus === 422) return setReasonError(result.messages?.reason?.[0]);
      setDialog(null);
      showApiError(result, (href) => router.push(href), () => run(action));
      return;
    }
    setDialog(null);
    setReason("");
    toast.success(result.data.meta.message);
    router.refresh();
  }

  function openDialog(action: Exclude<Action, "approve">) {
    setReason("");
    setReasonError(undefined);
    setDialog(action);
  }

  return (
    <div className="max-w-md">
      {status === "pending" && (
        <ButtonPair>
          <Button variant="danger" onClick={() => openDialog("reject")}>
            Tolak properti
          </Button>
          <Button loading={busy === "approve"} onClick={() => run("approve")}>
            Setujui properti
          </Button>
        </ButtonPair>
      )}
      {status === "suspended" && (
        <Button fullWidth loading={busy === "approve"} onClick={() => run("approve")}>
          Aktifkan kembali
        </Button>
      )}
      {status === "approved" && (
        <Button variant="danger" fullWidth onClick={() => openDialog("suspend")}>
          Nonaktifkan properti
        </Button>
      )}
      {dialog && (
        <ConfirmDialog
          open
          onClose={() => setDialog(null)}
          title={REASON_DIALOG[dialog].title}
          description={
            <div className="flex flex-col gap-3">
              <p>{REASON_DIALOG[dialog].description}</p>
              <Textarea label="Alasan" maxLength={500} value={reason} onChange={(event) => setReason(event.target.value)} error={reasonError} />
            </div>
          }
          confirmLabel={REASON_DIALOG[dialog].confirm}
          loading={busy === dialog}
          onConfirm={() => run(dialog)}
        />
      )}
    </div>
  );
}
