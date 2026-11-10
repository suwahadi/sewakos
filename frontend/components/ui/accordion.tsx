"use client";

import { ChevronDown } from "lucide-react";
import { useId, useState, type ReactNode } from "react";
import { cn } from "@/lib/cn";

export interface AccordionItemProps {
  title: string;
  children: ReactNode;
  defaultOpen?: boolean;
}

export function AccordionItem({ title, children, defaultOpen = false }: AccordionItemProps) {
  const [open, setOpen] = useState(defaultOpen);
  const id = useId();

  return (
    <div className="border-b border-neutral-200">
      <h3>
        <button
          type="button"
          id={`${id}-trigger`}
          aria-expanded={open}
          aria-controls={`${id}-panel`}
          onClick={() => setOpen((value) => !value)}
          className="flex min-h-12 w-full items-center justify-between gap-3 py-3 text-left text-title-sm font-semibold text-neutral-900"
        >
          {title}
          <ChevronDown
            aria-hidden="true"
            size={20}
            strokeWidth={1.75}
            className={cn("shrink-0 text-neutral-600 transition-transform duration-[var(--duration-base)] ease-standard", open && "rotate-180")}
          />
        </button>
      </h3>
      <div
        id={`${id}-panel`}
        role="region"
        aria-labelledby={`${id}-trigger`}
        inert={!open}
        className={cn(
          "grid transition-[grid-template-rows] duration-[var(--duration-base)] ease-standard",
          open ? "grid-rows-[1fr]" : "grid-rows-[0fr]",
        )}
      >
        <div className="overflow-hidden">
          <div className="pb-4 text-body text-neutral-700">{children}</div>
        </div>
      </div>
    </div>
  );
}
