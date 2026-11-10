"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import { ConfirmDialog } from "@/components/ui/dialog";
import { TextField } from "@/components/ui/text-field";
import { callBff } from "@/lib/api/browser";
import { formatPhoneDisplay } from "@/lib/format";
import { toast } from "@/lib/toast";

export interface AkunUser {
  id: number;
  name: string;
  role: string;
  phone_number: string;
  email: string | null;
  is_verified: boolean;
}

export function AkunClient({ user }: { user: AkunUser }) {
  return (
    <main className="mx-auto flex max-w-[480px] flex-col gap-8 px-4 py-6">
      <ProfileSection user={user} />
      <PasswordSection />
      <PhoneSection currentPhone={user.phone_number} />
      <LogoutSection />
    </main>
  );
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <section className="flex flex-col gap-4 border-b border-neutral-200 pb-8 last:border-0">
      <p className="text-title-sm font-semibold text-neutral-900">{title}</p>
      {children}
    </section>
  );
}

function ProfileSection({ user }: { user: AkunUser }) {
  const [name, setName] = useState(user.name);
  const [email, setEmail] = useState(user.email ?? "");
  const [errors, setErrors] = useState<{ name?: string; email?: string }>({});
  const [submitting, setSubmitting] = useState(false);

  async function handleSave() {
    setSubmitting(true);
    setErrors({});
    const result = await callBff("/api/me", { method: "PATCH", body: { name, email: email || null } });
    setSubmitting(false);

    if (!result.ok) {
      if (result.httpStatus === 422) {
        setErrors({ name: result.messages?.name?.[0], email: result.messages?.email?.[0] });
      } else {
        toast.error(result.meta.message);
      }
      return;
    }
    toast.success("Profil diperbarui.");
  }

  return (
    <Section title="Profil">
      <TextField label="Nomor HP" value={formatPhoneDisplay(user.phone_number)} disabled readOnly />
      <TextField label="Nama" value={name} onChange={(event) => setName(event.target.value)} error={errors.name} />
      <TextField
        label="Email (opsional)"
        type="email"
        value={email}
        onChange={(event) => setEmail(event.target.value)}
        error={errors.email}
      />
      <Button onClick={handleSave} loading={submitting}>
        Simpan profil
      </Button>
    </Section>
  );
}

function PasswordSection() {
  const [currentPassword, setCurrentPassword] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | undefined>();
  const [submitting, setSubmitting] = useState(false);

  async function handleSave() {
    setSubmitting(true);
    setError(undefined);
    const result = await callBff("/api/me/password", {
      method: "PUT",
      body: { current_password: currentPassword, password },
    });
    setSubmitting(false);

    if (!result.ok) {
      if (result.httpStatus === 422) {
        setError(result.messages?.current_password?.[0] ?? result.messages?.password?.[0]);
      } else {
        toast.error(result.meta.message);
      }
      return;
    }
    setCurrentPassword("");
    setPassword("");
    toast.success("Password berhasil diubah.");
  }

  return (
    <Section title="Ganti password">
      <TextField
        label="Password saat ini"
        type="password"
        autoComplete="current-password"
        value={currentPassword}
        onChange={(event) => setCurrentPassword(event.target.value)}
        error={error}
      />
      <TextField
        label="Password baru"
        type="password"
        autoComplete="new-password"
        value={password}
        onChange={(event) => setPassword(event.target.value)}
      />
      <Button onClick={handleSave} loading={submitting}>
        Simpan password
      </Button>
    </Section>
  );
}

function PhoneSection({ currentPhone }: { currentPhone: string }) {
  const [step, setStep] = useState<"form" | "verify">("form");
  const [newPhone, setNewPhone] = useState("");
  const [code, setCode] = useState("");
  const [error, setError] = useState<string | undefined>();
  const [submitting, setSubmitting] = useState(false);

  async function requestChange() {
    setSubmitting(true);
    setError(undefined);
    const result = await callBff("/api/me/phone/request", { method: "POST", body: { phone_number: newPhone } });
    setSubmitting(false);

    if (!result.ok) {
      setError(result.httpStatus === 422 ? result.messages?.phone_number?.[0] : result.meta.message);
      return;
    }
    setStep("verify");
  }

  async function verifyChange() {
    setSubmitting(true);
    setError(undefined);
    const result = await callBff("/api/me/phone/verify", {
      method: "POST",
      body: { phone_number: newPhone, code },
    });
    setSubmitting(false);

    if (!result.ok) {
      setError(result.meta.message);
      return;
    }
    toast.success("Nomor handphone diperbarui.");
    setStep("form");
    setNewPhone("");
    setCode("");
  }

  return (
    <Section title="Ganti nomor HP">
      <p className="text-body text-neutral-600">Nomor saat ini: {formatPhoneDisplay(currentPhone)}</p>
      {step === "form" ? (
        <>
          <TextField
            label="Nomor HP baru"
            type="tel"
            inputMode="numeric"
            placeholder="081234567890"
            value={newPhone}
            onChange={(event) => setNewPhone(event.target.value)}
            error={error}
          />
          <Button onClick={requestChange} loading={submitting}>
            Kirim kode verifikasi
          </Button>
        </>
      ) : (
        <>
          <p className="text-body text-neutral-600">Kode dikirim ke {newPhone}.</p>
          <TextField label="Kode verifikasi" inputMode="numeric" value={code} onChange={(event) => setCode(event.target.value)} error={error} />
          <Button onClick={verifyChange} loading={submitting}>
            Konfirmasi
          </Button>
        </>
      )}
    </Section>
  );
}

function LogoutSection() {
  const router = useRouter();
  const [open, setOpen] = useState(false);
  const [loading, setLoading] = useState(false);

  async function handleLogout() {
    setLoading(true);
    await callBff("/api/auth/logout", { method: "POST" });
    setLoading(false);
    setOpen(false);
    router.push("/masuk");
  }

  return (
    <section>
      <Button variant="outline" fullWidth onClick={() => setOpen(true)}>
        Keluar
      </Button>
      <ConfirmDialog
        open={open}
        onClose={() => setOpen(false)}
        title="Keluar dari akun?"
        description="Kamu perlu masuk kembali untuk mengakses akunmu."
        confirmLabel="Keluar"
        onConfirm={handleLogout}
        loading={loading}
      />
    </section>
  );
}
