import Image from "next/image";
import Link from "next/link";
import { Building2 } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { moderationStatusLabel } from "@/lib/labels";
import type { OwnerProperty } from "@/lib/owner";

export function OwnerPropertyCard({ property }: { property: OwnerProperty }) {
  const moderation = moderationStatusLabel(property.moderation_status);
  const cover = property.photos.find((photo) => photo.room_id === null) ?? property.photos[0];

  return (
    <Link href={`/owner/properti/${property.id}`} prefetch={false} className="flex min-w-0 gap-3 rounded-lg border border-neutral-200 p-3">
      <span className="relative flex aspect-[4/3] w-24 shrink-0 items-center justify-center overflow-hidden rounded-md bg-neutral-100">
        {cover ? (
          <Image src={cover.thumbnail_url} alt="" fill sizes="96px" className="object-cover" />
        ) : (
          <Building2 aria-hidden="true" size={28} strokeWidth={1.5} className="text-neutral-400" />
        )}
      </span>
      <div className="flex min-w-0 flex-col gap-1">
        <div className="flex flex-wrap items-center gap-2">
          <Badge color={moderation.color}>{moderation.label}</Badge>
          {!property.is_active && <Badge color="neutral">Nonaktif</Badge>}
        </div>
        <p className="line-clamp-2 break-words text-title-sm font-semibold text-neutral-900">{property.name}</p>
        <p className="text-caption text-neutral-600">
          {property.area.name}, {property.rooms_count} kamar
        </p>
      </div>
    </Link>
  );
}
