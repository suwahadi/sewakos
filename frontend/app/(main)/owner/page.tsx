import Link from "next/link";
import { AppHeader } from "@/components/features/app-header";
import { OwnerBookingCard } from "@/components/features/owner-booking-card";
import { OwnerPropertyCard } from "@/components/features/owner-property-card";
import { ForbiddenState } from "@/components/ui/error-state";
import { StatCard } from "@/components/ui/stat-card";
import { apiFetch } from "@/lib/api/client";
import { requireAuthenticated } from "@/lib/auth/guard";
import { getNavRole } from "@/lib/auth/nav-role";
import { formatPercent } from "@/lib/format";
import { OWNER_PAGE_SIZE, type OwnerBooking, type OwnerFinanceSummary, type OwnerProperty, type OwnerStats } from "@/lib/owner";

export const metadata = { title: "Beranda pemilik" };

export default async function OwnerHomePage() {
  await requireAuthenticated("/owner");
  const [role, counter, bookings, properties, summary, stats] = await Promise.all([
    getNavRole(),
    apiFetch<{ unread: number }>("/notifications/counter"),
    apiFetch<{ bookings: OwnerBooking[] }>("/owner/bookings", { searchParams: { status: "paid", limit: OWNER_PAGE_SIZE } }),
    apiFetch<{ properties: OwnerProperty[] }>("/owner/properties", { searchParams: { limit: 50 } }),
    apiFetch<{ summary: OwnerFinanceSummary }>("/owner/finance/summary"),
    apiFetch<{ stats: OwnerStats }>("/owner/stats"),
  ]);

  if (role !== "owner") {
    return (
      <>
        <AppHeader variant="inner" title="Beranda" role={role} />
        <ForbiddenState />
      </>
    );
  }

  const waiting = bookings.ok ? bookings.data.bookings : [];
  const ownedProperties = properties.ok ? properties.data.properties : [];

  return (
    <>
      <AppHeader variant="home" role={role} unreadCount={counter.ok ? counter.data.unread : 0} />
      <main className="mx-auto flex max-w-[1200px] flex-col gap-6 px-4 py-6">
        <h1 className="sr-only">Beranda pemilik</h1>
        {(summary.ok || stats.ok) && (
          <section className="grid grid-cols-2 gap-3 lg:grid-cols-4">
            {summary.ok && (
              <>
                <StatCard label="Pendapatan" value={summary.data.summary.earnings_label} />
                <StatCard label="Saldo belum ditransfer" value={summary.data.summary.pending_balance_label} />
                <StatCard label="Deposit dipegang" value={summary.data.summary.deposits_held_label} />
              </>
            )}
            {stats.ok && (
              <StatCard
                label="Okupansi"
                value={formatPercent(stats.data.stats.occupancy.rate)}
                caption={`${stats.data.stats.occupancy.occupied} dari ${stats.data.stats.occupancy.total} kamar`}
              />
            )}
          </section>
        )}
        <section className="flex flex-col gap-3">
          <div className="flex items-center justify-between gap-3">
            <h2 className="text-title-md font-semibold text-neutral-900">Perlu direspons</h2>
            {waiting.length > 0 && (
              <Link href="/owner/booking" className="text-label font-semibold text-brand-700">
                Lihat semua
              </Link>
            )}
          </div>
          {waiting.length > 0 ? (
            <div className="grid gap-3 lg:grid-cols-2">
              {waiting.map((booking) => (
                <OwnerBookingCard key={booking.id} booking={booking} />
              ))}
            </div>
          ) : (
            <p className="text-body text-neutral-600">Belum ada booking yang menunggu responsmu.</p>
          )}
        </section>

        <section className="flex flex-col gap-3">
          <div className="flex items-center justify-between gap-3">
            <h2 className="text-title-md font-semibold text-neutral-900">Properti</h2>
            <Link
              href="/owner/properti/baru"
              className="inline-flex h-11 items-center justify-center rounded-md bg-brand-50 px-4 text-body-strong text-brand-800 hover:bg-brand-100"
            >
              Tambah properti
            </Link>
          </div>
          {ownedProperties.length > 0 ? (
            <div className="grid gap-3 lg:grid-cols-2">
              {ownedProperties.map((property) => (
                <OwnerPropertyCard key={property.id} property={property} />
              ))}
            </div>
          ) : (
            <p className="text-body text-neutral-600">Tambahkan kost pertamamu agar bisa tampil di pencarian.</p>
          )}
        </section>

        <Link href="/owner/ulasan" className="text-label font-semibold text-brand-700">
          Lihat ulasan penyewa
        </Link>
      </main>
    </>
  );
}
