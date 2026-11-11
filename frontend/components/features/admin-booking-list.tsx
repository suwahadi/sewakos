"use client";

import Link from "next/link";
import { Badge } from "@/components/ui/badge";
import { DataTable } from "@/components/ui/data-table";
import { LoadMore } from "@/components/ui/load-more";
import { Select } from "@/components/ui/select";
import { Tabs, type TabItem } from "@/components/ui/tabs";
import type { AdminBooking, AdminInvoice } from "@/lib/admin";
import { formatDateShort } from "@/lib/format";
import { bookingStatusLabel, invoiceStatusLabel, invoiceTypeLabel } from "@/lib/labels";
import { useAdminList } from "@/lib/use-admin-list";

type BookingTab = "" | "pending" | "paid" | "approved" | "completed" | "rejected" | "cancelled";

const BOOKING_TABS: TabItem<BookingTab>[] = [
  { value: "", label: "Semua" },
  ...(["pending", "paid", "approved", "completed", "rejected", "cancelled"] as const).map((status) => ({
    value: status,
    label: bookingStatusLabel(status).label,
  })),
];

export function AdminBookingList({ initial }: { initial: AdminBooking[] }) {
  const list = useAdminList<AdminBooking>("bookings", "bookings", initial);
  const status = (list.query.status ?? "") as BookingTab;

  return (
    <div className="flex flex-col gap-4">
      <h1 className="text-title-lg font-semibold text-neutral-900">Booking</h1>
      <div className="border-b border-neutral-200">
        <Tabs items={BOOKING_TABS} value={status} onChange={(next) => list.filter({ status: next })} aria-label="Status booking" idPrefix="admin-booking" />
      </div>
      {list.rows.length === 0 && !list.loading ? (
        <p className="text-body text-neutral-600">Tidak ada booking dengan status ini.</p>
      ) : (
        <DataTable
          caption="Daftar booking"
          rows={list.rows}
          rowKey={(booking) => booking.id}
          columns={[
            {
              key: "id",
              header: "Booking",
              render: (booking) => (
                <Link href={`/admin/booking/${booking.id}`} className="font-semibold text-brand-700 underline">
                  #{booking.id}
                </Link>
              ),
            },
            { key: "tenant", header: "Penyewa", render: (booking) => booking.tenant.name },
            { key: "room", header: "Kamar", render: (booking) => `${booking.room.name}, ${booking.property.name}` },
            {
              key: "status",
              header: "Status",
              render: (booking) => {
                const label = bookingStatusLabel(booking.status);
                return <Badge color={label.color}>{label.label}</Badge>;
              },
            },
            { key: "start", header: "Mulai", render: (booking) => formatDateShort(booking.start_date) },
            { key: "payout", header: "Dana pemilik", render: (booking) => booking.payout_amount_label, align: "right" },
          ]}
        />
      )}
      {list.hasMore && <LoadMore loading={list.loading} onClick={list.loadMore} />}
    </div>
  );
}

export function AdminInvoiceList({ initial }: { initial: AdminInvoice[] }) {
  const list = useAdminList<AdminInvoice>("invoices", "invoices", initial);

  return (
    <div className="flex flex-col gap-4">
      <h1 className="text-title-lg font-semibold text-neutral-900">Tagihan</h1>
      <div className="grid gap-3 sm:grid-cols-2 lg:max-w-xl">
        <Select label="Status" value={list.query.status ?? ""} onChange={(event) => list.filter({ ...list.query, status: event.target.value })}>
          <option value="">Semua status</option>
          {["unpaid", "paid", "void"].map((status) => (
            <option key={status} value={status}>
              {invoiceStatusLabel(status).label}
            </option>
          ))}
        </Select>
        <Select label="Jenis" value={list.query.type ?? ""} onChange={(event) => list.filter({ ...list.query, type: event.target.value })}>
          <option value="">Semua jenis</option>
          {["initial", "renewal", "extension"].map((type) => (
            <option key={type} value={type}>
              {invoiceTypeLabel(type)}
            </option>
          ))}
        </Select>
      </div>
      {list.rows.length === 0 && !list.loading ? (
        <p className="text-body text-neutral-600">Tidak ada tagihan dengan filter ini.</p>
      ) : (
        <DataTable
          caption="Daftar tagihan"
          rows={list.rows}
          rowKey={(invoice) => invoice.id}
          columns={[
            { key: "number", header: "Nomor", render: (invoice) => invoice.number },
            {
              key: "booking",
              header: "Booking",
              render: (invoice) => (
                <Link href={`/admin/booking/${invoice.booking_id}`} className="text-brand-700 underline">
                  #{invoice.booking_id}
                </Link>
              ),
            },
            { key: "type", header: "Jenis", render: (invoice) => invoiceTypeLabel(invoice.type) },
            {
              key: "status",
              header: "Status",
              render: (invoice) => {
                const label = invoiceStatusLabel(invoice.status);
                return <Badge color={label.color}>{label.label}</Badge>;
              },
            },
            { key: "due", header: "Jatuh tempo", render: (invoice) => formatDateShort(invoice.due_date) },
            { key: "total", header: "Total", render: (invoice) => invoice.total_label, align: "right" },
          ]}
        />
      )}
      {list.hasMore && <LoadMore loading={list.loading} onClick={list.loadMore} />}
    </div>
  );
}
