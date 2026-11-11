"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import { InlineAlert } from "@/components/ui/inline-alert";
import { SegmentedControl } from "@/components/ui/segmented-control";
import { Select } from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import { TextField } from "@/components/ui/text-field";
import { callBff } from "@/lib/api/browser";
import { showApiError } from "@/lib/api/show-error";
import { genderLabel } from "@/lib/labels";
import type { OwnerProperty } from "@/lib/owner";
import { toast } from "@/lib/toast";

type Field = "name" | "area_id" | "address" | "description" | "latitude" | "longitude";
type Gender = "campur" | "putra" | "putri";

const FIELD_ORDER: Field[] = ["name", "area_id", "address", "description", "latitude", "longitude"];
const GENDERS: Gender[] = ["campur", "putra", "putri"];

export interface AreaOption {
  id: number;
  name: string;
}

interface Values {
  name: string;
  area_id: string;
  gender: Gender;
  address: string;
  description: string;
  latitude: string;
  longitude: string;
}

function initialValues(property?: OwnerProperty): Values {
  return {
    name: property?.name ?? "",
    area_id: property ? String(property.area.id) : "",
    gender: (property?.gender as Gender) ?? "campur",
    address: property?.address ?? "",
    description: property?.description ?? "",
    latitude: property ? String(property.latitude) : "",
    longitude: property ? String(property.longitude) : "",
  };
}

function inRange(value: string, limit: number): boolean {
  const n = Number(value);
  return value.trim() !== "" && Number.isFinite(n) && Math.abs(n) <= limit;
}

function validate(values: Values): Partial<Record<Field, string>> {
  const errors: Partial<Record<Field, string>> = {};
  if (!values.name.trim()) errors.name = "Nama kost wajib diisi.";
  if (!values.area_id) errors.area_id = "Pilih kota.";
  if (!values.address.trim()) errors.address = "Alamat wajib diisi.";
  if (!values.description.trim()) errors.description = "Deskripsi wajib diisi.";
  if (!inRange(values.latitude, 90)) errors.latitude = "Latitude harus berupa angka antara -90 dan 90.";
  if (!inRange(values.longitude, 180)) errors.longitude = "Longitude harus berupa angka antara -180 dan 180.";
  return errors;
}

export function OwnerPropertyForm({ areas, property }: { areas: AreaOption[]; property?: OwnerProperty }) {
  const router = useRouter();
  const [values, setValues] = useState(() => initialValues(property));
  const [errors, setErrors] = useState<Partial<Record<Field, string>>>({});
  const [formError, setFormError] = useState<string>();
  const [submitting, setSubmitting] = useState(false);

  function update<K extends keyof Values>(field: K, value: Values[K]) {
    setValues((current) => ({ ...current, [field]: value }));
  }

  function showErrors(next: Partial<Record<Field, string>>) {
    setErrors(next);
    const first = FIELD_ORDER.find((field) => next[field]);
    if (first) document.getElementById(`property-${first}`)?.focus();
  }

  async function submit() {
    setFormError(undefined);
    const clientErrors = validate(values);
    if (Object.keys(clientErrors).length > 0) return showErrors(clientErrors);
    setErrors({});
    setSubmitting(true);

    const body = {
      name: values.name.trim(),
      area_id: Number(values.area_id),
      gender: values.gender,
      address: values.address.trim(),
      description: values.description.trim(),
      latitude: Number(values.latitude),
      longitude: Number(values.longitude),
    };
    const result = property
      ? await callBff<{ property: OwnerProperty }>(`/api/owner/properties/${property.id}`, { method: "PATCH", body })
      : await callBff<{ property: OwnerProperty }>("/api/owner/properties", { method: "POST", body });
    setSubmitting(false);

    if (result.ok) {
      toast.success(result.data.meta.message);
      if (property) router.refresh();
      else router.push(`/owner/properti/${result.data.property.id}`);
      return;
    }
    if (result.httpStatus === 422) {
      return showErrors(Object.fromEntries(FIELD_ORDER.map((field) => [field, result.messages?.[field]?.[0]])));
    }
    if (result.httpStatus === 400) return setFormError(result.meta.message);
    showApiError(result, (href) => router.push(href), submit);
  }

  return (
    <div className="flex flex-col gap-4">
      {formError && <InlineAlert variant="danger">{formError}</InlineAlert>}
      <TextField id="property-name" label="Nama kost" maxLength={255} value={values.name} onChange={(event) => update("name", event.target.value)} error={errors.name} />
      <Select id="property-area_id" label="Kota" value={values.area_id} onChange={(event) => update("area_id", event.target.value)} error={errors.area_id}>
        <option value="">Pilih kota</option>
        {areas.map((area) => (
          <option key={area.id} value={area.id}>
            {area.name}
          </option>
        ))}
      </Select>
      <div className="flex flex-col gap-1.5">
        <p className="text-label font-medium text-neutral-700">Tipe kost</p>
        <SegmentedControl
          aria-label="Tipe kost"
          value={values.gender}
          onChange={(gender) => update("gender", gender)}
          options={GENDERS.map((gender) => ({ value: gender, label: genderLabel(gender) }))}
        />
      </div>
      <TextField id="property-address" label="Alamat" maxLength={500} value={values.address} onChange={(event) => update("address", event.target.value)} error={errors.address} />
      <Textarea
        id="property-description"
        label="Deskripsi"
        maxLength={5000}
        value={values.description}
        onChange={(event) => update("description", event.target.value)}
        error={errors.description}
      />
      <div className="grid grid-cols-2 gap-3">
        <TextField
          id="property-latitude"
          label="Latitude"
          inputMode="decimal"
          placeholder="-6.38"
          value={values.latitude}
          onChange={(event) => update("latitude", event.target.value)}
          error={errors.latitude}
        />
        <TextField
          id="property-longitude"
          label="Longitude"
          inputMode="decimal"
          placeholder="106.81"
          value={values.longitude}
          onChange={(event) => update("longitude", event.target.value)}
          error={errors.longitude}
        />
      </div>
      <Button fullWidth loading={submitting} onClick={submit}>
        {property ? "Simpan perubahan" : "Simpan properti"}
      </Button>
    </div>
  );
}
