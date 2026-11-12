"use client";

import { BedDouble, Building2, MapPin, Search } from "lucide-react";
import { useEffect, useId, useRef, useState, type KeyboardEvent, type ReactNode } from "react";
import { GenderBadge } from "@/components/ui/gender-badge";

export interface SuggestionArea {
  title: string;
  area: string;
  latitude: number;
  longitude: number;
  geocode_id: number;
}
export interface SuggestionLanding {
  id: number;
  title: string;
  slug: string;
  gender: string;
}
export interface SuggestionRoomType {
  id: number;
  property_id: number;
  title: string;
}
export interface PopularArea {
  id: number;
  name: string;
}

type FlatItem =
  | { kind: "area"; key: string; data: SuggestionArea }
  | { kind: "landing"; key: string; data: SuggestionLanding }
  | { kind: "roomType"; key: string; data: SuggestionRoomType }
  | { kind: "popular"; key: string; data: PopularArea }
  | { kind: "keyword"; key: string; data: { keyword: string } };

const GROUP_LABEL: Record<FlatItem["kind"], string> = {
  area: "Area",
  landing: "Kost",
  roomType: "Tipe kamar",
  popular: "Area populer",
  keyword: "Pencarian",
};

export interface SuggestionListProps {
  /** Debounced 300ms after typing stops; fetching (with AbortController) is the caller's job. */
  onQueryChange: (query: string) => void;
  areas: SuggestionArea[];
  landings: SuggestionLanding[];
  roomTypes: SuggestionRoomType[];
  popularAreas: PopularArea[];
  onSelectArea: (area: SuggestionArea) => void;
  onSelectLanding: (landing: SuggestionLanding) => void;
  onSelectRoomType: (roomType: SuggestionRoomType) => void;
  onSelectPopularArea: (area: PopularArea) => void;
  onSubmitKeyword: (keyword: string) => void;
}

/** DESIGN.md §6.26 — ARIA combobox over SearchBar's input, with a 300ms typing debounce. */
export function SuggestionList({
  onQueryChange,
  areas,
  landings,
  roomTypes,
  popularAreas,
  onSelectArea,
  onSelectLanding,
  onSelectRoomType,
  onSelectPopularArea,
  onSubmitKeyword,
}: SuggestionListProps) {
  const [draft, setDraft] = useState("");
  const [activeIndex, setActiveIndex] = useState(-1);
  const listboxId = useId();
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current);
    };
  }, []);

  function handleChange(value: string) {
    setDraft(value);
    setActiveIndex(-1);
    if (debounceRef.current) clearTimeout(debounceRef.current);
    if (value.trim().length < 3) {
      onQueryChange("");
      return;
    }
    debounceRef.current = setTimeout(() => onQueryChange(value.trim()), 300);
  }

  const showSuggestions = draft.trim().length >= 3;
  const items: FlatItem[] = showSuggestions
    ? [
        { kind: "keyword", key: "keyword", data: { keyword: draft.trim() } },
        ...areas.map((data): FlatItem => ({ kind: "area", key: `area-${data.geocode_id}`, data })),
        ...landings.map((data): FlatItem => ({ kind: "landing", key: `landing-${data.id}`, data })),
        ...roomTypes.map((data): FlatItem => ({ kind: "roomType", key: `room-type-${data.id}`, data })),
      ]
    : popularAreas.map((data): FlatItem => ({ kind: "popular", key: `popular-${data.id}`, data }));

  function selectItem(item: FlatItem) {
    if (item.kind === "area") onSelectArea(item.data);
    else if (item.kind === "landing") onSelectLanding(item.data);
    else if (item.kind === "roomType") onSelectRoomType(item.data);
    else if (item.kind === "keyword") onSubmitKeyword(item.data.keyword);
    else onSelectPopularArea(item.data);
  }

  function handleKeyDown(event: KeyboardEvent<HTMLInputElement>) {
    if (items.length === 0) return;
    if (event.key === "ArrowDown") {
      event.preventDefault();
      setActiveIndex((index) => Math.min(index + 1, items.length - 1));
    } else if (event.key === "ArrowUp") {
      event.preventDefault();
      setActiveIndex((index) => Math.max(index - 1, 0));
    } else if (event.key === "Enter" && activeIndex >= 0) {
      const item = items[activeIndex];
      if (item) {
        event.preventDefault();
        selectItem(item);
      }
    } else if (event.key === "Enter" && showSuggestions) {
      event.preventDefault();
      onSubmitKeyword(draft.trim());
    }
  }

  return (
    <div className="relative">
      <input
        role="combobox"
        aria-expanded={items.length > 0}
        aria-controls={listboxId}
        aria-activedescendant={activeIndex >= 0 ? `${listboxId}-item-${activeIndex}` : undefined}
        aria-autocomplete="list"
        aria-label="Cari kota, kampus, atau nama kost"
        value={draft}
        onChange={(event) => handleChange(event.target.value)}
        onKeyDown={handleKeyDown}
        placeholder="Cari kota, kampus, atau nama kost"
        className="h-12 w-full rounded-full border border-neutral-200 bg-neutral-50 px-4 text-body text-neutral-900 focus:border-brand-600 focus:outline-none focus:ring-[3px] focus:ring-brand-100"
      />
      {items.length > 0 && (
        <ul
          id={listboxId}
          role="listbox"
          className="absolute z-[var(--z-dropdown)] mt-2 max-h-80 w-full overflow-y-auto rounded-lg border border-neutral-200 bg-neutral-0 py-1 shadow-overlay"
        >
          {items.map((item, index) => {
            const previous = items[index - 1];
            const showHeader = index === 0 || previous?.kind !== item.kind;
            return (
              <SuggestionGroupRow
                key={item.key}
                item={item}
                showHeader={showHeader}
                id={`${listboxId}-item-${index}`}
                active={index === activeIndex}
                onClick={() => selectItem(item)}
              />
            );
          })}
        </ul>
      )}
    </div>
  );
}

function SuggestionGroupRow({
  item,
  showHeader,
  id,
  active,
  onClick,
}: {
  item: FlatItem;
  showHeader: boolean;
  id: string;
  active: boolean;
  onClick: () => void;
}) {
  return (
    <>
      {showHeader && <GroupHeader>{GROUP_LABEL[item.kind]}</GroupHeader>}
      <SuggestionRow id={id} active={active} icon={<ItemIcon kind={item.kind} />} onClick={onClick}>
        <ItemLabel item={item} />
      </SuggestionRow>
    </>
  );
}

function ItemIcon({ kind }: { kind: FlatItem["kind"] }) {
  if (kind === "keyword") return <Search aria-hidden="true" size={18} strokeWidth={1.75} />;
  if (kind === "landing") return <Building2 aria-hidden="true" size={18} strokeWidth={1.75} />;
  if (kind === "roomType") return <BedDouble aria-hidden="true" size={18} strokeWidth={1.75} />;
  return <MapPin aria-hidden="true" size={18} strokeWidth={1.75} />;
}

function ItemLabel({ item }: { item: FlatItem }) {
  if (item.kind === "keyword") return <>Cari &ldquo;{item.data.keyword}&rdquo;</>;
  if (item.kind === "area") return <>{item.data.title}</>;
  if (item.kind === "roomType") return <>{item.data.title}</>;
  if (item.kind === "popular") return <>{item.data.name}</>;
  return (
    <span className="flex items-center gap-2">
      {item.data.title}
      <GenderBadge gender={item.data.gender} />
    </span>
  );
}

function GroupHeader({ children }: { children: string }) {
  return <li className="px-4 py-2 text-caption-strong text-neutral-600">{children}</li>;
}

function SuggestionRow({
  id,
  active,
  icon,
  onClick,
  children,
}: {
  id: string;
  active: boolean;
  icon: ReactNode;
  onClick: () => void;
  children: ReactNode;
}) {
  return (
    <li
      id={id}
      role="option"
      aria-selected={active}
      onClick={onClick}
      className={`flex h-[52px] cursor-pointer items-center gap-3 px-4 text-body text-neutral-900 ${active ? "bg-brand-50" : ""}`}
    >
      <span className="shrink-0 text-neutral-500">{icon}</span>
      {children}
    </li>
  );
}
