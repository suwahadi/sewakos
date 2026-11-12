import Link from "next/link";
import type { ReactNode } from "react";
import { SiteFooter } from "@/components/features/site-footer";

export default function LegalLayout({ children }: { children: ReactNode }) {
  return (
    <div className="flex min-h-dvh flex-col">
      <main className="mx-auto flex w-full max-w-[680px] flex-col gap-6 px-4 py-8">
        <Link href="/" className="self-start text-title-lg font-bold text-brand-700">
          Sewa Kos
        </Link>
        {children}
      </main>
      <SiteFooter />
    </div>
  );
}
