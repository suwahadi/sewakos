import { AirVent, BedDouble, Bath, Check, Wifi, type LucideIcon } from "lucide-react";
import { AccordionItem } from "@/components/ui/accordion";

const ICON_RULES: { match: (text: string) => boolean; icon: LucideIcon }[] = [
  { match: (text) => text.includes("wifi"), icon: Wifi },
  { match: (text) => /\bac\b/.test(text), icon: AirVent },
  { match: (text) => text.includes("mandi"), icon: Bath },
  { match: (text) => text.includes("kasur") || text.includes("bed"), icon: BedDouble },
  { match: (text) => text.includes("24 jam"), icon: Check },
];

export function facilityIcon(facility: string): LucideIcon {
  const normalized = facility.toLowerCase();
  return ICON_RULES.find((rule) => rule.match(normalized))?.icon ?? Check;
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
