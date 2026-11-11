"use client";

import { AuditLogList } from "@/components/features/audit-log-list";
import type { AdminAuditLogEntry } from "@/lib/admin";

export function AdminAuditLogList({ initialEntries }: { initialEntries: AdminAuditLogEntry[] }) {
  return (
    <AuditLogList
      path="/api/admin/audit-logs"
      initialEntries={initialEntries}
      extraColumns={[
        { key: "ip", header: "IP", render: (row) => row.ip ?? "–" },
        { key: "user_agent", header: "User agent", render: (row) => row.user_agent ?? "–" },
      ]}
    />
  );
}
