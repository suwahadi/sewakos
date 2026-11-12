import { ListingCard, type ListingCardRoom } from "@/components/features/listing-card";
import { cn } from "@/lib/cn";

const LAYOUT_CLASSES = {
  home: "sm:grid-cols-3 lg:grid-cols-4",
  detail: "lg:grid-cols-3",
} as const;

export function ListingGrid({ rooms, layout }: { rooms: ListingCardRoom[]; layout: keyof typeof LAYOUT_CLASSES }) {
  return (
    <div className={cn("grid grid-cols-2 gap-x-3 gap-y-6", LAYOUT_CLASSES[layout])}>
      {rooms.map((room) => (
        <ListingCard key={room.id} room={room} vertical />
      ))}
    </div>
  );
}
