"use client";

import { useRef, useState } from "react";
import type { ApiResult } from "@/lib/api/envelope";
import { toast } from "@/lib/toast";

export const MAX_OTP_RESENDS = 3;
export const OTP_RESEND_LIMIT_MESSAGE = "Batas kirim ulang kode tercapai. Silakan coba lagi nanti.";
const PROVIDER_FAILURE_WAIT_SECONDS = 60;

export interface OtpRequestResponse {
  verification_code: { can_resend_at: number };
}

export interface OtpRequestState {
  request: () => Promise<boolean>;
  sending: boolean;
  canResendAt: number;
  resendsLeft: number;
  blockedMessage: string | null;
}

export interface OtpRequestOptions {
  onChannelUnavailable?: (message: string) => void;
}

export function useOtpRequest(
  send: () => Promise<ApiResult<OtpRequestResponse>>,
  { onChannelUnavailable }: OtpRequestOptions = {},
): OtpRequestState {
  const inFlight = useRef(false);
  const [sending, setSending] = useState(false);
  const [canResendAt, setCanResendAt] = useState(0);
  const [sentCount, setSentCount] = useState(0);
  const [blockedMessage, setBlockedMessage] = useState<string | null>(null);

  const resendsLeft = Math.max(0, MAX_OTP_RESENDS - Math.max(0, sentCount - 1));

  async function request(): Promise<boolean> {
    if (inFlight.current) return false;
    if (blockedMessage !== null) {
      toast.error(blockedMessage);
      return false;
    }
    if (sentCount > 0 && resendsLeft === 0) {
      toast.error(OTP_RESEND_LIMIT_MESSAGE);
      return false;
    }
    if (Math.floor(Date.now() / 1000) < canResendAt) {
      toast.error("Tunggu sebentar sebelum meminta kode baru.");
      return false;
    }

    inFlight.current = true;
    setSending(true);
    const result = await send();
    inFlight.current = false;
    setSending(false);

    if (!result.ok) {
      const channelMessage = result.messages?.via?.[0];
      if (channelMessage) {
        onChannelUnavailable?.(channelMessage);
        toast.error(channelMessage);
        return false;
      }
      if (result.httpStatus === 429) {
        setBlockedMessage(result.meta.message);
      } else if (result.httpStatus === 503) {
        setCanResendAt(Math.floor(Date.now() / 1000) + PROVIDER_FAILURE_WAIT_SECONDS);
      }
      toast.error(result.meta.message);
      return false;
    }

    setCanResendAt(result.data.verification_code.can_resend_at);
    setSentCount((count) => count + 1);
    return true;
  }

  return { request, sending, canResendAt, resendsLeft, blockedMessage };
}
