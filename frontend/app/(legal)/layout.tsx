import Link from "next/link";
import type { ReactNode } from "react";

export default function LegalLayout({ children }: { children: ReactNode }) {
  return (
    <main className="mx-auto flex min-h-dvh max-w-[680px] flex-col gap-6 px-4 py-8">
      <Link href="/" className="self-start text-title-lg font-bold text-brand-700">
        Sewa Kos
      </Link>
      {children}
    </main>
  );
}
