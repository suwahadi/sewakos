"use client";

import { useRouter } from "next/navigation";
import { useEffect, useState, type ReactNode } from "react";
import { RoomOption, type RoomOptionData } from "@/components/features/room-option";
import { FacilityList } from "@/components/features/facility-list";
import { AccordionItem } from "@/components/ui/accordion";
import { StickyActionBar } from "@/components/ui/sticky-action-bar";
import { callBff } from "@/lib/api/browser";

export function KostDetailClient({
  propertyId,
  description,
  rooms,
  reviews,
}: {
  propertyId: number;
  description: string;
  rooms: RoomOptionData[];
  reviews?: ReactNode;
}) {
  const router = useRouter();
  const firstAvailable = rooms.find((room) => room.available > 0) ?? rooms[0];
  const [selectedRoomId, setSelectedRoomId] = useState(firstAvailable?.id);

  useEffect(() => {
    const key = `viewed-property-${propertyId}`;
    if (sessionStorage.getItem(key)) return;
    sessionStorage.setItem(key, "1");
    callBff(`/api/listings/${propertyId}/view`, { method: "POST" });
  }, [propertyId]);

  const selectedRoom = rooms.find((room) => room.id === selectedRoomId);

  return (
    <div className="flex flex-col gap-6 px-4 pb-8">
      {rooms.length > 0 && (
        <section>
          <p className="mb-3 text-title-sm font-semibold text-neutral-900">Pilih kamar</p>
          <div role="radiogroup" aria-label="Pilih kamar" className="flex flex-col gap-3">
            {rooms.map((room) => (
              <RoomOption key={room.id} room={room} selected={room.id === selectedRoomId} onSelect={() => setSelectedRoomId(room.id)} />
            ))}
          </div>
        </section>
      )}

      {selectedRoom && selectedRoom.facilities.length > 0 && (
        <section>
          <p className="mb-3 text-title-sm font-semibold text-neutral-900">Fasilitas</p>
          <FacilityList facilities={selectedRoom.facilities} />
        </section>
      )}

      {description && (
        <section>
          <p className="mb-1 text-title-sm font-semibold text-neutral-900">Tentang kost</p>
          <AccordionItem title="Selengkapnya">
            <p className="whitespace-pre-line">{description}</p>
          </AccordionItem>
        </section>
      )}

      {reviews}

      {selectedRoom && (
        <StickyActionBar
          leadingLabel="Mulai dari"
          price={selectedRoom.price_label}
          priceUnit="/bulan"
          actionLabel="Ajukan sewa"
          onAction={() => router.push(`/kost/${propertyId}/sewa?room=${selectedRoom.id}`)}
          disabled={selectedRoom.available === 0}
        />
      )}
    </div>
  );
}
