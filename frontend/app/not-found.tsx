"use client";

import { useRouter } from "next/navigation";
import { NotFoundState } from "@/components/ui/error-state";

/** Next.js's special 404 file — rendered for any unmatched route (DESIGN.md §6.19). */
export default function NotFound() {
  const router = useRouter();
  return (
    <div className="flex min-h-[60dvh] items-center justify-center px-4">
      <NotFoundState onGoHome={() => router.push("/")} />
    </div>
  );
}
