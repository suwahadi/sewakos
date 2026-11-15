"use client";

import { SlidersHorizontal } from "lucide-react";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { AuditLogList } from "@/components/features/audit-log-list";
import { CountBadge } from "@/components/ui/badge";
import { Select } from "@/components/ui/select";
import { roleLabel, type AdminAuditLogEntry, type AdminAuditLogFilterOptions } from "@/lib/admin";
import { callBff } from "@/lib/api/browser";
import { showApiError } from "@/lib/api/show-error";
import { cn } from "@/lib/cn";

const PAGE_SIZE = 20;

interface AuditFilters {
  actor_id: string;
  action: string;
}

function filterQuery(filters: AuditFilters): string {
  return new URLSearchParams(Object.entries(filters).filter(([, value]) => value !== "")).toString();
}

export function AdminAuditLogList({
  initialEntries,
  options,
}: {
  initialEntries: AdminAuditLogEntry[];
  options: AdminAuditLogFilterOptions;
}) {
  const router = useRouter();
  const [filters, setFilters] = useState<AuditFilters>({ actor_id: "", action: "" });
  const [result, setResult] = useState({ query: "", entries: initialEntries });
  const [loading, setLoading] = useState(false);
  const [showFilters, setShowFilters] = useState(false);
  const activeCount = Object.values(filters).filter((value) => value !== "").length;

  async function applyFilters(next: AuditFilters) {
    setFilters(next);
    setLoading(true);
    const query = filterQuery(next);
    const response = await callBff<{ audit_logs: AdminAuditLogEntry[] }>(
      `/api/admin/audit-logs?${query ? `${query}&` : ""}page=1&limit=${PAGE_SIZE}`,
    );
    setLoading(false);
    if (!response.ok) return showApiError(response, (href) => router.push(href), () => applyFilters(next));
    setResult({ query, entries: response.data.audit_logs });
  }

  return (
    <div className="flex flex-col gap-3">
      <button
        type="button"
        aria-expanded={showFilters}
        aria-controls="audit-log-filters"
        onClick={() => setShowFilters((current) => !current)}
        className="inline-flex h-9 items-center gap-1.5 self-start rounded-md border border-neutral-300 px-3 text-label font-medium text-neutral-700 lg:hidden"
      >
        <SlidersHorizontal aria-hidden="true" size={16} strokeWidth={1.75} />
        {showFilters ? "Sembunyikan filter" : "Filter"}
        {activeCount > 0 && <CountBadge count={activeCount} />}
      </button>
      <div id="audit-log-filters" className={cn("grid grid-cols-2 gap-3 lg:grid lg:max-w-[640px]", !showFilters && "hidden")}>
        <Select
          label="Aktor"
          value={filters.actor_id}
          disabled={loading}
          onChange={(event) => applyFilters({ ...filters, actor_id: event.target.value })}
        >
          <option value="">Semua aktor</option>
          {options.actors.map((actor) => (
            <option key={actor.id} value={String(actor.id)}>
              {actor.name} ({roleLabel(actor.role)})
            </option>
          ))}
        </Select>
        <Select
          label="Aksi"
          value={filters.action}
          disabled={loading}
          onChange={(event) => applyFilters({ ...filters, action: event.target.value })}
        >
          <option value="">Semua aksi</option>
          {options.actions.map((action) => (
            <option key={action} value={action}>
              {action}
            </option>
          ))}
        </Select>
      </div>
      <div aria-busy={loading} className={cn(loading && "opacity-60")}>
        <AuditLogList
          key={result.query}
          path="/api/admin/audit-logs"
          initialEntries={result.entries}
          query={result.query}
          emptyMessage={result.query ? "Tidak ada aktivitas yang cocok dengan filter." : undefined}
          extraColumns={[
            { key: "ip", header: "IP", render: (row) => row.ip ?? "–" },
            { key: "user_agent", header: "User agent", render: (row) => row.user_agent ?? "–" },
          ]}
        />
      </div>
    </div>
  );
}
