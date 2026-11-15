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
