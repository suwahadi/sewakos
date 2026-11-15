"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";
import { OtpChannelPicker, OtpSentNotice } from "@/components/features/otp-channel-picker";
import { Button } from "@/components/ui/button";
import { InlineAlert } from "@/components/ui/inline-alert";
import { OtpResendCountdown } from "@/components/ui/otp-input";
import { TextField } from "@/components/ui/text-field";
import { callBff } from "@/lib/api/browser";
import { toast } from "@/lib/toast";
import {
  OTP_CHANNEL_LABEL,
  OTP_CHANNEL_VIA,
  otherOtpChannel,
  useOtpChannels,
  type OtpChannel,
} from "@/lib/otp-channels";
import { useOtpRequest, type OtpRequestResponse } from "@/lib/use-otp-request";

const PHONE_PATTERN = /^08\d{7,12}$/;

type Step = "phone" | "channel" | "reset";

export default function LupaPasswordPage() {
  const router = useRouter();
  const [step, setStep] = useState<Step>("phone");
  const [phone, setPhone] = useState("");
  const [phoneError, setPhoneError] = useState<string | undefined>();
  const channels = useOtpChannels();
  const [channel, setChannel] = useState<OtpChannel>("whatsapp");
  const otp = useOtpRequest(
    () =>
      callBff<OtpRequestResponse>("/api/auth/otp/request", {
        method: "POST",
        body: { for: "reset_password", via: OTP_CHANNEL_VIA[channel], destination: phone },
      }),
    {
      onChannelUnavailable: () => {
        const next = otherOtpChannel(channel, channels);
        if (next) setChannel(next);
        setStep("channel");
      },
    },
  );

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
    if (await otp.request()) setStep("reset");
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
        <p className="text-center text-title-lg font-semibold text-neutral-900">Kirim kode verifikasi</p>
        <p className="text-center text-body text-neutral-600">
          Kode verifikasi akan dikirim lewat {OTP_CHANNEL_LABEL[channel]} ke {phone}.
        </p>
        <OtpChannelPicker channels={channels} value={channel} onChange={setChannel} />
        <Button fullWidth onClick={requestOtp} loading={otp.sending}>
          Kirim kode lewat {OTP_CHANNEL_LABEL[channel]}
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
        <OtpSentNotice channel={channel} phone={phone} className="text-center" />
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
        <OtpResendCountdown
          canResendAt={otp.canResendAt}
          onResend={otp.request}
          sending={otp.sending}
          resendsLeft={otp.resendsLeft}
          blockedMessage={otp.blockedMessage}
        />
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
