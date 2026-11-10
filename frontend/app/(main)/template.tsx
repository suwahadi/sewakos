import type { ReactNode } from "react";

/** DESIGN.md §5.1 — re-mounted on every navigation (unlike layout.tsx), so the enter animation replays. */
export default function Template({ children }: { children: ReactNode }) {
  return <div className="animate-page-enter">{children}</div>;
}
