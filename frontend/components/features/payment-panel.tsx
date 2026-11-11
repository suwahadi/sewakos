"use client";

import { useEffect, useRef, useState } from "react";
import { Button } from "@/components/ui/button";
import { toast } from "@/lib/toast";

const SECOND = 1000;
const MINUTE = 60 * SECOND;
const HOUR = 60 * MINUTE;
const WARNING_THRESHOLD = 5 * MINUTE;

export function formatRemaining(ms: number): string {
  if (ms >= HOUR) return `${Math.floor(ms / HOUR)} jam ${Math.floor((ms % HOUR) / MINUTE)} menit`;
  const seconds = Math.max(0, Math.floor(ms / SECOND));
  return `${String(Math.floor(seconds / 60)).padStart(2, "0")}:${String(seconds % 60).padStart(2, "0")}`;
}

function nextTickDelay(remaining: number): number {
  const unit = remaining >= HOUR ? MINUTE : SECOND;
  return remaining % unit || unit;
}

export interface CountdownProps {
  expiresAt: string;
  activePaymentExpiresAt?: string;
  onExpire: () => void;
}

export function Countdown({ expiresAt, activePaymentExpiresAt, onExpire }: CountdownProps) {
  const [now, setNow] = useState(() => Date.now());
  const paymentDeadline = activePaymentExpiresAt ? new Date(activePaymentExpiresAt).getTime() : 0;
  const deadline = paymentDeadline > now ? paymentDeadline : new Date(expiresAt).getTime();
  const remaining = deadline - now;
  const previousRemaining = useRef(remaining);

  useEffect(() => {
    if (previousRemaining.current > WARNING_THRESHOLD && remaining <= WARNING_THRESHOLD && remaining > 0) {
      toast.warning("Sisa waktu pembayaran 5 menit.");
    }
    previousRemaining.current = remaining;
    if (remaining <= 0) {
      onExpire();
      return;
    }
    const timer = setTimeout(() => setNow(Date.now()), nextTickDelay(remaining));
    return () => clearTimeout(timer);
  }, [remaining, onExpire]);

  return (
    <p className="text-body text-neutral-700">
      {remaining > 0 ? (
        <>
          Selesaikan pembayaran dalam{" "}
          <span className="font-semibold tabular-nums text-neutral-900" suppressHydrationWarning>
            {formatRemaining(remaining)}
          </span>
        </>
      ) : (
        "Batas waktu pembayaran sudah lewat."
      )}
    </p>
  );
}

export interface PaymentPanelProps extends CountdownProps {
  paying: boolean;
  onPay: () => void;
}

export function PaymentPanel({ paying, onPay, ...countdown }: PaymentPanelProps) {
  return (
    <section aria-label="Pembayaran" className="flex flex-col gap-3 rounded-lg border border-neutral-200 p-4">
      <Countdown {...countdown} />
      <Button size="lg" fullWidth loading={paying} onClick={onPay}>
        Bayar sekarang
      </Button>
    </section>
  );
}
