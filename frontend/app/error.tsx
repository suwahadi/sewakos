"use client";

import { RefreshCw } from "lucide-react";
import { Button } from "@/components/ui/button";

export default function AppError({ reset }: { error: Error & { digest?: string }; reset: () => void }) {
  return (
    <main className="flex min-h-dvh flex-col items-center justify-center gap-4 px-4 py-10 text-center">
      <p className="text-title-md font-semibold text-neutral-900">Halaman belum bisa dimuat</p>
      <p className="text-body text-neutral-600">Periksa koneksi lalu coba lagi.</p>
      <Button variant="neutral" icon={<RefreshCw aria-hidden="true" />} onClick={reset}>
        Coba lagi
      </Button>
    </main>
  );
}
