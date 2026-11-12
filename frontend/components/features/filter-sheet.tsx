"use client";

import { useState } from "react";
import { BottomSheet } from "@/components/ui/bottom-sheet";
import { Button } from "@/components/ui/button";
import { ButtonPair } from "@/components/ui/button-pair";
import { Chip, ChipRow } from "@/components/ui/chip";
import { SegmentedControl } from "@/components/ui/segmented-control";
import { Switch } from "@/components/ui/switch";
import { TextField } from "@/components/ui/text-field";
import { DEFAULT_SEARCH_FILTERS } from "@/lib/listings";

export interface SearchFilters {
  rentType: 1 | 2 | 3;
  gender: number[];
  priceMin?: number;
  priceMax?: number;
  flashSale?: boolean;
  isAvailable?: boolean;
}

const GENDER_OPTIONS = [
  { value: 1, label: "Putra" },
  { value: 2, label: "Putri" },
  { value: 0, label: "Campur" },
];

export interface FilterSheetProps {
  open: boolean;
  onClose: () => void;
  filters: SearchFilters;
  onApply: (filters: SearchFilters) => void;
}

/** DESIGN.md §6.27 — maps 1:1 to `filters` in `POST /listings/search`. */
export function FilterSheet({ open, onClose, filters, onApply }: FilterSheetProps) {
  const [draft, setDraft] = useState(filters);
  const [prevOpen, setPrevOpen] = useState(open);

  // Reset the draft to the committed filters each time the sheet opens — done during render
  // (React's documented "adjusting state when a prop changes" pattern, using state rather than a
  // ref since refs can't be read during render) instead of an effect, so there's no stale-draft
  // flash before an effect would have run.
  if (open !== prevOpen) {
    setPrevOpen(open);
    if (open) setDraft(filters);
  }

  return (
    <BottomSheet
      open={open}
      onClose={onClose}
      title="Filter"
      footer={
        <ButtonPair>
          <Button variant="neutral" onClick={() => setDraft(DEFAULT_SEARCH_FILTERS)}>
            Atur ulang
          </Button>
          <Button
            onClick={() => {
              onApply(draft);
              onClose();
            }}
          >
            Tampilkan
          </Button>
        </ButtonPair>
      }
    >
      <div className="flex flex-col gap-6 py-2">
        <section>
          <p className="mb-2 text-label font-medium text-neutral-700">Tipe sewa</p>
          <SegmentedControl
            aria-label="Tipe sewa"
            value={String(draft.rentType)}
            onChange={(value) => setDraft((prev) => ({ ...prev, rentType: Number(value) as 1 | 2 | 3 }))}
            options={[
              { value: "3", label: "Bulanan" },
              { value: "2", label: "Mingguan" },
              { value: "1", label: "Harian" },
            ]}
          />
        </section>
        <section>
          <p className="mb-2 text-label font-medium text-neutral-700">Gender</p>
          <ChipRow className="px-0">
            {GENDER_OPTIONS.map((option) => {
              const active = draft.gender.includes(option.value);
              return (
                <Chip
                  key={option.value}
                  active={active}
                  onClick={() =>
                    setDraft((prev) => ({
                      ...prev,
                      gender: active
                        ? prev.gender.filter((value) => value !== option.value)
                        : [...prev.gender, option.value],
                    }))
                  }
                >
                  {option.label}
                </Chip>
              );
            })}
          </ChipRow>
        </section>
        <section>
          <p className="mb-2 text-label font-medium text-neutral-700">Rentang harga</p>
          <div className="grid grid-cols-2 gap-3">
            <TextField
              label="Minimum"
              inputMode="numeric"
              placeholder="Rp0"
              value={draft.priceMin ?? ""}
              onChange={(event) => setDraft((prev) => ({ ...prev, priceMin: Number(event.target.value) || undefined }))}
            />
            <TextField
              label="Maksimum"
              inputMode="numeric"
              placeholder="Rp20.000.000"
              value={draft.priceMax ?? ""}
              onChange={(event) => setDraft((prev) => ({ ...prev, priceMax: Number(event.target.value) || undefined }))}
            />
          </div>
        </section>
        <section className="flex items-center justify-between gap-3">
          <p className="text-body text-neutral-900">Hanya yang sedang promo</p>
          <Switch
            aria-label="Hanya yang sedang promo"
            checked={Boolean(draft.flashSale)}
            onChange={(checked) => setDraft((prev) => ({ ...prev, flashSale: checked }))}
          />
        </section>
        <section className="flex items-center justify-between gap-3">
          <p className="text-body text-neutral-900">Hanya kamar yang tersedia</p>
          <Switch
            aria-label="Hanya kamar yang tersedia"
            checked={Boolean(draft.isAvailable)}
            onChange={(checked) => setDraft((prev) => ({ ...prev, isAvailable: checked }))}
          />
        </section>
      </div>
    </BottomSheet>
  );
}

export interface SortOption {
  field: "price" | "availability";
  direction: "+" | "-";
}

const SORT_CHOICES: { key: string; sort: SortOption; label: string }[] = [
  { key: "price-asc", sort: { field: "price", direction: "+" }, label: "Harga terendah" },
  { key: "price-desc", sort: { field: "price", direction: "-" }, label: "Harga tertinggi" },
  { key: "availability-desc", sort: { field: "availability", direction: "-" }, label: "Kamar terbanyak" },
];

export interface SortSheetProps {
  open: boolean;
  onClose: () => void;
  value: SortOption;
  onChange: (value: SortOption) => void;
}

/** DESIGN.md §6.27 — separate from FilterSheet; applies immediately on selection. */
export function SortSheet({ open, onClose, value, onChange }: SortSheetProps) {
  return (
    <BottomSheet open={open} onClose={onClose} title="Urutkan">
      <div role="radiogroup" aria-label="Urutkan" className="flex flex-col py-2">
        {SORT_CHOICES.map((option) => {
          const selected = option.sort.field === value.field && option.sort.direction === value.direction;
          return (
            <button
              key={option.key}
              type="button"
              role="radio"
              aria-checked={selected}
              onClick={() => {
                onChange(option.sort);
                onClose();
              }}
              className={`flex h-12 items-center px-2 text-body ${selected ? "font-semibold text-brand-700" : "text-neutral-900"}`}
            >
              {option.label}
            </button>
          );
        })}
      </div>
    </BottomSheet>
  );
}
