"use client";

import Link from "next/link";
import { useState } from "react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { DataTable } from "@/components/ui/data-table";
import { LoadMore } from "@/components/ui/load-more";
import { Tabs, type TabItem } from "@/components/ui/tabs";
import { TextField } from "@/components/ui/text-field";
import type { AdminProperty } from "@/lib/admin";
import { formatDateShort } from "@/lib/format";
import { moderationStatusLabel } from "@/lib/labels";
import { useAdminList } from "@/lib/use-admin-list";

type StatusTab = "" | "pending" | "approved" | "rejected" | "suspended";

const TABS: TabItem<StatusTab>[] = [
  { value: "pending", label: "Sedang ditinjau" },
  { value: "approved", label: "Tayang" },
  { value: "rejected", label: "Ditolak" },
  { value: "suspended", label: "Dinonaktifkan" },
  { value: "", label: "Semua" },
];

export function AdminPropertyList({ initial }: { initial: AdminProperty[] }) {
  const list = useAdminList<AdminProperty>("properties", "properties", initial, { moderation_status: "pending" });
  const [search, setSearch] = useState("");
  const status = (list.query.moderation_status ?? "") as StatusTab;

  return (
    <div className="flex flex-col gap-4">
      <h1 className="text-title-lg font-semibold text-neutral-900">Moderasi properti</h1>
      <div className="border-b border-neutral-200">
        <Tabs items={TABS} value={status} onChange={(next) => list.filter({ ...list.query, moderation_status: next })} aria-label="Status moderasi" idPrefix="admin-property" />
      </div>
      <form
        className="flex items-end gap-3"
        onSubmit={(event) => {
          event.preventDefault();
          list.filter({ ...list.query, q: search.trim() });
        }}
      >
        <div className="min-w-0 flex-1">
          <TextField label="Cari nama kost" value={search} onChange={(event) => setSearch(event.target.value)} />
        </div>
        <Button type="submit" variant="outline">
          Cari
        </Button>
      </form>
      {list.rows.length === 0 && !list.loading ? (
        <p className="text-body text-neutral-600">Tidak ada properti dengan status ini.</p>
      ) : (
        <DataTable
          caption="Daftar properti"
          rows={list.rows}
          rowKey={(property) => property.id}
          columns={[
            {
              key: "name",
              header: "Properti",
              render: (property) => (
                <Link href={`/admin/properti/${property.id}`} className="font-semibold text-brand-700 underline">
                  {property.name}
                </Link>
              ),
            },
            { key: "owner", header: "Pemilik", render: (property) => property.owner.name },
            { key: "area", header: "Area", render: (property) => property.area.name },
            {
              key: "status",
              header: "Status",
              render: (property) => {
                const label = moderationStatusLabel(property.moderation_status);
                return <Badge color={label.color}>{label.label}</Badge>;
              },
            },
            { key: "created", header: "Dibuat", render: (property) => formatDateShort(property.created_at) },
          ]}
        />
      )}
      {list.hasMore && <LoadMore loading={list.loading} onClick={list.loadMore} />}
    </div>
  );
}
