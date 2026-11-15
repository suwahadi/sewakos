"use client";

import { ChevronDown } from "lucide-react";
import { useRouter } from "next/navigation";
import { useId, useState } from "react";
import { DataTable, type DataTableColumn } from "@/components/ui/data-table";
import { LoadMore } from "@/components/ui/load-more";
import { callBff } from "@/lib/api/browser";
import { showApiError } from "@/lib/api/show-error";
import { formatDateShort, formatTimeWIB } from "@/lib/format";
import type { AuditLogEntry } from "@/lib/owner";

const PAGE_SIZE = 20;

function changeLines(changes: AuditLogEntry["changes"]): string[] {
  return Object.entries(changes ?? {}).map(([field, [before, after]]) => `${field}: ${String(before)} → ${String(after)}`);
}

function ChangesToggle({ changes }: { changes: AuditLogEntry["changes"] }) {
  const [open, setOpen] = useState(false);
  const panelId = useId();
  const lines = changeLines(changes);
  if (lines.length === 0) return <>–</>;

  return (
    <div className="flex flex-col items-end gap-1 lg:items-start">
      <button
        type="button"
        aria-expanded={open}
        aria-controls={panelId}
        onClick={() => setOpen((current) => !current)}
        className="inline-flex items-center gap-1 font-semibold text-brand-700"
      >
        {open ? "Sembunyikan" : "Selengkapnya"}
        <ChevronDown aria-hidden="true" size={14} strokeWidth={2} className={open ? "rotate-180" : undefined} />
      </button>
      <ul id={panelId} hidden={!open} className="flex flex-col gap-0.5 text-left text-neutral-700">
        {lines.map((line) => (
          <li key={line} className="break-words">
            {line}
          </li>
        ))}
      </ul>
    </div>
  );
}

export interface AuditLogListProps<T extends AuditLogEntry> {
  path: string;
  initialEntries: T[];
  extraColumns?: DataTableColumn<T>[];
  query?: string;
  emptyMessage?: string;
}

export function AuditLogList<T extends AuditLogEntry>({
  path,
  initialEntries,
  extraColumns = [],
  query = "",
  emptyMessage = "Belum ada aktivitas tercatat.",
}: AuditLogListProps<T>) {
  const router = useRouter();
  const [entries, setEntries] = useState(initialEntries);
  const [page, setPage] = useState(1);
  const [hasMore, setHasMore] = useState(initialEntries.length >= PAGE_SIZE);
  const [loading, setLoading] = useState(false);

  async function loadMore() {
    setLoading(true);
    const nextPage = page + 1;
    const result = await callBff<{ audit_logs: T[] }>(`${path}?${query ? `${query}&` : ""}page=${nextPage}&limit=${PAGE_SIZE}`);
    setLoading(false);
    if (!result.ok) return showApiError(result, (href) => router.push(href), loadMore);
    setEntries((current) => [...current, ...result.data.audit_logs]);
    setPage(nextPage);
    setHasMore(result.data.audit_logs.length >= PAGE_SIZE);
  }

  if (entries.length === 0) return <p className="text-body text-neutral-600">{emptyMessage}</p>;

  const columns: DataTableColumn<T>[] = [
    { key: "time", header: "Waktu", render: (row) => `${formatDateShort(row.created_at)}, ${formatTimeWIB(row.created_at)}` },
    { key: "actor", header: "Aktor", render: (row) => (row.actor ? `${row.actor.name} (${row.actor.role})` : "Sistem") },
    { key: "action", header: "Aksi", render: (row) => row.action },
    { key: "entity", header: "Entitas", render: (row) => `${row.entity_type} #${row.entity_id}` },
    { key: "changes", header: "Perubahan", render: (row) => <ChangesToggle changes={row.changes} /> },
    ...extraColumns,
  ];

  return (
    <div className="flex flex-col gap-3">
      <DataTable caption="Riwayat aktivitas" size="compact" columns={columns} rows={entries} rowKey={(row) => row.id} />
      {hasMore && <LoadMore loading={loading} onClick={loadMore} />}
    </div>
  );
}
