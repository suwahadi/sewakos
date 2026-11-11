"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import { TextField } from "@/components/ui/text-field";
import { callBff } from "@/lib/api/browser";
import { showApiError } from "@/lib/api/show-error";
import type { BankAccount } from "@/lib/owner";
import { toast } from "@/lib/toast";

type AccountField = keyof BankAccount;
type AccountErrors = Partial<Record<AccountField, string>>;

const FIELD_ORDER: AccountField[] = ["bank_name", "account_number", "account_name"];

function validateAccount(account: BankAccount): AccountErrors {
  const errors: AccountErrors = {};
  if (!account.bank_name.trim()) errors.bank_name = "Nama bank wajib diisi.";
  if (!/^\d{5,20}$/.test(account.account_number)) errors.account_number = "Nomor rekening harus berisi 5–20 digit angka.";
  if (!account.account_name.trim()) errors.account_name = "Nama pemilik rekening wajib diisi.";
  return errors;
}

export interface BankAccountFormProps {
  endpoint: string;
  idPrefix: string;
  initial?: BankAccount | null;
  onSaved?: () => void;
  onRejected?: () => void;
}

export function BankAccountForm({ endpoint, idPrefix, initial, onSaved, onRejected }: BankAccountFormProps) {
  const router = useRouter();
  const [account, setAccount] = useState<BankAccount>(initial ?? { bank_name: "", account_number: "", account_name: "" });
  const [errors, setErrors] = useState<AccountErrors>({});
  const [submitting, setSubmitting] = useState(false);

  function showErrors(next: AccountErrors) {
    setErrors(next);
    const first = FIELD_ORDER.find((field) => next[field]);
    if (first) document.getElementById(`${idPrefix}-${first}`)?.focus();
  }

  async function save() {
    const clientErrors = validateAccount(account);
    if (Object.keys(clientErrors).length > 0) return showErrors(clientErrors);
    setErrors({});
    setSubmitting(true);
    const result = await callBff(endpoint, { method: "PUT", body: account });
    setSubmitting(false);

    if (result.ok) {
      toast.success(result.data.meta.message);
      onSaved?.();
      return;
    }
    if (result.httpStatus === 422) {
      return showErrors(Object.fromEntries(FIELD_ORDER.map((field) => [field, result.messages?.[field]?.[0]])));
    }
    showApiError(result, (href) => router.push(href), save);
    if (result.httpStatus === 400) onRejected?.();
  }

  function update(field: AccountField, value: string) {
    setAccount((current) => ({ ...current, [field]: value }));
  }

  return (
    <div className="flex flex-col gap-4">
      <TextField
        id={`${idPrefix}-bank_name`}
        label="Nama bank"
        placeholder="BCA"
        maxLength={100}
        value={account.bank_name}
        onChange={(event) => update("bank_name", event.target.value)}
        error={errors.bank_name}
      />
      <TextField
        id={`${idPrefix}-account_number`}
        label="Nomor rekening"
        inputMode="numeric"
        maxLength={20}
        value={account.account_number}
        onChange={(event) => update("account_number", event.target.value.trim())}
        error={errors.account_number}
      />
      <TextField
        id={`${idPrefix}-account_name`}
        label="Nama pemilik rekening"
        autoComplete="name"
        maxLength={255}
        value={account.account_name}
        onChange={(event) => update("account_name", event.target.value)}
        error={errors.account_name}
      />
      <Button fullWidth loading={submitting} onClick={save}>
        Simpan rekening
      </Button>
    </div>
  );
}
