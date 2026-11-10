import { Search } from "lucide-react";
import type { InputHTMLAttributes } from "react";
import { cn } from "@/lib/cn";

export interface SearchBarProps extends InputHTMLAttributes<HTMLInputElement> {
  /** Renders as a button that navigates away instead of an editable field (DESIGN.md §6.26 — home page use). */
  onActivate?: () => void;
}

export function SearchBar({
  onActivate,
  className,
  placeholder = "Cari kota, kampus, atau nama kost",
  ...props
}: SearchBarProps) {
  const shellClassName = cn(
    "flex h-12 w-full items-center gap-2 rounded-full border border-neutral-200 bg-neutral-50 px-4",
    className,
  );

  if (onActivate) {
    return (
      <button type="button" onClick={onActivate} className={cn(shellClassName, "text-left text-body text-neutral-500")}>
        <Search aria-hidden="true" size={20} strokeWidth={1.75} className="shrink-0 text-neutral-500" />
        {placeholder}
      </button>
    );
  }

  return (
    <div className={shellClassName}>
      <Search aria-hidden="true" size={20} strokeWidth={1.75} className="shrink-0 text-neutral-500" />
      <input
        type="search"
        placeholder={placeholder}
        className="w-full bg-transparent text-body text-neutral-900 placeholder:text-neutral-500 focus:outline-none"
        {...props}
      />
    </div>
  );
}
