"use client";

import { ChevronRight } from "lucide-react";
import { useState } from "react";
import { BottomSheet } from "@/components/ui/bottom-sheet";
import { facilityIcon } from "@/lib/facility-icons";

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

  return <FacilityPreview facilities={facilities} />;
}

function FacilityPreview({ facilities }: { facilities: string[] }) {
  const [open, setOpen] = useState(false);

  return (
    <div className="flex flex-col gap-3">
      <FacilityGrid facilities={facilities.slice(0, VISIBLE_COUNT)} />
      {facilities.length > VISIBLE_COUNT && (
        <button
          type="button"
          onClick={() => setOpen(true)}
          className="inline-flex min-h-11 items-center gap-1 self-start text-label font-semibold text-brand-700"
        >
          Lihat semua fasilitas
          <ChevronRight aria-hidden="true" size={16} strokeWidth={1.75} />
        </button>
      )}
      <BottomSheet open={open} onClose={() => setOpen(false)} title="Fasilitas">
        <div className="flex flex-col gap-4 pb-6">
          <FacilityGrid facilities={facilities} />
        </div>
      </BottomSheet>
    </div>
  );
}
