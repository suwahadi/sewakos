"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import { ButtonPair } from "@/components/ui/button-pair";
import { Dialog } from "@/components/ui/dialog";
import { TextField } from "@/components/ui/text-field";
import type { AdminUser } from "@/lib/admin";
import { callBff } from "@/lib/api/browser";
import { showApiError } from "@/lib/api/show-error";
import { toast } from "@/lib/toast";

const MIN_LENGTH = 8;
const MAX_LENGTH = 72;

interface FieldErrors {
  password?: string;
  password_confirmation?: string;
}

function validate(password: string, confirmation: string): FieldErrors {
  const errors: FieldErrors = {};
  if (!password) errors.password = "Password wajib diisi.";
  else if (password.length < MIN_LENGTH) errors.password = `Password minimal ${MIN_LENGTH} karakter.`;
  else if (password.length > MAX_LENGTH) errors.password = `Password maksimal ${MAX_LENGTH} karakter.`;
  if (!confirmation) errors.password_confirmation = "Ulangi password wajib diisi.";
  else if (confirmation !== password) errors.password_confirmation = "Konfirmasi password tidak sama.";
  return errors;
}

export function AdminSetPasswordDialog({ user, onClose }: { user: AdminUser | null; onClose: () => void }) {
  const router = useRouter();
  const [password, setPassword] = useState("");
  const [confirmation, setConfirmation] = useState("");
  const [errors, setErrors] = useState<FieldErrors>({});
  const [saving, setSaving] = useState(false);

  function close() {
    setPassword("");
    setConfirmation("");
    setErrors({});
    onClose();
  }

  async function submit() {
    if (!user) return;
    const found = validate(password, confirmation);
    setErrors(found);
    if (found.password || found.password_confirmation) return;

    setSaving(true);
    const result = await callBff(`/api/admin/users/${user.id}/password`, {
      method: "POST",
      body: { password, password_confirmation: confirmation },
    });
    setSaving(false);
    if (!result.ok) {
      if (result.httpStatus === 422) {
        return setErrors({
          password: result.messages?.password?.[0],
          password_confirmation: result.messages?.password_confirmation?.[0] ? "Konfirmasi password tidak sama." : undefined,
        });
      }
      close();
      showApiError(result, (href) => router.push(href), submit);
      return;
    }
    toast.success(result.data.meta.message);
    close();
  }

  return (
    <Dialog open={user !== null} onClose={close} title={user ? `Set password ${user.name}` : "Set password"}>
      <form
        className="flex flex-col gap-4 pt-2"
        noValidate
        onSubmit={(event) => {
          event.preventDefault();
          submit();
        }}
      >
        <TextField
          label="Password baru"
          type="password"
          autoComplete="new-password"
          value={password}
          onChange={(event) => setPassword(event.target.value)}
          error={errors.password}
        />
        <TextField
          label="Ulangi password"
          type="password"
          autoComplete="new-password"
          value={confirmation}
          onChange={(event) => setConfirmation(event.target.value)}
          error={errors.password_confirmation}
        />
        <ButtonPair>
          <Button type="button" variant="neutral" onClick={close}>
            Batal
          </Button>
          <Button type="submit" loading={saving}>
            Simpan
          </Button>
        </ButtonPair>
      </form>
    </Dialog>
  );
}
