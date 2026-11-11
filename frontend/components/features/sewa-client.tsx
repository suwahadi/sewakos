"use client";

import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { DurationPicker } from "@/components/features/duration-picker";
import { DateField } from "@/components/ui/date-field";
import { InlineAlert } from "@/components/ui/inline-alert";
import { PriceBreakdown } from "@/components/ui/price-breakdown";
import { SegmentedControl } from "@/components/ui/segmented-control";
import { StickyActionBar } from "@/components/ui/sticky-action-bar";
import { callBff } from "@/lib/api/browser";
import { showApiError } from "@/lib/api/show-error";
import { rentTypeLabel, rentTypeUnit } from "@/lib/labels";
import { rentalEndDate, todayWIB, type RentType } from "@/lib/period";
import { toast } from "@/lib/toast";

const START_DATE_FIELD_ID = "start-date";
const PRICE_REFRESH_DELAY_MS = 250;

export interface DraftPriceItem {
  type: RentType;
  price: number;
  price_label: string;
  deposit_label: string;
  admin_fee: number;
  admin_fee_label: string;
  flash_sale_discount: number;
  flash_sale_discount_label: string;
  total_off_first_payment_label: string;
  remaining_payment_label: string;
}

interface SubmitResponse {
  booking: { id: number };
}

export interface SewaClientProps {
  roomId: number;
  isFlashSale: boolean;
  monthlyDurationOptions: number[];
  initialPrices: DraftPriceItem[];
}

function defaultDuration(rentType: RentType, monthlyOptions: number[]): number {
  return rentType === "monthly" && monthlyOptions.length > 0 ? Math.min(...monthlyOptions) : 1;
}

export function SewaClient({ roomId, isFlashSale, monthlyDurationOptions, initialPrices }: SewaClientProps) {
  const router = useRouter();
  const [prices, setPrices] = useState(initialPrices);
  const [rentType, setRentType] = useState<RentType>(initialPrices[0]?.type ?? "monthly");
  const [duration, setDuration] = useState(() => defaultDuration(rentType, monthlyDurationOptions));
  const [startDate, setStartDate] = useState(todayWIB);
  const [dateError, setDateError] = useState<string>();
  const [formError, setFormError] = useState<string>();
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    if (rentType === "monthly") return;
    let stale = false;
    const timer = setTimeout(async () => {
      const result = await callBff<{ data: DraftPriceItem[] }>(
        `/api/booking/draft-price/${roomId}?is_flash_sale=${isFlashSale}&duration=${duration}`,
      );
      if (stale) return;
      if (result.ok) setPrices(result.data.data);
      else showApiError(result, (href) => router.push(href));
    }, PRICE_REFRESH_DELAY_MS);
    return () => {
      stale = true;
      clearTimeout(timer);
    };
  }, [rentType, duration, roomId, isFlashSale, router]);

  function changeRentType(next: RentType) {
    setRentType(next);
    setDuration(defaultDuration(next, monthlyDurationOptions));
  }

  function showDateError(message: string) {
    setDateError(message);
    document.getElementById(START_DATE_FIELD_ID)?.focus();
  }

  async function submit() {
    setFormError(undefined);
    if (!startDate) return showDateError("Pilih tanggal mulai sewa.");
    if (startDate < todayWIB()) return showDateError("Tanggal mulai tidak boleh sebelum hari ini.");
    setDateError(undefined);
    setSubmitting(true);

    const result = await callBff<SubmitResponse>("/api/booking/submit", {
      method: "POST",
      body: { room_id: roomId, rent_type: rentType, start_date: startDate, duration },
    });
    if (result.ok) {
      toast.success(result.data.meta.message);
      router.push(`/booking/${result.data.booking.id}`);
      return;
    }

    setSubmitting(false);
    if (result.httpStatus === 422) {
      const { start_date: startDateMessages, ...others } = result.messages ?? {};
      if (startDateMessages?.[0]) showDateError(startDateMessages[0]);
      const otherMessage = Object.values(others)[0]?.[0];
      if (otherMessage || !startDateMessages) setFormError(otherMessage ?? result.meta.message);
    } else if (result.httpStatus === 400) {
      setFormError(result.meta.message);
    } else {
      showApiError(result, (href) => router.push(href), submit);
    }
  }

  const selected = prices.find((item) => item.type === rentType);
  const periodic = rentType !== "monthly";

  return (
    <div className="flex flex-col gap-6 px-4 pb-24">
      {formError && <InlineAlert variant="danger">{formError}</InlineAlert>}

      <div className="flex flex-col gap-1.5">
        <p className="text-label font-medium text-neutral-700">Tipe sewa</p>
        <SegmentedControl
          aria-label="Tipe sewa"
          value={rentType}
          onChange={changeRentType}
          options={prices.map((item) => ({ value: item.type, label: rentTypeLabel(item.type) }))}
        />
      </div>

      <DurationPicker
        rentType={rentType}
        value={duration}
        onChange={setDuration}
        monthlyOptions={monthlyDurationOptions}
        endDate={startDate ? rentalEndDate(startDate, rentType, duration) : undefined}
      />

      <DateField
        id={START_DATE_FIELD_ID}
        label="Tanggal mulai"
        value={startDate}
        onChange={setStartDate}
        error={dateError}
        required
      />

      {selected && (
        <>
          <PriceBreakdown
            priceLabel={periodic ? `${selected.price_label} × ${duration} ${rentTypeUnit(rentType)}` : selected.price_label}
            depositLabel={selected.deposit_label}
            adminFeeLabel={selected.admin_fee > 0 ? selected.admin_fee_label : undefined}
            flashSaleDiscountLabel={selected.flash_sale_discount > 0 ? selected.flash_sale_discount_label : undefined}
            remainingPaymentLabel={selected.flash_sale_discount > 0 ? selected.remaining_payment_label : undefined}
            totalLabel={selected.total_off_first_payment_label}
            nextMonthLabel={
              !periodic && duration > 1 ? `Bulan berikutnya ditagih ${selected.price_label} per bulan` : undefined
            }
          />
          <StickyActionBar
            leadingLabel="Total"
            price={selected.total_off_first_payment_label}
            actionLabel="Ajukan sewa"
            onAction={submit}
            loading={submitting}
          />
        </>
      )}
    </div>
  );
}
