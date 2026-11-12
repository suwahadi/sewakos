"use client";

import { useRouter } from "next/navigation";
import { Pencil, Plus, Trash2 } from "lucide-react";
import { useState, type ReactNode } from "react";
import { OwnerPropertyForm, type AreaOption } from "@/components/features/owner-property-form";
import { OwnerRoomSheet } from "@/components/features/owner-room-sheet";
import { AccordionItem } from "@/components/ui/accordion";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { ButtonPair } from "@/components/ui/button-pair";
import { ConfirmDialog } from "@/components/ui/dialog";
import { InlineAlert } from "@/components/ui/inline-alert";
import { Switch } from "@/components/ui/switch";
import { TextField } from "@/components/ui/text-field";
import { callBff } from "@/lib/api/browser";
import { showApiError } from "@/lib/api/show-error";
import { digitsOnly, formatRupiah, formatThousands } from "@/lib/format";
import { moderationStatusLabel } from "@/lib/labels";
import type { OwnerProperty, OwnerRoom } from "@/lib/owner";
import { toast } from "@/lib/toast";

export interface OwnerPropertyDetailProps {
  property: OwnerProperty;
  rooms: OwnerRoom[];
  areas: AreaOption[];
  photos?: ReactNode;
}

export function OwnerPropertyDetail({ property, rooms, areas, photos }: OwnerPropertyDetailProps) {
  const router = useRouter();
  const navigate = (href: string) => router.push(href);
  const [busy, setBusy] = useState<string | null>(null);
  const [discount, setDiscount] = useState(String(property.flash_sale_discount || ""));
  const [roomSheet, setRoomSheet] = useState<{ room?: OwnerRoom } | null>(null);
  const [roomToDelete, setRoomToDelete] = useState<OwnerRoom | null>(null);
  const moderation = moderationStatusLabel(property.moderation_status);
  const suspended = property.moderation_status === "suspended";

  async function send(key: string, path: string, method: "POST" | "PATCH" | "DELETE", body?: unknown) {
    setBusy(key);
    const result = await callBff(path, { method, body });
    setBusy(null);
    if (!result.ok) {
      showApiError(result, navigate, () => send(key, path, method, body));
      return false;
    }
    toast.success(result.data.meta.message);
    router.refresh();
    return true;
  }

  const patchProperty = (key: string, body: unknown) => send(key, `/api/owner/properties/${property.id}`, "PATCH", body);

  return (
    <div className="flex flex-col gap-6 px-4 py-4">
      <div className="flex flex-col gap-3">
        <div className="flex flex-wrap items-center gap-2">
          <Badge color={moderation.color}>{moderation.label}</Badge>
          {!property.is_active && <Badge color="neutral">Nonaktif</Badge>}
        </div>
        <h1 className="break-words text-title-lg font-semibold text-neutral-900">{property.name}</h1>
        {property.moderation_status === "pending" && (
          <InlineAlert variant="info">Properti sedang ditinjau admin. Kamu tetap bisa melengkapi kamar dan foto.</InlineAlert>
        )}
        {property.moderation_status === "rejected" && (
          <InlineAlert variant="danger" title="Properti ditolak admin">
            <p className="break-words">{property.moderation_note ?? "Perbaiki data properti lalu ajukan ulang."}</p>
            <Button
              size="sm"
              className="mt-2"
              loading={busy === "submit"}
              onClick={() => send("submit", `/api/owner/properties/${property.id}/submit`, "POST")}
            >
              Ajukan ulang
            </Button>
          </InlineAlert>
        )}
        {suspended && (
          <InlineAlert variant="danger" title="Properti dinonaktifkan admin">
            <p className="break-words">{property.moderation_note ?? "Hubungi admin untuk memulihkan properti ini."}</p>
          </InlineAlert>
        )}
      </div>

      <section className="flex flex-col gap-4 rounded-lg border border-neutral-200 p-4">
        <SwitchRow
          label="Tampilkan di pencarian"
          description="Matikan bila kost sedang tidak menerima penyewa."
          checked={property.is_active}
          disabled={suspended || busy !== null}
          onChange={(isActive) => patchProperty("active", { is_active: isActive })}
        />
        <SwitchRow
          label="Promo"
          description="Potongan harga untuk pembayaran pertama."
          checked={property.is_flash_sale}
          disabled={suspended || busy !== null}
          onChange={(isFlashSale) => patchProperty("promo", { is_flash_sale: isFlashSale })}
        />
        {property.is_flash_sale && (
          <div className="flex flex-col gap-3">
            <TextField
              label="Potongan harga promo"
              prefix="Rp"
              inputMode="numeric"
              value={formatThousands(discount)}
              onChange={(event) => setDiscount(digitsOnly(event.target.value))}
            />
            <Button
              fullWidth
              disabled={suspended}
              loading={busy === "discount"}
              onClick={() => patchProperty("discount", { flash_sale_discount: Number(discount || 0) })}
            >
              Simpan potongan
            </Button>
          </div>
        )}
      </section>

      <section className="flex flex-col gap-3">
        <div className="flex items-center justify-between gap-3">
          <h2 className="text-title-md font-semibold text-neutral-900">Kamar</h2>
          <Button size="sm" icon={<Plus aria-hidden="true" />} disabled={suspended} onClick={() => setRoomSheet({})}>
            Tambah kamar
          </Button>
        </div>
        {rooms.length === 0 ? (
          <p className="text-body text-neutral-600">Tambahkan minimal satu kamar agar kost bisa disewa.</p>
        ) : (
          rooms.map((room) => (
            <article key={room.id} className="flex flex-col gap-3 rounded-lg border border-neutral-200 p-3">
              <div className="flex min-w-0 flex-col gap-1">
                <p className="break-words text-title-sm font-semibold text-neutral-900">{room.name}</p>
                <p className="text-price font-bold tabular-nums text-neutral-900">
                  {formatRupiah(room.price_monthly)}
                  <span className="text-body font-normal text-neutral-600"> /bulan</span>
                </p>
                <p className="text-caption text-neutral-600">
                  {room.available} kosong, {room.held} dikunci, {room.occupied} terisi
                </p>
              </div>
              <ButtonPair>
                <Button variant="danger" icon={<Trash2 aria-hidden="true" />} onClick={() => setRoomToDelete(room)}>
                  Hapus
                </Button>
                <Button variant="neutral" icon={<Pencil aria-hidden="true" />} onClick={() => setRoomSheet({ room })}>
                  Ubah
                </Button>
              </ButtonPair>
            </article>
          ))
        )}
      </section>

      {photos}

      <section className="flex flex-col">
        <AccordionItem title="Ubah data kost">
          <OwnerPropertyForm areas={areas} property={property} />
        </AccordionItem>
      </section>

      <OwnerRoomSheet
        open={roomSheet !== null}
        onClose={() => setRoomSheet(null)}
        propertyId={property.id}
        room={roomSheet?.room}
      />
      <ConfirmDialog
        open={roomToDelete !== null}
        onClose={() => setRoomToDelete(null)}
        title="Hapus kamar?"
        description="Kamar beserta fotonya dihapus permanen. Kamar yang sudah pernah dibooking tidak bisa dihapus."
        confirmLabel="Hapus"
        loading={busy === "delete-room"}
        onConfirm={async () => {
          if (roomToDelete) await send("delete-room", `/api/owner/rooms/${roomToDelete.id}`, "DELETE");
          setRoomToDelete(null);
        }}
      />
    </div>
  );
}

function SwitchRow({
  label,
  description,
  checked,
  disabled,
  onChange,
}: {
  label: string;
  description: string;
  checked: boolean;
  disabled: boolean;
  onChange: (checked: boolean) => void;
}) {
  return (
    <div className="flex items-center justify-between gap-4">
      <div className="min-w-0">
        <p className="text-body font-medium text-neutral-900">{label}</p>
        <p className="text-caption text-neutral-600">{description}</p>
      </div>
      <Switch aria-label={label} checked={checked} disabled={disabled} onChange={onChange} />
    </div>
  );
}
