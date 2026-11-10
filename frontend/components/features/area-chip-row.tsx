"use client";

import { useRouter } from "next/navigation";
import { Chip, ChipRow } from "@/components/ui/chip";

export interface AreaChipOption {
  id: number;
  name: string;
}

export function AreaChipRow({ areas }: { areas: AreaChipOption[] }) {
  const router = useRouter();
  if (areas.length === 0) return null;

  return (
    <ChipRow>
      {areas.map((area) => (
        <Chip key={area.id} onClick={() => router.push(`/cari?area_id=${area.id}&area_name=${encodeURIComponent(area.name)}`)}>
          {area.name}
        </Chip>
      ))}
    </ChipRow>
  );
}
