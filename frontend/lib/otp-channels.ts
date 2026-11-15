"use client";

import { useEffect, useState } from "react";
import { callBff } from "@/lib/api/browser";

export type OtpChannel = "whatsapp" | "sms";

export const OTP_CHANNEL_LABEL: Record<OtpChannel, string> = { whatsapp: "WhatsApp", sms: "SMS" };
export const OTP_CHANNEL_VIA: Record<OtpChannel, "0" | "1"> = { whatsapp: "1", sms: "0" };

const FALLBACK_CHANNELS: OtpChannel[] = ["whatsapp"];

function isOtpChannel(value: unknown): value is OtpChannel {
  return value === "whatsapp" || value === "sms";
}

export function useOtpChannels(): OtpChannel[] {
  const [channels, setChannels] = useState<OtpChannel[]>(FALLBACK_CHANNELS);

  useEffect(() => {
    let active = true;
    callBff<{ channels: unknown[] }>("/api/auth/otp/channels").then((result) => {
      if (!active || !result.ok) return;
      const available = result.data.channels.filter(isOtpChannel);
      setChannels(available.length > 0 ? available : FALLBACK_CHANNELS);
    });
    return () => {
      active = false;
    };
  }, []);

  return channels;
}
