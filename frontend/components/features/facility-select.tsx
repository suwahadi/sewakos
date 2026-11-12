"use client";

import { X } from "lucide-react";
import { useState } from "react";
import { BottomSheet } from "@/components/ui/bottom-sheet";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { IconButton } from "@/components/ui/icon-button";
import { FACILITY_CATALOG } from "@/lib/facilities";

export interface FacilitySelectProps {
  value: string[];
  onChange: (next: string[]) => void;
  error?: string;
}

export function FacilitySelect({ value, onChange, error }: FacilitySelectProps) {
  const [open, setOpen] = useState(false);
  const options = [...FACILITY_CATALOG, ...value.filter((item) => !(FACILITY_CATALOG as readonly string[]).includes(item))];

  function toggle(facility: string, checked: boolean) {
    onChange(checked ? [...value, facility] : value.filter((item) => item !== facility));
  }

  return (
    <div className="flex flex-col gap-2">
      <p className="text-label font-medium text-neutral-700">Fasilitas</p>
      {value.length > 0 ? (
        <ul className="flex flex-wrap gap-2" aria-label="Fasilitas terpilih">
          {value.map((facility) => (
            <li
              key={facility}
              className="inline-flex min-h-9 items-center gap-1 rounded-full border border-brand-600 bg-brand-50 py-0.5 pl-3 pr-0.5 text-label font-medium text-brand-800"
            >
              <span className="break-all">{facility}</span>
              <IconButton
                size="sm"
                aria-label={`Hapus ${facility}`}
                icon={<X aria-hidden="true" />}
                onClick={() => toggle(facility, false)}
              />
            </li>
          ))}
        </ul>
      ) : (
        <p className="text-caption text-neutral-600">Belum ada fasilitas dipilih.</p>
      )}
      <div>
        <Button variant="neutral" size="sm" onClick={() => setOpen(true)}>
          Pilih fasilitas
        </Button>
      </div>
      {error && <p className="text-caption text-danger">{error}</p>}

      <BottomSheet
        open={open}
        onClose={() => setOpen(false)}
        title="Pilih fasilitas"
        footer={
          <Button fullWidth onClick={() => setOpen(false)}>
            Selesai
          </Button>
        }
      >
        {open && (
          <div className="flex flex-col divide-y divide-neutral-200 pb-2">
            {options.map((facility) => (
              <Checkbox
                key={facility}
                label={facility}
                checked={value.includes(facility)}
                onChange={(event) => toggle(facility, event.target.checked)}
              />
            ))}
          </div>
        )}
      </BottomSheet>
    </div>
  );
}
