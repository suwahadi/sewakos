import Link from "next/link";
import type { ReactNode } from "react";

/** DESIGN.md §7.0 wireframe — "Masuk & daftar": centered, max 400px, no AppHeader/BottomNav. */
export default function AuthLayout({ children }: { children: ReactNode }) {
  return (
    <main className="mx-auto flex min-h-dvh max-w-[400px] flex-col justify-center gap-6 px-4 py-8">
      <Link href="/" className="text-center text-title-lg font-bold text-brand-700">
        Sewa Kos
      </Link>
      {children}
    </main>
  );
}
