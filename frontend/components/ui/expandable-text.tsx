"use client";

import { ChevronDown } from "lucide-react";
import { useId, useLayoutEffect, useRef, useState } from "react";
import { cn } from "@/lib/cn";

export function ExpandableText({ text }: { text: string }) {
  const textRef = useRef<HTMLParagraphElement>(null);
  const [expanded, setExpanded] = useState(false);
  const [clamped, setClamped] = useState(false);
  const textId = useId();

  useLayoutEffect(() => {
    const element = textRef.current;
    if (!element) return;
    const measure = () => {
      if (!expanded) setClamped(element.scrollHeight > element.clientHeight + 1);
    };
    measure();
    window.addEventListener("resize", measure);
    return () => window.removeEventListener("resize", measure);
  }, [text, expanded]);

  return (
    <div className="flex flex-col items-start gap-1">
      <p
        ref={textRef}
        id={textId}
        className={cn("whitespace-pre-line text-body text-neutral-700", !expanded && "line-clamp-3")}
      >
        {text}
      </p>
      {clamped && (
        <button
          type="button"
          aria-expanded={expanded}
          aria-controls={textId}
          onClick={() => setExpanded((value) => !value)}
          className="inline-flex min-h-11 items-center gap-1 text-label font-semibold text-brand-700"
        >
          {expanded ? "Tutup" : "Selengkapnya"}
          <ChevronDown
            aria-hidden="true"
            size={16}
            strokeWidth={1.75}
            className={cn("transition-transform duration-[var(--duration-base)] ease-standard", expanded && "rotate-180")}
          />
        </button>
      )}
    </div>
  );
}
