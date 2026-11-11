"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";
import { BottomSheet } from "@/components/ui/bottom-sheet";
import { Button } from "@/components/ui/button";
import { Chip } from "@/components/ui/chip";
import { InlineAlert } from "@/components/ui/inline-alert";
import { TextField } from "@/components/ui/text-field";
import { callBff } from "@/lib/api/browser";
import { showApiError } from "@/lib/api/show-error";
import { digitsOnly, formatThousands } from "@/lib/format";
import type { OwnerRoom } from "@/lib/owner";
import { toast } from "@/lib/toast";

type Field = "name" | "price_monthly" | "price_weekly" | "price_daily" | "available" | "capacity" | "facilities";

const DURATION_CHOICES = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12];
const MAX_DURATIONS = 6;

interface Values {
  name: string;
  price_monthly: string;
  price_weekly: string;
  price_daily: string;
  available: string;
  capacity: string;
  facilities: string;
  durations: number[];
}

function initialValues(room?: OwnerRoom): Values {
  return {
    name: room?.name ?? "",
    price_monthly: room ? String(room.price_monthly) : "",
    price_weekly: room?.price_weekly ? String(room.price_weekly) : "",
    price_daily: room?.price_daily ? String(room.price_daily) : "",
    available: room ? String(room.available) : "1",
    capacity: room ? String(room.capacity) : "1",
    facilities: room?.facilities.join(", ") ?? "",
    durations: room?.monthly_duration_options ?? [1],
  };
}

function validate(values: Values): Partial<Record<Field, string>> {
  const errors: Partial<Record<Field, string>> = {};
  const available = Number(values.available);
  const capacity = Number(values.capacity);
  if (!values.name.trim()) errors.name = "Nama kamar wajib diisi.";
  if (!(Number(values.price_monthly) >= 1)) errors.price_monthly = "Harga per bulan wajib diisi.";
  if (values.available === "" || available > 255) errors.available = "Jumlah kamar kosong harus 0–255.";
  if (values.capacity === "" || capacity < 1 || capacity > 20) errors.capacity = "Kapasitas harus 1–20 orang.";
  const facilities = splitFacilities(values.facilities);
  if (facilities.length > 30) errors.facilities = "Maksimal 30 fasilitas.";
  else if (facilities.some((facility) => facility.length > 50)) errors.facilities = "Setiap fasilitas maksimal 50 karakter.";
  return errors;
}

function splitFacilities(value: string): string[] {
  return value
    .split(",")
    .map((facility) => facility.trim())
    .filter(Boolean);
}

function optionalPrice(value: string): number | null {
  return value === "" ? null : Number(value);
}

export interface OwnerRoomSheetProps {
  open: boolean;
  onClose: () => void;
  propertyId: number;
  room?: OwnerRoom;
}

export function OwnerRoomSheet({ open, onClose, propertyId, room }: OwnerRoomSheetProps) {
  return (
    <BottomSheet open={open} onClose={onClose} title={room ? "Ubah kamar" : "Tambah kamar"}>
      {open && <RoomForm key={room?.id ?? "new"} propertyId={propertyId} room={room} onSaved={onClose} />}
    </BottomSheet>
  );
}

function RoomForm({ propertyId, room, onSaved }: { propertyId: number; room?: OwnerRoom; onSaved: () => void }) {
  const router = useRouter();
  const [values, setValues] = useState(() => initialValues(room));
  const [errors, setErrors] = useState<Partial<Record<Field, string>>>({});
  const [formError, setFormError] = useState<string>();
  const [submitting, setSubmitting] = useState(false);

  function update<K extends keyof Values>(field: K, value: Values[K]) {
    setValues((current) => ({ ...current, [field]: value }));
  }

  function toggleDuration(months: number) {
    setValues((current) => ({
      ...current,
      durations: current.durations.includes(months)
        ? current.durations.filter((d) => d !== months)
        : [...current.durations, months].sort((a, b) => a - b),
    }));
  }

  async function submit() {
    setFormError(undefined);
    const clientErrors = validate(values);
    setErrors(clientErrors);
    if (Object.keys(clientErrors).length > 0) return;
    setSubmitting(true);

    const body = {
      name: values.name.trim(),
      price_monthly: Number(values.price_monthly),
      price_weekly: optionalPrice(values.price_weekly),
      price_daily: optionalPrice(values.price_daily),
      available: Number(values.available),
      capacity: Number(values.capacity),
      facilities: splitFacilities(values.facilities),
      monthly_duration_options: values.durations.length > 0 ? values.durations : [1],
    };
    const result = room
      ? await callBff(`/api/owner/rooms/${room.id}`, { method: "PATCH", body })
      : await callBff(`/api/owner/properties/${propertyId}/rooms`, { method: "POST", body });
    setSubmitting(false);

    if (result.ok) {
      toast.success(result.data.meta.message);
      onSaved();
      router.refresh();
      return;
    }
    if (result.httpStatus === 422) {
      const messages = result.messages ?? {};
      const facilitiesMessage = Object.entries(messages).find(([key]) => key.startsWith("facilities"))?.[1]?.[0];
      setErrors({
        name: messages.name?.[0], price_monthly: messages.price_monthly?.[0], price_weekly: messages.price_weekly?.[0],
        price_daily: messages.price_daily?.[0], available: messages.available?.[0], capacity: messages.capacity?.[0],
        facilities: facilitiesMessage,
      });
      return;
    }
    if (result.httpStatus === 400) return setFormError(result.meta.message);
    showApiError(result, (href) => router.push(href), submit);
  }

  const priceField = (field: "price_monthly" | "price_weekly" | "price_daily", label: string, helper?: string) => (
    <TextField
      label={label}
      prefix="Rp"
      inputMode="numeric"
      helper={helper}
      value={formatThousands(values[field])}
      onChange={(event) => update(field, digitsOnly(event.target.value))}
      error={errors[field]}
    />
  );

  return (
    <div className="flex flex-col gap-4 pb-4">
      {formError && <InlineAlert variant="danger">{formError}</InlineAlert>}
      <TextField label="Nama kamar" placeholder="Kamar Tipe A" maxLength={255} value={values.name} onChange={(event) => update("name", event.target.value)} error={errors.name} />
      {priceField("price_monthly", "Harga per bulan")}
      <div className="grid grid-cols-2 gap-3">
        {priceField("price_weekly", "Per minggu", "Opsional")}
        {priceField("price_daily", "Per hari", "Opsional")}
      </div>
      <div className="grid grid-cols-2 gap-3">
        <TextField
          label="Kamar kosong"
          inputMode="numeric"
          value={values.available}
          onChange={(event) => update("available", digitsOnly(event.target.value))}
          error={errors.available}
        />
        <TextField
          label="Kapasitas (orang)"
          inputMode="numeric"
          value={values.capacity}
          onChange={(event) => update("capacity", digitsOnly(event.target.value))}
          error={errors.capacity}
        />
      </div>
      <TextField
        label="Fasilitas"
        placeholder="WiFi, AC, Kasur"
        helper="Pisahkan dengan koma"
        value={values.facilities}
        onChange={(event) => update("facilities", event.target.value)}
        error={errors.facilities}
      />
      <div role="group" aria-labelledby="room-durations-label" className="flex flex-col gap-1.5">
        <p id="room-durations-label" className="text-label font-medium text-neutral-700">
          Durasi sewa bulanan
        </p>
        <div className="flex flex-wrap gap-2">
          {DURATION_CHOICES.map((months) => {
            const active = values.durations.includes(months);
            return (
              <Chip
                key={months}
                active={active}
                disabled={!active && values.durations.length >= MAX_DURATIONS}
                onClick={() => toggleDuration(months)}
              >
                {months} bulan
              </Chip>
            );
          })}
        </div>
        <p className="text-caption text-neutral-500">Pilih hingga {MAX_DURATIONS} pilihan.</p>
      </div>
      <Button fullWidth loading={submitting} onClick={submit}>
        Simpan kamar
      </Button>
    </div>
  );
}
