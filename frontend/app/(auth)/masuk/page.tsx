"use client";

import { useRouter, useSearchParams } from "next/navigation";
import { Suspense, useState } from "react";
import { OtpChannelPicker, OtpSentNotice } from "@/components/features/otp-channel-picker";
import { Button } from "@/components/ui/button";
import { InlineAlert } from "@/components/ui/inline-alert";
import { OtpInput, OtpResendCountdown } from "@/components/ui/otp-input";
import { SegmentedControl } from "@/components/ui/segmented-control";
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

type Method = "password" | "otp";
type OtpStep = "phone" | "channel" | "code";

export default function MasukPage() {
  return (
    <Suspense>
      <MasukForm />
    </Suspense>
  );
}

function MasukForm() {
  const router = useRouter();
  const next = useSearchParams().get("next");
  const destination = (role: string) => next ?? (role === "admin" ? "/admin" : role === "owner" ? "/owner" : "/");
  const [method, setMethod] = useState<Method>("password");

  const [phone, setPhone] = useState("");
  const [password, setPassword] = useState("");
  const [loginError, setLoginError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const [otpStep, setOtpStep] = useState<OtpStep>("phone");
  const [phoneError, setPhoneError] = useState<string | undefined>();
  const [otpError, setOtpError] = useState<string | undefined>();
  const channels = useOtpChannels();
  const [channel, setChannel] = useState<OtpChannel>("whatsapp");
  const otp = useOtpRequest(
    () =>
      callBff<OtpRequestResponse>("/api/auth/otp/request", {
        method: "POST",
        body: { for: "login", via: OTP_CHANNEL_VIA[channel], destination: phone },
      }),
    {
      onChannelUnavailable: () => {
        const next = otherOtpChannel(channel, channels);
        if (next) setChannel(next);
        setOtpStep("channel");
      },
    },
  );

  async function handlePasswordLogin() {
    if (!PHONE_PATTERN.test(phone)) {
      setLoginError("Nomor HP harus diawali 08 dan berisi 9–14 digit.");
      return;
    }
    setSubmitting(true);
    setLoginError(null);
    const result = await callBff<{ user: { role: string } }>("/api/auth/login", {
      method: "POST",
      body: { phone_number: phone, password },
    });
    setSubmitting(false);

    if (!result.ok) {
      setLoginError(result.meta.message);
      return;
    }
    toast.success("Login berhasil.");
    router.refresh();
    router.push(destination(result.data.user.role));
  }

  async function sendLoginOtp() {
    if (!PHONE_PATTERN.test(phone)) {
      setPhoneError("Nomor HP harus diawali 08 dan berisi 9–14 digit.");
      return;
    }
    setPhoneError(undefined);
    setOtpStep("channel");
  }

  async function requestOtp() {
    if (await otp.request()) setOtpStep("code");
  }

  async function checkOtp(code: string) {
    setOtpError(undefined);
    const result = await callBff<{ user: { role: string } }>("/api/auth/otp/check", {
      method: "POST",
      body: { for: "login", destination: phone, code },
    });
    if (!result.ok) {
      setOtpError(result.meta.message);
      return;
    }
    toast.success("Login berhasil.");
    router.refresh();
    router.push(destination(result.data.user.role));
  }

  return (
    <div className="flex flex-col gap-6">
      <p className="text-center text-title-lg font-semibold text-neutral-900">Masuk</p>
      <SegmentedControl
        aria-label="Metode masuk"
        value={method}
        onChange={(value) => {
          setMethod(value);
          setOtpStep("phone");
        }}
        options={[
          { value: "password", label: "Password" },
          { value: "otp", label: "Kode OTP" },
        ]}
      />

      {method === "password" ? (
        <form
          className="flex flex-col gap-4"
          onSubmit={(event) => {
            event.preventDefault();
            handlePasswordLogin();
          }}
        >
          {loginError && <InlineAlert variant="danger">{loginError}</InlineAlert>}
          <TextField
            label="Nomor HP"
            type="tel"
            inputMode="numeric"
            autoComplete="tel-national"
            placeholder="081234567890"
            value={phone}
            onChange={(event) => setPhone(event.target.value)}
          />
          <TextField
            label="Password"
            type="password"
            autoComplete="current-password"
            value={password}
            onChange={(event) => setPassword(event.target.value)}
          />
          <Button type="submit" fullWidth loading={submitting}>
            Masuk
          </Button>
          <a href="/lupa-password" className="text-center text-body text-brand-700 underline">
            Lupa password?
          </a>
        </form>
      ) : otpStep === "phone" ? (
        <div className="flex flex-col gap-4">
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
          <Button fullWidth onClick={sendLoginOtp}>
            Lanjut
          </Button>
        </div>
      ) : otpStep === "channel" ? (
        <div className="flex flex-col gap-4">
          <p className="text-body text-neutral-600">
            Kode verifikasi akan dikirim lewat {OTP_CHANNEL_LABEL[channel]} ke {phone}.
          </p>
          <OtpChannelPicker channels={channels} value={channel} onChange={setChannel} />
          <Button fullWidth onClick={requestOtp} loading={otp.sending}>
            Kirim kode lewat {OTP_CHANNEL_LABEL[channel]}
          </Button>
        </div>
      ) : (
        <div className="flex flex-col items-center gap-4">
          <OtpSentNotice channel={channel} phone={phone} className="text-center" />
          <OtpInput onComplete={checkOtp} error={otpError} />
          <OtpResendCountdown
            canResendAt={otp.canResendAt}
            onResend={otp.request}
            sending={otp.sending}
            resendsLeft={otp.resendsLeft}
            blockedMessage={otp.blockedMessage}
          />
        </div>
      )}

      <p className="text-center text-body text-neutral-600">
        Belum punya akun?{" "}
        <a href="/daftar" className="text-brand-700 underline">
          Daftar
        </a>
      </p>
    </div>
  );
}
