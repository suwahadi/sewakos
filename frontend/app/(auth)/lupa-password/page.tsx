"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";
import { OtpChannelPicker, type OtpChannel } from "@/components/features/otp-channel-picker";
import { Button } from "@/components/ui/button";
import { InlineAlert } from "@/components/ui/inline-alert";
import { OtpResendCountdown } from "@/components/ui/otp-input";
import { TextField } from "@/components/ui/text-field";
import { callBff } from "@/lib/api/browser";
import { toast } from "@/lib/toast";

const PHONE_PATTERN = /^08\d{7,12}$/;

type Step = "phone" | "channel" | "reset";

export default function LupaPasswordPage() {
  const router = useRouter();
  const [step, setStep] = useState<Step>("phone");
  const [phone, setPhone] = useState("");
  const [phoneError, setPhoneError] = useState<string | undefined>();
  const [channel, setChannel] = useState<OtpChannel>("0");
  const [canResendAt, setCanResendAt] = useState(0);

  const [code, setCode] = useState("");
  const [password, setPassword] = useState("");
  const [resetError, setResetError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  function continueFromPhone() {
    if (!PHONE_PATTERN.test(phone)) {
      setPhoneError("Nomor HP harus diawali 08 dan berisi 9–14 digit.");
      return;
    }
    setPhoneError(undefined);
    setStep("channel");
  }

  async function requestOtp() {
    const result = await callBff<{ verification_code: { can_resend_at: number } }>("/api/auth/otp/request", {
      method: "POST",
      body: { for: "reset_password", via: channel, destination: phone },
    });
    if (!result.ok) {
      toast.error(result.meta.message);
      return;
    }
    setCanResendAt(result.data.verification_code.can_resend_at);
    setStep("reset");
  }

  async function handleReset() {
    if (password.length < 8) {
      setResetError("Password minimal 8 karakter.");
      return;
    }
    setSubmitting(true);
    setResetError(null);
    const result = await callBff("/api/auth/password/reset", {
      method: "POST",
      body: { destination: phone, code, password },
    });
    setSubmitting(false);

    if (!result.ok) {
      setResetError(result.meta.message);
      return;
    }
    toast.success("Password berhasil diubah. Silakan login kembali.");
    router.push("/masuk");
  }

  if (step === "channel") {
    return (
      <div className="flex flex-col gap-4">
        <p className="text-center text-title-lg font-semibold text-neutral-900">Pilih kanal kode</p>
        <OtpChannelPicker value={channel} onChange={setChannel} />
        <Button fullWidth onClick={requestOtp}>
          Kirim kode
        </Button>
      </div>
    );
  }

  if (step === "reset") {
    return (
      <form
        className="flex flex-col gap-4"
        onSubmit={(event) => {
          event.preventDefault();
          handleReset();
        }}
      >
        <p className="text-center text-title-lg font-semibold text-neutral-900">Atur ulang password</p>
        <p className="text-center text-body text-neutral-600">Kode dikirim ke {phone}.</p>
        {resetError && <InlineAlert variant="danger">{resetError}</InlineAlert>}
        <TextField label="Kode verifikasi" inputMode="numeric" value={code} onChange={(event) => setCode(event.target.value)} />
        <TextField
          label="Password baru"
          type="password"
          autoComplete="new-password"
          value={password}
          onChange={(event) => setPassword(event.target.value)}
        />
        <Button type="submit" fullWidth loading={submitting}>
          Simpan password baru
        </Button>
        <OtpResendCountdown canResendAt={canResendAt} onResend={requestOtp} />
      </form>
    );
  }

  return (
    <div className="flex flex-col gap-4">
      <p className="text-center text-title-lg font-semibold text-neutral-900">Lupa password</p>
      <TextField
        label="Nomor HP"
        type="tel"
        inputMode="numeric"
        autoComplete="tel-national"
        placeholder="081234567890"
        value={phone}
        onChange={(event) => setPhone(event.target.value)}
        error={phoneError}
      />
      <Button fullWidth onClick={continueFromPhone}>
        Lanjut
      </Button>
    </div>
  );
}
