import Image from "next/image";
import Link from "next/link";
import { notFound } from "next/navigation";
import { AdminPropertyActions } from "@/components/features/admin-property-actions";
import { Badge } from "@/components/ui/badge";
import { ErrorState } from "@/components/ui/error-state";
import { InlineAlert } from "@/components/ui/inline-alert";
import type { AdminProperty } from "@/lib/admin";
import { apiFetch } from "@/lib/api/client";
import { formatPhoneDisplay } from "@/lib/format";
import { genderLabel, moderationStatusLabel } from "@/lib/labels";

export const metadata = { title: "Detail properti" };

export default async function AdminPropertyPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  const result = await apiFetch<{ property: AdminProperty }>(`/admin/properties/${encodeURIComponent(id)}`);
  if (!result.ok && result.httpStatus === 404) notFound();
  if (!result.ok) return <ErrorState title={result.meta.message} />;
  const property = result.data.property;
  const moderation = moderationStatusLabel(property.moderation_status);

  return (
    <div className="flex flex-col gap-6">
      <Link href="/admin/properti" className="text-label font-semibold text-brand-700">
        Kembali ke daftar
      </Link>
      <div className="flex flex-col gap-2">
        <div className="flex flex-wrap items-center gap-2">
          <Badge color={moderation.color}>{moderation.label}</Badge>
          {!property.is_active && <Badge color="neutral">Nonaktif oleh pemilik</Badge>}
        </div>
        <h1 className="break-words text-title-lg font-semibold text-neutral-900">{property.name}</h1>
        <p className="text-body text-neutral-600">
          {property.address}, {property.area.name}
        </p>
      </div>
      {property.moderation_note && (
        <InlineAlert variant="info" title="Catatan moderasi">
          <p className="break-words">{property.moderation_note}</p>
        </InlineAlert>
      )}
      <AdminPropertyActions propertyId={property.id} status={property.moderation_status} />

      <section className="grid gap-4 lg:grid-cols-2">
        <dl className="flex flex-col gap-2 rounded-lg border border-neutral-200 p-4">
          <Row label="Pemilik">{property.owner.name}</Row>
          <Row label="Nomor HP pemilik">{formatPhoneDisplay(property.owner.phone_number)}</Row>
          <Row label="Tipe">{genderLabel(property.gender)}</Row>
          <Row label="Jumlah kamar">{property.rooms_count}</Row>
          <Row label="Koordinat">
            {property.latitude}, {property.longitude}
          </Row>
        </dl>
        <div className="rounded-lg border border-neutral-200 p-4">
          <h2 className="mb-2 text-title-sm font-semibold text-neutral-900">Deskripsi</h2>
          <p className="whitespace-pre-line break-words text-body text-neutral-700">{property.description}</p>
        </div>
      </section>

      <section className="flex flex-col gap-3">
        <h2 className="text-title-md font-semibold text-neutral-900">Foto ({property.photos.length})</h2>
        {property.photos.length === 0 ? (
          <p className="text-body text-neutral-600">Pemilik belum mengunggah foto.</p>
        ) : (
          <div className="grid grid-cols-3 gap-2 lg:grid-cols-6">
            {property.photos.map((photo) => (
              <div key={photo.id} className="relative aspect-square overflow-hidden rounded-md bg-neutral-100">
                <Image src={photo.thumbnail_url} alt={photo.caption ?? ""} fill sizes="(min-width: 1024px) 16vw, 33vw" className="object-cover" />
              </div>
            ))}
          </div>
        )}
      </section>
    </div>
  );
}

function Row({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex justify-between gap-3">
      <dt className="shrink-0 text-body text-neutral-600">{label}</dt>
      <dd className="min-w-0 break-words text-right text-body text-neutral-900">{children}</dd>
    </div>
  );
}
