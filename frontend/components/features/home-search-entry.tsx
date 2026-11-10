"use client";

import { useRouter } from "next/navigation";
import { SearchBar } from "./search-bar";

export function HomeSearchEntry() {
  const router = useRouter();
  return <SearchBar onActivate={() => router.push("/cari")} />;
}
