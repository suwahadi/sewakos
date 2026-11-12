"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { DataTable } from "@/components/ui/data-table";
import { ConfirmDialog } from "@/components/ui/dialog";
import { LoadMore } from "@/components/ui/load-more";
import { Select } from "@/components/ui/select";
import { TextField } from "@/components/ui/text-field";
import { roleLabel, type AdminUser } from "@/lib/admin";
import { callBff } from "@/lib/api/browser";
import { showApiError } from "@/lib/api/show-error";
import { formatPhoneDisplay } from "@/lib/format";
import { toast } from "@/lib/toast";
import { useAdminList } from "@/lib/use-admin-list";

export function AdminUserList({ initial }: { initial: AdminUser[] }) {
  const router = useRouter();
  const list = useAdminList<AdminUser>("users", "users", initial);
  const [search, setSearch] = useState("");
  const [searchError, setSearchError] = useState<string>();
  const [target, setTarget] = useState<AdminUser | null>(null);
  const [saving, setSaving] = useState(false);

  async function toggle(user: AdminUser) {
    setSaving(true);
    const action = user.suspended_at ? "unsuspend" : "suspend";
    const result = await callBff(`/api/admin/users/${user.id}/${action}`, { method: "POST" });
    setSaving(false);
    setTarget(null);
    if (!result.ok) return showApiError(result, (href) => router.push(href), () => toggle(user));
    toast.success(result.data.meta.message);
    list.reload();
  }

  return (
    <div className="flex flex-col gap-4">
      <h1 className="text-title-lg font-semibold text-neutral-900">Pengguna</h1>
      <form
        className="grid gap-3 sm:grid-cols-[200px_minmax(0,1fr)_auto] sm:items-start"
        onSubmit={(event) => {
          event.preventDefault();
          const q = search.trim();
          if (q && q.length < 3) return setSearchError("Masukkan minimal 3 karakter.");
          setSearchError(undefined);
          list.filter({ ...list.query, q });
        }}
      >
        <Select label="Peran" value={list.query.role ?? ""} onChange={(event) => list.filter({ ...list.query, role: event.target.value })}>
          <option value="">Semua peran</option>
          {["tenant", "owner", "admin"].map((role) => (
            <option key={role} value={role}>
              {roleLabel(role)}
            </option>
          ))}
        </Select>
        <TextField label="Cari nama atau nomor HP" value={search} onChange={(event) => setSearch(event.target.value)} error={searchError} />
        <Button type="submit" className="sm:mt-[26px]">
          Cari
        </Button>
      </form>
      {list.rows.length === 0 && !list.loading ? (
        <p className="text-body text-neutral-600">Tidak ada pengguna yang cocok.</p>
      ) : (
        <DataTable
          caption="Daftar pengguna"
          rows={list.rows}
          rowKey={(user) => user.id}
          columns={[
            { key: "name", header: "Nama", render: (user) => user.name },
            { key: "role", header: "Peran", render: (user) => roleLabel(user.role) },
            { key: "phone", header: "Nomor HP", render: (user) => formatPhoneDisplay(user.phone_number) },
            {
              key: "status",
              header: "Status",
              render: (user) => (user.suspended_at ? <Badge color="danger">Dinonaktifkan</Badge> : <Badge color="success">Aktif</Badge>),
            },
            {
              key: "action",
              header: "Aksi",
              render: (user) =>
                user.role === "admin" ? (
                  "–"
                ) : (
                  <Button size="sm" variant={user.suspended_at ? "success" : "danger"} onClick={() => setTarget(user)}>
                    {user.suspended_at ? "Aktifkan" : "Nonaktifkan"}
                  </Button>
                ),
            },
          ]}
        />
      )}
      {list.hasMore && <LoadMore loading={list.loading} onClick={list.loadMore} />}
      {target && (
        <ConfirmDialog
          open
          onClose={() => setTarget(null)}
          title={target.suspended_at ? `Aktifkan ${target.name}?` : `Nonaktifkan ${target.name}?`}
          description={
            target.suspended_at
              ? "Pengguna bisa masuk kembali seperti biasa."
              : "Semua sesi pengguna diakhiri dan ia tidak bisa masuk. Properti miliknya berhenti tayang."
          }
          confirmLabel={target.suspended_at ? "Aktifkan" : "Nonaktifkan"}
          destructive={!target.suspended_at}
          loading={saving}
          onConfirm={() => toggle(target)}
        />
      )}
    </div>
  );
}
