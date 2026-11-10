import type { ReactNode } from "react";

/** Wraps exactly two buttons so they always render 50:50 (DESIGN.md §0.1-B). */
export function ButtonPair({ children }: { children: [ReactNode, ReactNode] }) {
  return <div className="grid grid-cols-2 gap-3">{children}</div>;
}
