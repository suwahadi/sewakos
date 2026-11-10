import { ArrowUpDown, SlidersHorizontal, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { ButtonPair } from "@/components/ui/button-pair";
import { CountBadge } from "@/components/ui/badge";
import { Chip, ChipRow } from "@/components/ui/chip";

export interface ActiveFilterChip {
  key: string;
  label: string;
  onRemove: () => void;
}

export interface FilterBarProps {
  activeFilterCount: number;
  onOpenFilter: () => void;
  onOpenSort: () => void;
  activeChips: ActiveFilterChip[];
}

/** DESIGN.md §6.27 — sticky trigger row above the active-filter chips; hides with the header on scroll (§0.1-E). */
export function FilterBar({ activeFilterCount, onOpenFilter, onOpenSort, activeChips }: FilterBarProps) {
  return (
    <div className="sticky top-14 z-[var(--z-sticky)] bg-neutral-0 py-3">
      <div className="px-4">
        <ButtonPair>
          <span className="relative inline-flex w-full">
            <Button
              variant="outline"
              size="sm"
              fullWidth
              icon={<SlidersHorizontal aria-hidden="true" />}
              onClick={onOpenFilter}
            >
              Filter
            </Button>
            {activeFilterCount > 0 && (
              <CountBadge count={activeFilterCount} className="absolute -right-1 -top-1" />
            )}
          </span>
          <Button variant="outline" size="sm" fullWidth icon={<ArrowUpDown aria-hidden="true" />} onClick={onOpenSort}>
            Urutkan
          </Button>
        </ButtonPair>
      </div>
      {activeChips.length > 0 && (
        <ChipRow className="mt-2">
          {activeChips.map((chip) => (
            <Chip key={chip.key} active onClick={chip.onRemove}>
              {chip.label}
              <X aria-hidden="true" size={14} strokeWidth={1.75} />
            </Chip>
          ))}
        </ChipRow>
      )}
    </div>
  );
}
