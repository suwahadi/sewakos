import {
  AirVent,
  Bath,
  BedDouble,
  Bike,
  Camera,
  Car,
  Check,
  Clock,
  CookingPot,
  Fan,
  Flame,
  Moon,
  Refrigerator,
  Shirt,
  Sofa,
  Sun,
  Table2,
  Tv,
  WashingMachine,
  Wifi,
  type LucideIcon,
} from "lucide-react";
import { AccordionItem } from "@/components/ui/accordion";

const ICON_RULES: { match: RegExp; icon: LucideIcon }[] = [
  { match: /cctv/, icon: Camera },
  { match: /wifi/, icon: Wifi },
  { match: /\bac\b/, icon: AirVent },
  { match: /kipas/, icon: Fan },
  { match: /mandi/, icon: Bath },
  { match: /kasur|bed/, icon: BedDouble },
  { match: /lemari/, icon: Shirt },
  { match: /meja/, icon: Table2 },
  { match: /\btv\b|televisi/, icon: Tv },
  { match: /water heater|air panas/, icon: Flame },
  { match: /kulkas/, icon: Refrigerator },
  { match: /dapur/, icon: CookingPot },
  { match: /laundry|cuci/, icon: WashingMachine },
  { match: /jemur/, icon: Sun },
  { match: /ruang tamu/, icon: Sofa },
  { match: /parkir motor/, icon: Bike },
  { match: /parkir mobil/, icon: Car },
  { match: /mushola|masjid/, icon: Moon },
  { match: /24 jam/, icon: Clock },
];

export function facilityIcon(facility: string): LucideIcon {
  const normalized = facility.toLowerCase();
  return ICON_RULES.find((rule) => rule.match.test(normalized))?.icon ?? Check;
}

const VISIBLE_COUNT = 6;

function FacilityGrid({ facilities }: { facilities: string[] }) {
  return (
    <div className="grid grid-cols-2 gap-3 md:grid-cols-3">
      {facilities.map((facility) => {
        const Icon = facilityIcon(facility);
        return (
          <span key={facility} className="flex min-w-0 items-center gap-2 text-body text-neutral-900">
            <Icon aria-hidden="true" size={20} strokeWidth={1.75} className="shrink-0 text-neutral-600" />
            <span className="truncate">{facility}</span>
          </span>
        );
      })}
    </div>
  );
}

export function FacilityList({ facilities }: { facilities: string[] }) {
  if (facilities.length === 0) return null;

  const visible = facilities.slice(0, VISIBLE_COUNT);
  const rest = facilities.slice(VISIBLE_COUNT);

  return (
    <div className="flex flex-col gap-3">
      <FacilityGrid facilities={visible} />
      {rest.length > 0 && (
        <AccordionItem title={`Lihat semua fasilitas (${facilities.length})`}>
          <FacilityGrid facilities={rest} />
        </AccordionItem>
      )}
    </div>
  );
}
