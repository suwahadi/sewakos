"use client";

import { useEffect, useRef } from "react";
import { Button } from "@/components/ui/button";
import { toast } from "@/lib/toast";
import { formatRemaining, useRemainingMs } from "@/lib/use-remaining";

const WARNING_THRESHOLD = 5 * 60 * 1000;

export interface CountdownProps {
  expiresAt: string;
  activePaymentExpiresAt?: string;
  onExpire: () => void;
}

export function Countdown({ expiresAt, activePaymentExpiresAt, onExpire }: CountdownProps) {
  const paymentRemaining = useRemainingMs(activePaymentExpiresAt ? new Date(activePaymentExpiresAt).getTime() : 0);
  const bookingRemaining = useRemainingMs(new Date(expiresAt).getTime());
  const remaining = paymentRemaining > 0 ? paymentRemaining : bookingRemaining;
  const previousRemaining = useRef(remaining);

  useEffect(() => {
    if (previousRemaining.current > WARNING_THRESHOLD && remaining <= WARNING_THRESHOLD && remaining > 0) {
      toast.warning("Sisa waktu pembayaran 5 menit.");
    }
    previousRemaining.current = remaining;
    if (remaining <= 0) onExpire();
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
