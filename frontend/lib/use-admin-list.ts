"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";
import { ADMIN_PAGE_SIZE } from "./admin";
import { callBff } from "./api/browser";
import { showApiError } from "./api/show-error";

export type AdminQuery = Record<string, string>;

export function useAdminList<T>(path: string, field: string, initial: T[], initialQuery: AdminQuery = {}) {
  const router = useRouter();
  const [rows, setRows] = useState(initial);
  const [query, setQuery] = useState(initialQuery);
  const [page, setPage] = useState(1);
  const [hasMore, setHasMore] = useState(initial.length >= ADMIN_PAGE_SIZE);
  const [loading, setLoading] = useState(false);

  async function load(nextQuery: AdminQuery, nextPage: number) {
    setLoading(true);
    const params = new URLSearchParams(Object.entries(nextQuery).filter(([, value]) => value !== ""));
    params.set("page", String(nextPage));
    params.set("limit", String(ADMIN_PAGE_SIZE));
    const result = await callBff<Record<string, T[]>>(`/api/admin/${path}?${params}`);
    setLoading(false);
    if (!result.ok) {
      showApiError(result, (href) => router.push(href), () => load(nextQuery, nextPage));
      return;
    }
    const items = result.data[field] ?? [];
    setRows((current) => (nextPage === 1 ? items : [...current, ...items]));
    setQuery(nextQuery);
    setPage(nextPage);
    setHasMore(items.length >= ADMIN_PAGE_SIZE);
  }

  return {
    rows,
    query,
    loading,
    hasMore,
    filter: (next: AdminQuery) => load(next, 1),
    loadMore: () => load(query, page + 1),
    reload: () => load(query, 1),
  };
}
