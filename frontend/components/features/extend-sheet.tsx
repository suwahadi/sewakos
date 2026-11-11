"use client";

import { useState } from "react";
import { DurationPicker } from "@/components/features/duration-picker";
import { BottomSheet } from "@/components/ui/bottom-sheet";
import { Button } from "@/components/ui/button";
import { InlineAlert } from "@/components/ui/inline-alert";
import { callBff } from "@/lib/api/browser";
import { showApiError } from "@/lib/api/show-error";
import type { Invoice } from "@/lib/booking";
import { rentalEndDate, type RentType } from "@/lib/period";
import { toast } from "@/lib/toast";

export interface ExtendSheetProps {
  open: boolean;
  onClose: () => void;
  bookingId: number;
  rentType: RentType;
  startDate: string;
  currentDuration: number;
  monthlyOptions: number[];
  navigate: (href: string) => void;
  onExtended: () => void;
}

function defaultDuration(rentType: RentType, monthlyOptions: number[]): number {
  return rentType === "monthly" && monthlyOptions.length > 0 ? Math.min(...monthlyOptions) : 1;
}

export function ExtendSheet({
  open,
  onClose,
  bookingId,
  rentType,
  startDate,
  currentDuration,
  monthlyOptions,
  navigate,
  onExtended,
}: ExtendSheetProps) {
  const [duration, setDuration] = useState(() => defaultDuration(rentType, monthlyOptions));
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string>();

  async function submit() {
    setSubmitting(true);
    setError(undefined);
    const result = await callBff<{ invoice: Invoice }>(`/api/booking/${bookingId}/extend`, {
      method: "POST",
      body: { duration },
    });
    setSubmitting(false);
    if (!result.ok) {
      if (result.httpStatus === 400 || result.httpStatus === 422) {
        setError(result.messages?.duration?.[0] ?? result.meta.message);
        return;
      }
      showApiError(result, navigate, submit);
      return;
    }
    toast.success(result.data.meta.message);
    onExtended();
    onClose();
  }

  return (
    <BottomSheet open={open} onClose={onClose} title="Perpanjang sewa">
      <div className="flex flex-col gap-4 pb-4">
        {error && <InlineAlert variant="danger">{error}</InlineAlert>}
        <DurationPicker
          rentType={rentType}
          value={duration}
          onChange={setDuration}
          monthlyOptions={monthlyOptions}
          endDate={rentalEndDate(startDate, rentType, currentDuration + duration)}
        />
        <Button fullWidth loading={submitting} onClick={submit}>
          Lanjutkan
        </Button>
      </div>
    </BottomSheet>
  );
}
