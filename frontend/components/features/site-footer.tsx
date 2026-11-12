import Link from "next/link";

const FOOTER_LINKS = [
  { href: "/cari", label: "Cari kost" },
  { href: "/syarat-ketentuan", label: "Syarat & Ketentuan" },
  { href: "/kebijakan-privasi", label: "Kebijakan Privasi" },
];

export function SiteFooter() {
  return (
    <footer className="mt-auto border-t border-neutral-200 bg-neutral-50">
      <div className="mx-auto flex max-w-[1200px] flex-col gap-6 px-4 pb-6 pt-8 lg:pt-10">
        <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
          <div className="flex flex-col gap-1">
            <Link href="/" className="self-start text-title-md font-bold text-brand-700">
              Sewa Kos
            </Link>
            <p className="text-caption text-neutral-600">Cari, pesan, dan bayar sewa kost secara online.</p>
          </div>
          <nav aria-label="Tautan footer">
            <ul className="-mx-3 flex flex-wrap">
              {FOOTER_LINKS.map((link) => (
                <li key={link.href}>
                  <Link
                    href={link.href}
                    className="inline-flex min-h-11 items-center rounded-md px-3 text-body text-neutral-700 hover:text-brand-700"
                  >
                    {link.label}
                  </Link>
                </li>
              ))}
            </ul>
          </nav>
        </div>
        <p className="border-t border-neutral-200 pt-6 text-center text-caption font-light text-neutral-600">
          Copyright &copy; 2026 - All Rights Reserved
        </p>
      </div>
    </footer>
  );
}
