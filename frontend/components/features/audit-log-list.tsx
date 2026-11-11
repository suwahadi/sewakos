"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";
import { DataTable, type DataTableColumn } from "@/components/ui/data-table";
import { LoadMore } from "@/components/ui/load-more";
import { callBff } from "@/lib/api/browser";
import { showApiError } from "@/lib/api/show-error";
import { formatDateShort, formatTimeWIB } from "@/lib/format";
import type { AuditLogEntry } from "@/lib/owner";

const PAGE_SIZE = 20;

function changesText(changes: AuditLogEntry["changes"]): string {
  return Object.entries(changes)
    .map(([field, [before, after]]) => `${field}: ${String(before)} → ${String(after)}`)
    .join("; ");
}

export interface AuditLogListProps<T extends AuditLogEntry> {
  path: string;
  initialEntries: T[];
  extraColumns?: DataTableColumn<T>[];
}

export function AuditLogList<T extends AuditLogEntry>({ path, initialEntries, extraColumns = [] }: AuditLogListProps<T>) {
  const router = useRouter();
  const [entries, setEntries] = useState(initialEntries);
  const [page, setPage] = useState(1);
  const [hasMore, setHasMore] = useState(initialEntries.length >= PAGE_SIZE);
  const [loading, setLoading] = useState(false);

  async function loadMore() {
    setLoading(true);
    const nextPage = page + 1;
    const result = await callBff<{ audit_logs: T[] }>(`${path}?page=${nextPage}&limit=${PAGE_SIZE}`);
    setLoading(false);
    if (!result.ok) return showApiError(result, (href) => router.push(href), loadMore);
    setEntries((current) => [...current, ...result.data.audit_logs]);
    setPage(nextPage);
    setHasMore(result.data.audit_logs.length >= PAGE_SIZE);
  }

  if (entries.length === 0) return <p className="text-body text-neutral-600">Belum ada aktivitas tercatat.</p>;

  const columns: DataTableColumn<T>[] = [
    { key: "time", header: "Waktu", render: (row) => `${formatDateShort(row.created_at)}, ${formatTimeWIB(row.created_at)}` },
    { key: "actor", header: "Aktor", render: (row) => (row.actor ? `${row.actor.name} (${row.actor.role})` : "Sistem") },
    { key: "action", header: "Aksi", render: (row) => row.action },
    { key: "entity", header: "Entitas", render: (row) => `${row.entity_type} #${row.entity_id}` },
    { key: "changes", header: "Perubahan", render: (row) => changesText(row.changes) },
    ...extraColumns,
  ];

  return (
    <div className="flex flex-col gap-3">
      <DataTable caption="Riwayat aktivitas" columns={columns} rows={entries} rowKey={(row) => row.id} />
      {hasMore && <LoadMore loading={loading} onClick={loadMore} />}
    </div>
  );
}
