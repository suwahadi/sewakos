"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import { ButtonPair } from "@/components/ui/button-pair";
import { Checkbox } from "@/components/ui/checkbox";
import { DataTable } from "@/components/ui/data-table";
import { ConfirmDialog, Dialog } from "@/components/ui/dialog";
import { LoadMore } from "@/components/ui/load-more";
import { Select } from "@/components/ui/select";
import { TextField } from "@/components/ui/text-field";
import { areaTypeLabel, type AdminArea } from "@/lib/admin";
import { callBff } from "@/lib/api/browser";
import { showApiError } from "@/lib/api/show-error";
import { toast } from "@/lib/toast";
import { useAdminList } from "@/lib/use-admin-list";

const AREA_TYPES = ["city", "campus", "mrt"] as const;
type AreaField = "name" | "slug" | "latitude" | "longitude" | "place_id";

interface AreaForm {
  name: string;
  slug: string;
  type: AdminArea["type"];
  is_top: boolean;
  latitude: string;
  longitude: string;
  place_id: string;
}

function formFromArea(area?: AdminArea): AreaForm {
  return {
    name: area?.name ?? "",
    slug: area?.slug ?? "",
    type: area?.type ?? "city",
    is_top: area?.is_top ?? false,
    latitude: area ? String(area.latitude) : "",
    longitude: area ? String(area.longitude) : "",
    place_id: area?.place_id ?? "",
  };
}

export function AdminAreaList({ initial }: { initial: AdminArea[] }) {
  const router = useRouter();
  const list = useAdminList<AdminArea>("areas", "areas", initial);
  const [editing, setEditing] = useState<{ area?: AdminArea } | null>(null);
  const [deleting, setDeleting] = useState<AdminArea | null>(null);
  const [busy, setBusy] = useState(false);

  async function remove(area: AdminArea) {
    setBusy(true);
    const result = await callBff(`/api/admin/areas/${area.id}`, { method: "DELETE" });
    setBusy(false);
    setDeleting(null);
    if (!result.ok) return showApiError(result, (href) => router.push(href), () => remove(area));
    toast.success(result.data.meta.message);
    list.reload();
  }

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h1 className="text-title-lg font-semibold text-neutral-900">Area</h1>
        <Button onClick={() => setEditing({})}>Tambah area</Button>
      </div>
      <div className="max-w-xs">
        <Select label="Tipe" value={list.query.type ?? ""} onChange={(event) => list.filter({ type: event.target.value })}>
          <option value="">Semua tipe</option>
          {AREA_TYPES.map((type) => (
            <option key={type} value={type}>
              {areaTypeLabel(type)}
            </option>
          ))}
        </Select>
      </div>
      <DataTable
        caption="Daftar area"
        rows={list.rows}
        rowKey={(area) => area.id}
        columns={[
          { key: "name", header: "Nama", render: (area) => area.name },
          { key: "slug", header: "Slug", render: (area) => area.slug },
          { key: "type", header: "Tipe", render: (area) => areaTypeLabel(area.type) },
          { key: "top", header: "Populer", render: (area) => (area.is_top ? "Ya" : "Tidak") },
          {
            key: "action",
            header: "Aksi",
            render: (area) => (
              <div className="w-48 max-w-full">
                <ButtonPair>
                  <Button size="sm" variant="danger" onClick={() => setDeleting(area)}>
                    Hapus
                  </Button>
                  <Button size="sm" variant="neutral" onClick={() => setEditing({ area })}>
                    Ubah
                  </Button>
                </ButtonPair>
              </div>
            ),
          },
        ]}
      />
      {list.hasMore && <LoadMore loading={list.loading} onClick={list.loadMore} />}
      {editing && (
        <AreaDialog
          area={editing.area}
          onClose={() => setEditing(null)}
          onSaved={() => {
            setEditing(null);
            list.reload();
          }}
        />
      )}
      {deleting && (
        <ConfirmDialog
          open
          onClose={() => setDeleting(null)}
          title={`Hapus ${deleting.name}?`}
          description="Area yang masih dipakai properti tidak bisa dihapus."
          confirmLabel="Hapus"
          loading={busy}
          onConfirm={() => remove(deleting)}
        />
      )}
    </div>
  );
}

function AreaDialog({ area, onClose, onSaved }: { area?: AdminArea; onClose: () => void; onSaved: () => void }) {
  const router = useRouter();
  const [form, setForm] = useState(() => formFromArea(area));
  const [errors, setErrors] = useState<Partial<Record<AreaField, string>>>({});
  const [saving, setSaving] = useState(false);

  function update<K extends keyof AreaForm>(field: K, value: AreaForm[K]) {
    setForm((current) => ({ ...current, [field]: value }));
  }

  async function save() {
    const body = {
      name: form.name.trim(),
      slug: form.slug.trim(),
      type: form.type,
      is_top: form.is_top,
      latitude: Number(form.latitude),
      longitude: Number(form.longitude),
      place_id: form.place_id.trim() || null,
    };
    setSaving(true);
    const result = area
      ? await callBff(`/api/admin/areas/${area.id}`, { method: "PATCH", body })
      : await callBff("/api/admin/areas", { method: "POST", body });
    setSaving(false);
    if (result.ok) {
      toast.success(result.data.meta.message);
      onSaved();
      return;
    }
    if (result.httpStatus === 422) {
      const messages = result.messages ?? {};
      setErrors({
        name: messages.name?.[0], slug: messages.slug?.[0], latitude: messages.latitude?.[0],
        longitude: messages.longitude?.[0], place_id: messages.place_id?.[0],
      });
      return;
    }
    showApiError(result, (href) => router.push(href), save);
  }

  return (
    <Dialog open onClose={onClose} title={area ? "Ubah area" : "Tambah area"}>
      <div className="flex flex-col gap-4 pt-2">
        <TextField label="Nama" maxLength={255} value={form.name} onChange={(event) => update("name", event.target.value)} error={errors.name} />
        <TextField label="Slug" placeholder="kota-depok" value={form.slug} onChange={(event) => update("slug", event.target.value)} error={errors.slug} />
        <Select label="Tipe" value={form.type} onChange={(event) => update("type", event.target.value as AdminArea["type"])}>
          {AREA_TYPES.map((type) => (
            <option key={type} value={type}>
              {areaTypeLabel(type)}
            </option>
          ))}
        </Select>
        <div className="grid grid-cols-2 gap-3">
          <TextField label="Latitude" inputMode="decimal" value={form.latitude} onChange={(event) => update("latitude", event.target.value)} error={errors.latitude} />
          <TextField label="Longitude" inputMode="decimal" value={form.longitude} onChange={(event) => update("longitude", event.target.value)} error={errors.longitude} />
        </div>
        <TextField label="Place ID (opsional)" maxLength={255} value={form.place_id} onChange={(event) => update("place_id", event.target.value)} error={errors.place_id} />
        <Checkbox label="Tampilkan sebagai area populer" checked={form.is_top} onChange={(event) => update("is_top", event.target.checked)} />
        <ButtonPair>
          <Button variant="neutral" onClick={onClose}>
            Batal
          </Button>
          <Button loading={saving} onClick={save}>
            Simpan area
          </Button>
        </ButtonPair>
      </div>
    </Dialog>
  );
}
