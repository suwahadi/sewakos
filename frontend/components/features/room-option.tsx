import { Check } from "lucide-react";
import { facilityIcon } from "@/components/features/facility-list";
import { cn } from "@/lib/cn";

const MAX_FACILITY_ICONS = 3;

export interface RoomOptionData {
  id: number;
  name: string;
  price_label: string;
  available: number;
  facilities: string[];
}

export function RoomOption({
  room,
  selected,
  onSelect,
}: {
  room: RoomOptionData;
  selected: boolean;
  onSelect: () => void;
}) {
  const full = room.available === 0;
  const shown = room.facilities.slice(0, MAX_FACILITY_ICONS);
  const remaining = room.facilities.length - shown.length;

  return (
    <button
      type="button"
      role="radio"
      aria-checked={selected}
      aria-disabled={full}
      disabled={full}
      onClick={onSelect}
      className={cn(
        "flex w-full flex-col gap-2 rounded-lg border p-4 text-left",
        selected ? "border-2 border-brand-600 bg-brand-50" : "border-neutral-200",
        full && "opacity-60",
      )}
    >
      <p className="text-title-sm font-semibold text-neutral-900">{room.name}</p>
      {shown.length > 0 && (
        <span className="flex items-center gap-2 text-neutral-600">
          {shown.map((facility) => {
            const Icon = facilityIcon(facility);
            return <Icon key={facility} aria-hidden="true" size={18} strokeWidth={1.75} />;
          })}
          {remaining > 0 && <span className="text-caption">+{remaining}</span>}
        </span>
      )}
      <div className="flex items-center justify-between">
        <p className="text-price font-bold text-neutral-900">
          {room.price_label}
          <span className="text-body font-normal text-neutral-600"> /bulan</span>
        </p>
        {full ? (
          <span className="text-caption-strong text-danger">Penuh</span>
        ) : selected ? (
          <Check aria-hidden="true" size={20} strokeWidth={1.75} className="text-brand-600" />
        ) : null}
      </div>
    </button>
  );
}
