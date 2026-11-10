"use client";

import { useRouter, useSearchParams } from "next/navigation";
import { Suspense, useState } from "react";
import { OtpChannelPicker, type OtpChannel } from "@/components/features/otp-channel-picker";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { InlineAlert } from "@/components/ui/inline-alert";
import { OtpInput, OtpResendCountdown } from "@/components/ui/otp-input";
import { SegmentedControl } from "@/components/ui/segmented-control";
import { TextField } from "@/components/ui/text-field";
import { callBff } from "@/lib/api/browser";
import { toast } from "@/lib/toast";

const PHONE_PATTERN = /^08\d{7,12}$/;

type Step = "form" | "channel" | "code";

interface FieldErrors {
  name?: string;
  phone_number?: string;
  email?: string;
  password?: string;
  is_agree_privacy_policy?: string;
}

export default function DaftarPage() {
  return (
    <Suspense>
      <DaftarForm />
    </Suspense>
  );
}

function DaftarForm() {
  const router = useRouter();
  const next = useSearchParams().get("next") ?? "/";

  const [step, setStep] = useState<Step>("form");
  const [name, setName] = useState("");
  const [phone, setPhone] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [role, setRole] = useState<"tenant" | "owner">("tenant");
  const [agree, setAgree] = useState(false);
  const [fieldErrors, setFieldErrors] = useState<FieldErrors>({});
  const [formError, setFormError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const [channel, setChannel] = useState<OtpChannel>("0");
  const [canResendAt, setCanResendAt] = useState(0);
  const [otpError, setOtpError] = useState<string | undefined>();

  function validate(): FieldErrors {
    const errors: FieldErrors = {};
    if (!name.trim()) errors.name = "Nama wajib diisi.";
    if (!PHONE_PATTERN.test(phone)) errors.phone_number = "Nomor HP harus diawali 08 dan berisi 9–14 digit.";
    if (email && !/^\S+@\S+\.\S+$/.test(email)) errors.email = "Format email tidak valid.";
    if (password.length < 8) errors.password = "Password minimal 8 karakter.";
    if (!agree) errors.is_agree_privacy_policy = "Kamu harus menyetujui Kebijakan Privasi.";
    return errors;
  }

  async function handleSubmit() {
    const errors = validate();
    setFieldErrors(errors);
    if (Object.keys(errors).length > 0) return;

    setSubmitting(true);
    setFormError(null);
    const result = await callBff("/api/auth/register", {
      method: "POST",
      body: {
        name,
        phone_number: phone,
        email: email || undefined,
        password,
        role,
        is_agree_privacy_policy: agree,
      },
    });
    setSubmitting(false);

    if (!result.ok) {
      if (result.httpStatus === 422) {
        setFieldErrors(mapServerFieldErrors(result.messages));
      } else {
        setFormError(result.meta.message);
      }
      return;
    }
    setStep("channel");
  }

  async function sendOtp() {
    setOtpError(undefined);
    const result = await callBff<{ verification_code: { can_resend_at: number } }>("/api/auth/otp/request", {
      method: "POST",
      body: { for: "tenant_verification", via: channel, destination: phone },
    });
    if (!result.ok) {
      toast.error(result.meta.message);
      return;
    }
    setCanResendAt(result.data.verification_code.can_resend_at);
    setStep("code");
  }

  async function checkOtp(code: string) {
    setOtpError(undefined);
    const result = await callBff("/api/auth/otp/check", {
      method: "POST",
      body: { for: "tenant_verification", destination: phone, code },
    });
    if (!result.ok) {
      setOtpError(result.meta.message);
      return;
    }
    toast.success("Nomor handphone terverifikasi.");
    router.push(next);
  }

  if (step === "channel") {
    return (
      <div className="flex flex-col gap-6">
        <div className="text-center">
          <p className="text-title-lg font-semibold text-neutral-900">Verifikasi nomor HP</p>
          <p className="mt-1 text-body text-neutral-600">Pilih kanal pengiriman kode verifikasi.</p>
        </div>
        <OtpChannelPicker value={channel} onChange={setChannel} />
        <Button fullWidth onClick={sendOtp}>
          Kirim kode
        </Button>
      </div>
    );
  }

  if (step === "code") {
    return (
      <div className="flex flex-col items-center gap-6">
        <div className="text-center">
          <p className="text-title-lg font-semibold text-neutral-900">Masukkan kode verifikasi</p>
          <p className="mt-1 text-body text-neutral-600">Kode dikirim ke {phone}.</p>
        </div>
        <OtpInput onComplete={checkOtp} error={otpError} />
        <OtpResendCountdown canResendAt={canResendAt} onResend={sendOtp} />
      </div>
    );
  }

  return (
    <form
      className="flex flex-col gap-4"
      onSubmit={(event) => {
        event.preventDefault();
        handleSubmit();
      }}
    >
      <p className="text-center text-title-lg font-semibold text-neutral-900">Daftar akun</p>
      {formError && <InlineAlert variant="danger">{formError}</InlineAlert>}
      <TextField label="Nama" value={name} onChange={(event) => setName(event.target.value)} error={fieldErrors.name} />
      <TextField
        label="Nomor HP"
        type="tel"
        inputMode="numeric"
        autoComplete="tel-national"
        placeholder="081234567890"
        value={phone}
        onChange={(event) => setPhone(event.target.value)}
        error={fieldErrors.phone_number}
      />
      <TextField
        label="Email (opsional)"
        type="email"
        value={email}
        onChange={(event) => setEmail(event.target.value)}
        error={fieldErrors.email}
      />
      <TextField
        label="Password"
        type="password"
        autoComplete="new-password"
        value={password}
        onChange={(event) => setPassword(event.target.value)}
        error={fieldErrors.password}
      />
      <div className="flex flex-col gap-1.5">
        <label className="text-label font-medium text-neutral-700">Daftar sebagai</label>
        <SegmentedControl
          aria-label="Daftar sebagai"
          value={role}
          onChange={setRole}
          options={[
            { value: "tenant", label: "Penyewa" },
            { value: "owner", label: "Pemilik kost" },
          ]}
        />
      </div>
      <Checkbox
        checked={agree}
        onChange={(event) => setAgree(event.target.checked)}
        label={
          <>
            Saya menyetujui{" "}
            <a href="/kebijakan-privasi" className="text-brand-700 underline" target="_blank" rel="noreferrer">
              Kebijakan Privasi
            </a>
          </>
        }
      />
      {fieldErrors.is_agree_privacy_policy && (
        <p className="text-caption text-danger">{fieldErrors.is_agree_privacy_policy}</p>
      )}
      <Button type="submit" fullWidth loading={submitting}>
        Daftar
      </Button>
      <p className="text-center text-body text-neutral-600">
        Sudah punya akun?{" "}
        <a href="/masuk" className="text-brand-700 underline">
          Masuk
        </a>
      </p>
    </form>
  );
}

function mapServerFieldErrors(messages: Record<string, string[]> | null): FieldErrors {
  if (!messages) return {};
  const errors: FieldErrors = {};
  for (const key of ["name", "phone_number", "email", "password", "is_agree_privacy_policy"] as const) {
    if (messages[key]?.[0]) errors[key] = messages[key][0];
  }
  return errors;
}
