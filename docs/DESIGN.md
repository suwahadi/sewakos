# DESIGN.md — Design System Kos (Web, mobile-first)

> Dokumen ini untuk **Claude Code / AI coding agent** dan desainer. Letakkan di `docs/` bersama dokumen kontrak API.
> Proses kerja (git, gaya kode, pemeriksaan) diatur `AGENT.md`; urutan kerja di `IMPLEMENTATION.md`.
> Path kode di dokumen ini relatif terhadap folder `frontend/`. Design system ini **orisinal**; jangan meniru tampilan,
> logo, atau aset merek aplikasi lain.

## 0. Aturan untuk agent

1. Hanya gunakan token di §3. Dilarang menulis warna hex, ukuran px, durasi, atau bayangan langsung di komponen.
2. Hanya buat komponen yang ada di §6. Butuh komponen baru → tambahkan spesifikasinya ke dokumen ini dulu.
3. Semua data di UI berasal dari field API yang disebut di spesifikasi komponen. Jangan mengarang field.
   Field yang dibutuhkan tapi belum ada di kontrak → tambahkan ke dokumen API dulu (§11), jangan diakali di frontend.
4. Uang selalu ditampilkan dari field `*_label` API. Format sendiri hanya untuk nilai yang belum punya label (§8.3).
5. Semua teks berbahasa Indonesia, *sentence case*, mengikuti §9.
6. Setiap komponen interaktif wajib lolos checklist aksesibilitas §10.
7. Jangan menambah library UI, animasi, atau ikon selain yang ada di §2.
8. Aturan wajib di §0.1 berlaku untuk **semua** komponen dan halaman, termasuk yang tidak menyebutnya ulang.

### 0.1 Aturan wajib UI

**A. Kursor pointer di semua elemen yang bisa diklik.**
Tailwind v4 mengembalikan kursor tombol ke `default`, jadi aturan ini dipasang global di §3.10:
`a[href]`, `button`, `summary`, `label[for]`, `select`, checkbox/radio/date input, dan elemen dengan
`role="button" | "tab" | "radio" | "switch" | "option" | "menuitem" | "checkbox"`.
Elemen nonaktif (`disabled`, `aria-disabled="true"`) memakai `cursor: not-allowed`.
Kartu yang seluruhnya tautan otomatis ikut karena dibungkus `<a>`. Jangan menambah `cursor-pointer` ke elemen yang
tidak bisa diklik.

**B. Dua tombol dalam satu baris selalu sama lebar (50:50).**
- Wadah `grid grid-cols-2 gap-3`; kedua tombol `w-full`, ukuran sama. Jangan pakai lebar otomatis mengikuti teks.
- Label dibuat pendek agar muat di 360px (maks ±16 karakter). Bila tidak muat, susun bertumpuk (satu kolom, lebar penuh),
  bukan memperkecil font.
- Urutan: aksi sekunder di **kiri**, aksi utama di **kanan**. Saat bertumpuk: aksi utama di **atas**.
- **Tidak menambah ikon** ke tombol hanya agar terlihat ramai. Ikon di tombol hanya boleh bila menjelaskan aksi
  yang tidak cukup jelas dari teks, dan sudah ditentukan di spesifikasi komponen:
  `Download` (unduh PDF/CSV), `ImagePlus` (tambah foto), `RefreshCw` (coba lagi), `SlidersHorizontal` (filter),
  `ArrowUpDown` (urutkan), logo WhatsApp & `MessageSquareText` (pilih kanal OTP).
  Dalam pasangan 50:50, kedua tombol sama-sama tanpa ikon, atau sama-sama berikon.

**C. Ikon merek harus logo resmi.**
- Logo merek pihak ketiga (WhatsApp, Google, dll.) **hanya** dari file SVG resmi di halaman *brand resources*
  pemilik merek, disimpan di `/public/brand/` apa adanya: tidak digambar ulang, tidak diubah warna/proporsi,
  dan mengikuti aturan ruang kosong dari pedoman merek tersebut.
- **Dilarang** mengganti logo merek dengan ikon generik (mis. `MessageCircle` atau `Phone` untuk WhatsApp).
  Bila file resmi belum tersedia, tampilkan **teks** nama mereknya saja.
- Lucide tidak dipakai untuk logo merek.
- Logo metode pembayaran tidak ditampilkan di aplikasi; halaman Snap Midtrans sudah menampilkannya.

**D. Dua kolom selalu seimbang dan sama tinggi.**
- Grid dua kolom (atau lebih) memakai `grid` dengan `items-stretch` (default), setiap kartu `h-full`.
- Isi kartu disusun `flex flex-col`; bagian bawah (harga, aksi, angka) didorong ke dasar dengan `mt-auto`,
  sehingga harga dan tombol di kartu bersebelahan selalu sejajar.
- Teks yang panjangnya bervariasi dibatasi jumlah barisnya (`line-clamp-*`) agar tinggi tidak melompat.
- Gambar di kartu bersebelahan memakai rasio yang sama.
- Kolom isi di desktop (mis. detail kost: konten + panel harga) dibagi dengan proporsi tetap dari §3.8.

**E. Maksimalkan ruang di mobile.**
- Filter & urutkan di halaman hasil **disembunyikan** di balik dua tombol (50:50) "Filter" dan "Urutkan" yang membuka
  sheet. Chip filter aktif hanya muncul bila ada filter aktif, dalam satu baris yang bisa digeser.
- Bilah filter menempel di bawah header; saat pengguna menggulir ke bawah, header + bilah filter menyembunyikan diri
  (`translateY(-100%)`) dan muncul lagi saat menggulir ke atas.
- Kartu listing di mobile (< 640px) berbentuk **horizontal** (foto kiri, info kanan) agar lebih banyak kost terlihat
  per layar (§6.28).
- Ukuran teks mengikuti skala mobile §3.5; tidak ada teks judul > 22px di mobile.
- Padding halaman 16px, jarak antar section 24px, padding kartu 12px di mobile. Tidak ada blok ruang kosong dekoratif.
- Deskripsi panjang dipotong 3 baris + "Selengkapnya"; daftar panjang dibatasi + "Lihat semua".
- **Tidak boleh ada scroll horizontal di tingkat halaman** pada lebar 320–430px:
  semua anak flex memakai `min-w-0`, teks panjang (alamat, nama, nomor rekening) memakai `break-words`,
  gambar `max-w-full`, tabel & deretan chip di-scroll di dalam wadahnya sendiri, dan `html, body { overflow-x: clip; }`
  **tidak** dipakai untuk menutupi elemen yang kebablasan — perbaiki elemennya.
- Elemen yang menempel (header, bilah filter, sticky action bar, bottom nav) totalnya tidak boleh menutupi lebih dari
  ±30% tinggi layar 640px.

## 1. Prinsip

| Prinsip | Artinya di praktik |
|---|---|
| **Harga dulu** | Harga adalah keputusan utama penyewa. Di setiap kartu dan halaman detail, harga adalah teks paling tebal dan mudah dicari. Ini satu-satunya elemen yang diberi perlakuan tipografi berani. |
| **Mobile dulu** | Desain dimulai dari layar 360px. Desktop adalah perluasan (lebih banyak kolom), bukan desain terpisah. |
| **Ringan** | Tanpa UI kit berat, tanpa library animasi. CSS dulu, JavaScript hanya bila perlu. |
| **Tenang** | Latar putih dan abu netral; oranye hanya untuk aksi utama, harga promo, dan status aktif. Tidak ada gradien dekoratif. |
| **Gerak menjawab aksi** | Animasi hanya muncul sebagai respons aksi pengguna (pindah halaman, buka accordion, toast, sheet). Tidak ada animasi masuk per section. |
| **Jujur soal status** | Status booking, tagihan, dan pembayaran selalu terlihat dengan label teks, bukan hanya warna. |

## 2. Stack & setup

| Kebutuhan | Pilihan | Catatan |
|---|---|---|
| Framework | Next.js (App Router), versi stabil terbaru saat implementasi | Server Components untuk halaman publik (SEO listing) |
| Styling | Tailwind CSS v4 | Token didefinisikan di `@theme` (§3.9) |
| Font | Poppins lewat `next/font/google` | Bobot **400, 500, 600, 700** saja, subset `latin`, `display: "swap"` |
| Ikon | `lucide-react` (lisensi ISC) | Hanya ikon yang terdaftar di §4 |
| Utilitas class | `clsx` + `tailwind-merge` (fungsi `cn()`) | |
| Gambar | `next/image` | `remotePatterns` diisi host `STORAGE_PUBLIC_BASE_URL` |
| Pembayaran | Snap.js Midtrans | Dimuat hanya di halaman pembayaran (§6.40) |
| Lint | ESLint dengan `eslint-config-next` | Script `npm run lint` |
| Unit/komponen test | Vitest + `@testing-library/react` + `jsdom` | Script `npm run test` |
| End-to-end | Playwright (Chromium, viewport 360×640 dan 1280×800) | Script `npm run e2e`, dijalankan terhadap backend lokal + seed |

Script wajib di `package.json`: `dev`, `build`, `start`, `typecheck` (`tsc --noEmit`), `lint`, `test`, `e2e`.

**Dilarang:** framer-motion/GSAP, komponen UI kit siap pakai (MUI, Chakra, Ant), icon font, gambar ikon PNG.

**Anggaran performa (mobile, 4G lambat):**
- JavaScript first-load per route ≤ 150 KB (gzip).
- LCP ≤ 2,5 detik di halaman beranda, hasil cari, dan detail.
- CLS ≤ 0,1: semua gambar punya rasio tetap (§3.8), skeleton seukuran konten aslinya.

**Token sesi:** simpan token API di cookie `httpOnly` lewat route handler Next.js, bukan di `localStorage`.
Header `x-api-client` dan `Authorization` ditambahkan di sisi server.

---

## 3. Design token

### 3.1 Warna — brand (oranye koral)

| Token | Hex | Pemakaian |
|---|---|---|
| `brand-50` | `#FFF4EF` | Latar chip/kartu terpilih |
| `brand-100` | `#FFE6DA` | Latar badge promo, hover baris terpilih |
| `brand-200` | `#FFC9B0` | Border elemen terpilih |
| `brand-300` | `#FFA27C` | Ikon di atas latar gelap |
| `brand-500` | `#F2592A` | Aksen dekoratif & teks ≥ 24px saja (kontras putih 3,37:1) |
| `brand-600` | `#CC4316` | **Tombol utama**, fokus ring, tab aktif (kontras putih 4,79:1) |
| `brand-700` | `#B33812` | Teks link, harga promo, hover tombol utama (6,02:1) |
| `brand-800` | `#8C2D10` | Tombol utama saat ditekan, teks di atas `brand-100` (7,04:1) |

### 3.2 Warna — netral

| Token | Hex | Pemakaian |
|---|---|---|
| `neutral-0` | `#FFFFFF` | Latar halaman & kartu |
| `neutral-50` | `#F7F7F8` | Latar section, latar input disabled |
| `neutral-100` | `#EFEFF1` | Skeleton, latar badge netral, placeholder gambar |
| `neutral-200` | `#E2E2E6` | Divider, border kartu (dekoratif) |
| `neutral-300` | `#C9C9D0` | Border disabled |
| `neutral-400` | `#8E8E99` | **Border input** (3,24:1, memenuhi 3:1 untuk komponen UI) |
| `neutral-500` | `#70707C` | Teks sekunder & placeholder (4,89:1) |
| `neutral-600` | `#555561` | Teks label, ikon default (7,35:1) |
| `neutral-700` | `#3D3D47` | Teks isi panjang |
| `neutral-800` | `#26262E` | Latar toast |
| `neutral-900` | `#16161C` | Judul & teks utama (18:1) |

Teks disabled memakai `neutral-400` dan tidak wajib memenuhi kontras (pengecualian WCAG untuk elemen nonaktif).

### 3.3 Warna — semantik

| Token | Teks/ikon | Latar | Kontras | Pemakaian |
|---|---|---|---|---|
| `success` | `#1E7A4F` | `#E8F6EF` | 4,77:1 | Lunas, aktif, berhasil |
| `warning` | `#9A4A06` | `#FFF6E5` | 5,83:1 | Menunggu pembayaran, sisa kamar sedikit |
| `danger` | `#B42318` | `#FDECEC` | 5,76:1 | Ditolak, gagal, error |
| `info` | `#1A5FB8` | `#EAF2FD` | 5,52:1 | Menunggu konfirmasi, diproses |

Tombol bahaya: latar `danger` (`#B42318`) dengan teks putih (6,57:1).

**Ikon di toast gelap** (latar `neutral-800`): success `#5BD49A` (8,1:1), warning `#F5B841` (8,44:1),
danger `#FF8A80` (6,57:1), info `#7CB4FF` (7,03:1).

**Bintang rating:** `#BF7A00` (3,5:1 di putih) untuk bintang terisi; `neutral-300` untuk bintang kosong.

### 3.4 Warna — gender properti

Gender **selalu** ditampilkan sebagai teks dalam badge; warna hanya penanda tambahan.

| Nilai API `gender` | Label | Teks | Latar | Kontras |
|---|---|---|---|---|
| `putra` | Putra | `#1A5FB8` | `#EAF2FD` | 5,52:1 |
| `putri` | Putri | `#B0174F` | `#FCE8F0` | 5,82:1 |
| `campur` | Campur | `#5B3FB8` | `#F0ECFC` | 6,36:1 |

### 3.5 Tipografi

Satu keluarga: **Poppins**. Fallback: `ui-sans-serif, system-ui, -apple-system, "Segoe UI", Roboto, sans-serif`.
Poppins cukup lebar, jadi ukuran isi di mobile 14px dengan line-height lega.

| Token | Mobile (size/line-height) | ≥ 1024px | Bobot | Pemakaian |
|---|---|---|---|---|
| `display` | 22 / 30 | 32 / 40 | 700 | Judul beranda (satu per halaman) |
| `title-lg` | 18 / 26 | 24 / 32 | 600 | Judul halaman, nama properti di detail |
| `title-md` | 16 / 24 | 18 / 26 | 600 | Judul section, judul sheet/dialog |
| `title-sm` | 14 / 20 | 14 / 20 | 600 | Nama properti di kartu, judul item list |
| `body` | 14 / 22 | 15 / 24 | 400 | Teks isi |
| `body-strong` | 14 / 22 | 15 / 24 | 500 | Penekanan dalam teks, label tombol |
| `caption` | 12 / 18 | 12 / 18 | 400 | Info sekunder, waktu, helper text |
| `caption-strong` | 12 / 16 | 12 / 16 | 600 | Badge |
| `label` | 13 / 18 | 13 / 18 | 500 (600 di tombol kecil) | Label input, chip, tombol `sm`, label bottom nav |
| `price` | 18 / 24 | 20 / 28 | 700 | **Harga di kartu & sticky bar** |
| `price-lg` | 24 / 32 | 28 / 36 | 700 | Total di rincian harga |

Aturan:
- Tidak ada teks huruf kapital semua (all caps) dan tidak ada letter-spacing tambahan.
- Panjang baris teks isi maksimal 72 karakter (`max-w-prose` disesuaikan ke `68ch`).
- Angka uang dan waktu hitung mundur memakai `font-variant-numeric: tabular-nums` agar tidak "goyang".
- Satuan harga (`/bulan`, `/minggu`, `/hari`) ditulis `caption` warna `neutral-500` tepat setelah harga.

### 3.6 Spasi

Basis 4px. Hanya nilai ini yang dipakai: `0, 1 (4px), 2 (8px), 3 (12px), 4 (16px), 5 (20px), 6 (24px), 8 (32px), 10 (40px), 12 (48px), 16 (64px)`.

| Konteks | Nilai |
|---|---|
| Padding horizontal halaman | mobile 16px, ≥ 768px 24px, ≥ 1280px 32px |
| Jarak antar section | mobile 24px, desktop 48px |
| Padding dalam kartu/panel | mobile 12px, ≥ 768px 16px |
| Jarak antar field form | 16px |
| Jarak label ke input | 6px (satu-satunya pengecualian, `gap-1.5`) |
| Gap grid kartu listing | mobile 16px, desktop 24px |

### 3.7 Radius, border, elevasi

Radius dibedakan menurut hierarki, bukan disamakan:

| Token | Nilai | Pemakaian |
|---|---|---|
| `radius-sm` | 6px | Badge, chip kecil, checkbox |
| `radius-md` | 10px | Input, tombol, gambar kartu listing |
| `radius-lg` | 16px | Panel, dialog, kartu ringkasan |
| `radius-sheet` | 20px | Sudut atas bottom sheet |
| `radius-full` | 9999px | Chip filter, avatar, tombol ikon bulat, badge hitungan |

Border: 1px. Kartu listing **tidak** memakai border maupun bayangan; batasnya dibentuk gambar dan ruang kosong.
Panel informasi (rincian harga, status booking) memakai border `neutral-200`.

| Token elevasi | Nilai | Pemakaian |
|---|---|---|
| `shadow-raised` | `0 1px 2px rgb(22 22 28 / 0.06), 0 2px 8px rgb(22 22 28 / 0.06)` | Sticky action bar, header saat di-scroll |
| `shadow-overlay` | `0 8px 24px rgb(22 22 28 / 0.14), 0 2px 6px rgb(22 22 28 / 0.08)` | Dialog, dropdown saran, toast |
| `shadow-sheet` | `0 -4px 24px rgb(22 22 28 / 0.12)` | Bottom sheet |

Backdrop overlay: `rgb(22 22 28 / 0.48)`.

### 3.8 Layout, breakpoint, z-index

Breakpoint mengikuti default Tailwind: `sm 640`, `md 768`, `lg 1024`, `xl 1280`. Lebar kontainer maks **1200px**, rata tengah.
Di grup halaman utama batas ini dipasang sekali di template (`.page-container`); halaman boleh mempersempit dengan
`max-w-*` sendiri.
Konten rata kiri; rata tengah hanya untuk empty state dan halaman auth.
Dua kolom desktop (detail kost, form sewa, detail booking): konten 2/3 + panel 1/3 (`grid-cols-[2fr_1fr]`, gap 32px).

| Rasio gambar | Pemakaian |
|---|---|
| 4:3 | Gambar kartu listing, galeri detail di mobile, thumbnail foto owner |
| 16:9 | Galeri detail di ≥ 1024px |
| 1:1 | Avatar, thumbnail favorit di list |

| Token z-index | Nilai | Elemen |
|---|---|---|
| `z-sticky` | 20 | Header, sticky action bar, bottom nav |
| `z-dropdown` | 30 | Daftar saran pencarian |
| `z-overlay` | 40 | Backdrop, bottom sheet, dialog |
| `z-toast` | 50 | Toaster (selalu paling atas) |

Area aman perangkat: elemen yang menempel di bawah layar menambah `padding-bottom: env(safe-area-inset-bottom)`.
Tinggi bottom nav: 64px + safe area. Target sentuh minimum: **44 × 44px**.

### 3.9 Gerak (motion)

| Token | Nilai | Pemakaian |
|---|---|---|
| `duration-fast` | 120ms | Hover, tekan tombol, ganti warna |
| `duration-base` | 200ms | Accordion, toast masuk, transisi halaman |
| `duration-slow` | 280ms | Bottom sheet, dialog |
| `ease-standard` | `cubic-bezier(0.2, 0, 0, 1)` | Hampir semua transisi |
| `ease-exit` | `cubic-bezier(0.4, 0, 1, 1)` | Elemen yang keluar/menutup |

Hanya `opacity` dan `transform` yang dianimasikan, kecuali accordion (`grid-template-rows`, §5.2).

### 3.10 File token (`app/globals.css`)

```css
@import "tailwindcss";

@theme {
  --color-brand-50: #FFF4EF;
  --color-brand-100: #FFE6DA;
  --color-brand-200: #FFC9B0;
  --color-brand-300: #FFA27C;
  --color-brand-500: #F2592A;
  --color-brand-600: #CC4316;
  --color-brand-700: #B33812;
  --color-brand-800: #8C2D10;

  --color-neutral-0: #FFFFFF;
  --color-neutral-50: #F7F7F8;
  --color-neutral-100: #EFEFF1;
  --color-neutral-200: #E2E2E6;
  --color-neutral-300: #C9C9D0;
  --color-neutral-400: #8E8E99;
  --color-neutral-500: #70707C;
  --color-neutral-600: #555561;
  --color-neutral-700: #3D3D47;
  --color-neutral-800: #26262E;
  --color-neutral-900: #16161C;

  --color-success: #1E7A4F;   --color-success-bg: #E8F6EF;  --color-success-on-dark: #5BD49A;
  --color-warning: #9A4A06;   --color-warning-bg: #FFF6E5;  --color-warning-on-dark: #F5B841;
  --color-danger: #B42318;    --color-danger-bg: #FDECEC;   --color-danger-on-dark: #FF8A80;
  --color-info: #1A5FB8;      --color-info-bg: #EAF2FD;     --color-info-on-dark: #7CB4FF;
  --color-star: #BF7A00;

  --color-putra: #1A5FB8;   --color-putra-bg: #EAF2FD;
  --color-putri: #B0174F;   --color-putri-bg: #FCE8F0;
  --color-campur: #5B3FB8;  --color-campur-bg: #F0ECFC;

  --font-sans: var(--font-poppins), ui-sans-serif, system-ui, -apple-system, "Segoe UI", Roboto, sans-serif;

  /* Mobile sizes; desktop overrides live in the @layer base media query below. */
  --text-display: 1.375rem;   --text-display--line-height: 1.875rem;
  --text-title-lg: 1.125rem;  --text-title-lg--line-height: 1.625rem;
  --text-title-md: 1rem;      --text-title-md--line-height: 1.5rem;
  --text-title-sm: 0.875rem;  --text-title-sm--line-height: 1.25rem;
  --text-body: 0.875rem;      --text-body--line-height: 1.375rem;
  --text-caption: 0.75rem;    --text-caption--line-height: 1.125rem;
  --text-label: 0.8125rem;    --text-label--line-height: 1.125rem;
  --text-price: 1.125rem;     --text-price--line-height: 1.5rem;
  --text-price-lg: 1.5rem;    --text-price-lg--line-height: 2rem;

  --radius-sm: 6px;
  --radius-md: 10px;
  --radius-lg: 16px;
  --radius-sheet: 20px;

  --shadow-raised: 0 1px 2px rgb(22 22 28 / 0.06), 0 2px 8px rgb(22 22 28 / 0.06);
  --shadow-overlay: 0 8px 24px rgb(22 22 28 / 0.14), 0 2px 6px rgb(22 22 28 / 0.08);
  --shadow-sheet: 0 -4px 24px rgb(22 22 28 / 0.12);

  --ease-standard: cubic-bezier(0.2, 0, 0, 1);
  --ease-exit: cubic-bezier(0.4, 0, 1, 1);
  --animate-page-enter: page-enter 200ms var(--ease-standard) backwards;
  --animate-toast-in: toast-in 200ms var(--ease-standard) both;
  --animate-toast-out: toast-out 150ms var(--ease-exit) both;
  --animate-shimmer: shimmer 1.4s linear infinite;

  @keyframes page-enter {
    from { opacity: 0; transform: translateY(8px); }
    to   { opacity: 1; transform: none; }
  }
  @keyframes toast-in {
    from { opacity: 0; transform: translateY(8px) scale(0.98); }
    to   { opacity: 1; transform: none; }
  }
  @keyframes toast-out {
    from { opacity: 1; transform: none; }
    to   { opacity: 0; transform: translateY(4px) scale(0.98); }
  }
  @keyframes shimmer {
    from { background-position: 200% 0; }
    to   { background-position: -200% 0; }
  }
}

@layer base {
  html { color-scheme: light; -webkit-tap-highlight-color: transparent; }
  body {
    @apply bg-neutral-0 font-sans text-body text-neutral-900 antialiased;
  }
  :focus-visible { outline: 2px solid var(--color-brand-600); outline-offset: 2px; }

  /* Tailwind v4 preflight resets button cursors to default; restore pointer for every clickable element. */
  a[href], button, summary, label[for], select,
  input[type="checkbox"], input[type="radio"], input[type="date"],
  [role="button"], [role="tab"], [role="radio"], [role="switch"],
  [role="option"], [role="menuitem"], [role="checkbox"] {
    cursor: pointer;
  }
  :disabled, [aria-disabled="true"] { cursor: not-allowed; }

  @media (min-width: 1024px) {
    :root {
      --text-display: 2rem;       --text-display--line-height: 2.5rem;
      --text-title-lg: 1.5rem;    --text-title-lg--line-height: 2rem;
      --text-title-md: 1.125rem;  --text-title-md--line-height: 1.625rem;
      --text-body: 0.9375rem;     --text-body--line-height: 1.5rem;
      --text-price: 1.25rem;      --text-price--line-height: 1.75rem;
      --text-price-lg: 1.75rem;   --text-price-lg--line-height: 2.25rem;
    }
  }

  @media (prefers-reduced-motion: reduce) {
    *, *::before, *::after {
      animation-duration: 1ms !important;
      animation-iteration-count: 1 !important;
      transition-duration: 1ms !important;
      scroll-behavior: auto !important;
    }
  }
}

@utility tabular { font-variant-numeric: tabular-nums; }
```

Bobot huruf dipakai lewat utilitas Tailwind biasa (`font-normal` 400, `font-medium` 500, `font-semibold` 600, `font-bold` 700)
sesuai tabel §3.5. Setup font di `app/layout.tsx`:

```tsx
import { Poppins } from "next/font/google";

const poppins = Poppins({
  subsets: ["latin"],
  weight: ["400", "500", "600", "700"],
  display: "swap",
  variable: "--font-poppins",
});

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="id" className={poppins.variable}>
      <body>
        {children}
        <Toaster />
      </body>
    </html>
  );
}
```

---

## 4. Ikon

Sumber: `lucide-react`. Grid 24px, garis (stroke), sudut membulat — konsisten satu gaya.
**Jangan** mencampur ikon isi (filled) kecuali bintang rating terisi dan hati favorit aktif (lihat §6.35, §6.37).

| Ukuran | Pemakaian | `strokeWidth` |
|---|---|---|
| 16px | Di dalam badge, chip, caption | 2 |
| 20px | Di dalam tombol & input, item list | 1.75 |
| 24px | Bottom nav, header, tombol ikon | 1.75 |
| 40px | Empty state | 1.5 |

Aturan:
- Warna selalu `currentColor`; warna diatur dari teks induknya.
- Ikon dekoratif (di samping teks yang sudah menjelaskan): `aria-hidden="true"`.
- Ikon tanpa teks (tombol ikon): wajib `aria-label` di tombol, bukan di ikon.
- Impor per ikon (`import { Search } from "lucide-react"`), jangan impor seluruh paket.
- Nama di tabel diverifikasi di lucide.dev saat implementasi. Bila ada nama yang berubah, pakai pengganti
  terdekat di kategori yang sama dan perbarui tabel ini.

| Konteks | Ikon (komponen lucide-react) |
|---|---|
| Cari, lokasi, filter | `Search`, `MapPin`, `SlidersHorizontal`, `X` |
| Navigasi | `House`, `Heart`, `ReceiptText`, `Bell`, `UserRound`, `ChevronLeft`, `ChevronRight`, `ChevronDown` |
| Status & toast | `CircleCheck`, `TriangleAlert`, `CircleAlert`, `Info`, `Clock` |
| Properti | `Building2`, `BedDouble`, `Users`, `Star`, `Zap` (flash sale), `Eye`, `Camera` |
| Fasilitas | `Wifi`, `AirVent`, `Bath`, `BedDouble`, `Clock`, `Check` (fasilitas lain) |
| Transaksi | `Wallet`, `CreditCard`, `CalendarDays`, `FileText`, `Download`, `Banknote` |
| Akun | `Phone`, `Mail`, `LockKeyhole`, `LogOut`, `Pencil`, `ShieldCheck` |
| Owner/admin | `Plus`, `Trash2`, `Upload`, `ImagePlus`, `ChartColumn`, `ListChecks`, `Flag`, `EyeOff`, `GripVertical` |
| Menu admin | `Menu`, `Building2`, `Star`, `ListChecks`, `ReceiptText`, `Undo2`, `Banknote`, `Wallet`, `Scale`, `History`, `Users`, `MapPin`, `Settings`, `UserRound` |
| Lain | `Copy`, `ExternalLink`, `RefreshCw`, `LoaderCircle` (spinner), `ArrowUpDown` (urutkan), `MessageSquareText` (SMS) |

**Pemetaan fasilitas** (`facilities` berupa teks bebas dari owner). Cocokkan tanpa membedakan huruf besar/kecil:

| Teks mengandung | Ikon |
|---|---|
| `cctv` | `Camera` |
| `wifi` | `Wifi` |
| `ac` (kata utuh) | `AirVent` |
| `kipas` | `Fan` |
| `mandi` | `Bath` |
| `kasur`, `bed` | `BedDouble` |
| `lemari` | `Shirt` |
| `meja` | `Table2` |
| `tv`, `televisi` (kata utuh) | `Tv` |
| `water heater`, `air panas` | `Flame` |
| `kulkas` | `Refrigerator` |
| `dapur` | `CookingPot` |
| `laundry`, `cuci` | `WashingMachine` |
| `jemur` | `Sun` |
| `ruang tamu` | `Sofa` |
| `parkir motor` | `Bike` |
| `parkir mobil` | `Car` |
| `mushola`, `masjid` | `Moon` |
| `24 jam` | `Clock` |
| lainnya | `Check` |

Urutan pencocokan mengikuti urutan baris di atas (`CCTV 24 jam` menjadi `Camera`, bukan `Clock`).

Logo aplikasi adalah aset SVG milik kamu sendiri (`/public/logo.svg`), bukan dibuat dari ikon.

**Logo merek pihak ketiga** mengikuti §0.1-C. Yang dipakai di v1:

| Merek | File | Dipakai di | Sumber |
|---|---|---|---|
| WhatsApp | `/public/brand/whatsapp.svg` | Pilihan kanal OTP (§6.48) | Halaman brand resources resmi WhatsApp |

Logo merek ditampilkan sebagai `<img src alt="WhatsApp">` (atau `alt=""` bila nama merek sudah tertulis di sebelahnya),
ukuran 20px di dalam tombol, tidak diberi `currentColor`.

---

## 5. Pola gerak

### 5.1 Transisi halaman

Pakai `template.tsx` di App Router: berbeda dengan `layout.tsx`, template di-*mount* ulang setiap navigasi,
sehingga animasi masuk berjalan otomatis tanpa library.

File `app/(main)/template.tsx`:

```tsx
export default function Template({ children }: { children: React.ReactNode }) {
  return <div className="page-container animate-page-enter">{children}</div>;
}
```

- Hanya animasi **masuk** (fade + geser 8px, 200ms). Tidak ada animasi keluar; itu membuat navigasi terasa lambat.
- Header dan bottom nav berada di `layout.tsx` (di luar template), jadi tidak ikut beranimasi.
- Bottom sheet, dialog, dan toast tidak memakai animasi ini.

### 5.2 Accordion

Tinggi dianimasikan lewat trik `grid-template-rows` (0fr → 1fr), tanpa mengukur tinggi dengan JavaScript.

```tsx
<div className="border-b border-neutral-200">
  <h3>
    <button
      type="button"
      id={`${id}-trigger`}
      aria-expanded={open}
      aria-controls={`${id}-panel`}
      onClick={() => setOpen(!open)}
      className="flex min-h-12 w-full items-center justify-between gap-3 py-3 text-left text-title-sm font-semibold"
    >
      {title}
      <ChevronDown aria-hidden="true" size={20} strokeWidth={1.75}
        className={cn("shrink-0 text-neutral-600 transition-transform duration-200 ease-standard", open && "rotate-180")} />
    </button>
  </h3>
  <div
    id={`${id}-panel`}
    role="region"
    aria-labelledby={`${id}-trigger`}
    inert={!open}
    className={cn("grid transition-[grid-template-rows] duration-200 ease-standard",
      open ? "grid-rows-[1fr]" : "grid-rows-[0fr]")}
  >
    <div className="overflow-hidden">
      <div className="pb-4 text-body text-neutral-700">{children}</div>
    </div>
  </div>
</div>
```
`inert` membuat isi yang tertutup tidak bisa difokus atau dibaca pembaca layar.

### 5.3 Bottom sheet & dialog

Pakai elemen native `<dialog>` + `showModal()`: fokus terkunci di dalam, tombol Esc menutup, dan tombol Back
Android ditangani browser. Animasi masuk pakai `@starting-style` (browser lama tetap berfungsi, hanya tanpa animasi).

```css
dialog.sheet {
  transition: transform 280ms var(--ease-standard), opacity 280ms var(--ease-standard),
              overlay 280ms allow-discrete, display 280ms allow-discrete;
  transform: translateY(0);
}
dialog.sheet:not([open]) { transform: translateY(100%); }
@starting-style { dialog.sheet[open] { transform: translateY(100%); } }

dialog.sheet::backdrop { background: rgb(22 22 28 / 0.48); transition: opacity 280ms; opacity: 1; }
@starting-style { dialog.sheet[open]::backdrop { opacity: 0; } }
```
Dialog tengah memakai pola sama dengan `opacity` + `scale(0.98)` sebagai pengganti `translateY`.

Laci (`dialog.drawer`, dipakai AdminSidebar §6.23a) memakai pola yang sama dengan arah geser dari kiri:
`translateX(-100%)` → `translateX(0)`, backdrop sama.

### 5.4 Umpan balik tekan

- Tombol: `active:scale-[0.98]` dengan `duration-fast`. Tidak ada efek riak (ripple).
- Kartu listing: tidak ada animasi hover atau angkat; hanya judul berubah ke `brand-700` saat hover (desktop).
- Gambar galeri: geser pakai `scroll-snap`, bukan animasi JavaScript.

### 5.5 Gerak yang dikurangi

Aturan `prefers-reduced-motion` di §3.10 mematikan semua animasi dan transisi secara global. Jangan menimpanya.
Hitung mundur dan skeleton tetap berfungsi, hanya tanpa efek gerak (skeleton jadi blok abu statis).

---

## 6. Komponen

Format setiap komponen: **anatomi**, **varian/ukuran**, **state**, **aksesibilitas**, **data API**.
Semua komponen dibuat di `components/ui/` (dasar) atau `components/features/` (terkait domain).

### 6.1 Button

Semua tombol memakai border 1px dan radius `radius-md` yang sama. Warna mengikuti jenis aksi, bukan dekorasi.

| Varian | Latar | Teks | Border | Hover | Pressed |
|---|---|---|---|---|---|
| `success` (default) | `success` | `neutral-0` | `success` | gelapkan 8% (`brightness-92`) | `brightness-85` |
| `danger` | `danger` | `neutral-0` | `danger` | gelapkan 8% | `brightness-85` |
| `neutral` | `neutral-0` | `neutral-900` | `neutral-400` | latar `neutral-50` | latar `neutral-100` |

**Pemetaan aksi:**
- Hijau `success`: simpan, konfirmasi, masuk, daftar, lanjut, kirim, bayar, ajukan sewa, perpanjang, setujui, tandai selesai,
  aktifkan kembali, tampilkan (ulasan/kamar), tambah (kamar, area, foto), tampilkan hasil, cari, ajukan ulang.
- Merah `danger`: batalkan (booking), tolak (booking/properti), hapus (kamar, area, foto), keluar, nonaktifkan (pengguna/properti).
- Abu-abu `neutral`: ubah/edit, batal (menutup dialog/sheet), atur ulang, urutkan, muat lebih banyak, ulangi, coba lagi,
  kirim ulang kode, pilih fasilitas.

**Ikon tombol** (hanya halaman properti owner, `/owner/properti` dan `/owner/properti/{id}`):
- Ubah → `Pencil` (neutral), Hapus → `Trash2` (danger), Tambah → `Plus` (success). Di luar halaman itu tombol tanpa ikon,
  kecuali ikon yang sudah ditetapkan di §0.1-B (`Download`, `ImagePlus`, `RefreshCw`, `SlidersHorizontal`, `ArrowUpDown`).

| Ukuran | Tinggi | Padding X | Teks | Ikon |
|---|---|---|---|---|
| `sm` | 36px (area sentuh tetap 44px lewat padding luar) | 12px | `label` 600 | 16px |
| `md` (default) | 44px | 16px | `body-strong` | 20px |
| `lg` | 52px | 20px | `body-strong` | 20px |

- Radius `radius-md`. Lebar penuh (`w-full`) di form mobile dan sticky bar.
- **Loading:** label tetap terlihat (lebar tidak berubah), spinner 16px menggantikan ikon kiri,
  `aria-busy="true"`, tombol `disabled`.
- **Disabled:** latar `neutral-100`, teks `neutral-400`, kursor `not-allowed`. Jangan pakai opacity.
- Maksimal **satu** tombol `success` utama per layar/area aksi; aksi lain tetap hijau bila memang positif, tanpa dua tombol hijau bersebelahan untuk aksi yang sama.
- Label tombol = kata kerja + objek ("Ajukan sewa", "Bayar sekarang"). Tidak ada tanda panah di teks.
- Pasangan tombol sebaris wajib 50:50 dan ikon hanya sesuai daftar §0.1-B.
- Komponen `ButtonPair` (wrapper `grid grid-cols-2 gap-3`) dipakai untuk setiap pasangan tombol; jangan menyusun manual.

### 6.1a FacilitySelect (kamar, owner)

Pilihan fasilitas kamar multi-pilih dari daftar baku (`facilityCatalog`, mengikuti fasilitas yang sudah dipakai seed/demo:
WiFi, AC, Kipas Angin, Kasur, Kasur Queen, Lemari, Meja Belajar, Meja Kerja, Kamar Mandi Dalam, Kamar Mandi Luar,
Water Heater, TV, Akses 24 Jam, Dapur bersama, Ruang tamu, Layanan laundry, Area jemur di rooftop, Parkir motor,
Parkir mobil terbatas, CCTV 24 jam, Mushola). Tidak ada input teks bebas.
- Tampilan tertutup: label "Fasilitas" + daftar **chip** yang dipilih (`Chip` dengan tombol `X` ber-`aria-label`
  "Hapus {nama}", ikon `X` 16px) + tombol `neutral` "Pilih fasilitas".
- Tombol "Pilih fasilitas" membuka BottomSheet (§6.20) berisi daftar `Checkbox` (§6.7) per fasilitas; kotak yang sudah
  tercentang tetap menampilkan fasilitas lama yang tidak ada di daftar baku (agar data lama tidak hilang).
- Footer sheet: `success` "Selesai". Maksimal 30 fasilitas, masing-masing maksimal 50 karakter (sesuai kontrak).
- Teks bantuan "Pisahkan dengan koma" dihapus. Nilai dikirim sebagai array `facilities`.

### 6.2 IconButton

Tombol bulat tanpa teks. Ukuran sentuh selalu 44×44. Ukuran visual: `md` 44px (ikon 24px), `sm` 36px (ikon 20px),
`xs` 32px (ikon 18px, untuk tombol di atas foto yang sempit; area sentuh tetap 44px lewat padding transparan).
Varian: `plain` (transparan, ikon `neutral-700`), `overlay` (latar putih 92% + `shadow-raised`, dipakai di atas foto),
`onBrand` (transparan, ikon putih, hover `brand-700`; dipakai di AppHeader).
Wajib `aria-label`. Badge hitungan (§6.12) boleh ditempel di pojok kanan atas.

### 6.3 TextField

Anatomi: label (di atas) → input → helper text **atau** pesan error (di bawah).

| Bagian | Spesifikasi |
|---|---|
| Label | `label` 500, `neutral-700`, wajib ada (bukan placeholder sebagai label) |
| Input | tinggi 48px, padding X 14px, radius `radius-md`, border `neutral-400`, teks `body` `neutral-900`, ukuran font minimal 16px di mobile agar iOS tidak zoom |
| Placeholder | `neutral-500`, berisi contoh format, bukan instruksi |
| Helper | `caption` `neutral-500` |
| Error | `caption` `danger`, diawali ikon `CircleAlert` 16px |

State: fokus → border `brand-600` + ring 3px `brand-100`; error → border `danger`, `aria-invalid="true"`,
`aria-describedby` menunjuk pesan error; disabled → latar `neutral-50`, border `neutral-300`.

Varian:
- **Password:** tombol ikon `Eye`/`EyeOff` di kanan (`aria-label` "Tampilkan password"/"Sembunyikan password").
- **Nomor HP:** `type="tel"`, `inputMode="numeric"`, `autoComplete="tel-national"`, placeholder `081234567890`.
  Validasi klien sama dengan API: `^08\d{7,12}$`.
- **Rupiah (owner):** `inputMode="numeric"`, tampilkan `Rp` sebagai awalan visual, simpan angka murni.

### 6.4 OtpInput

4 kotak 52×56px, gap 12px, teks `title-lg` rata tengah, `tabular`.
- Satu `<input>` tersembunyi yang menerima semua ketikan (`inputMode="numeric"`, `autoComplete="one-time-code"`,
  `maxLength={4}`) di belakang 4 kotak visual. Ini membuat isi otomatis dari SMS dan tempel (paste) bekerja.
- Kotak aktif: border `brand-600`. Error: semua kotak border `danger` + pesan di bawah.
- Otomatis kirim saat 4 digit terisi.
- Kanal pengiriman dipilih sebelumnya lewat OtpChannelPicker (§6.48).
- Di bawahnya: teks hitung mundur kirim ulang dari `verification_code.can_resend_at`
  ("Kirim ulang dalam 0:42"), lalu tombol `neutral` "Kirim ulang kode" saat habis.

### 6.5 Textarea

Sama dengan TextField; tinggi minimal 96px, `resize-y`. Bila ada batas (mis. ulasan 1000, alasan tolak 500),
tampilkan penghitung `caption` di kanan bawah ("120/1000"), berubah ke `danger` saat lewat.

### 6.6 Select

Pakai `<select>` native (ringan, aksesibel, picker bawaan mobile). Tampilan = TextField + ikon `ChevronDown` 20px di kanan
(`pointer-events-none`). Untuk pilihan ≤ 4 yang sering diganti, pakai SegmentedControl (§6.8).

### 6.7 Checkbox

Kotak 20×20, radius `radius-sm`, border `neutral-400`; tercentang → latar `brand-600` + ikon `Check` putih 16px.
Label di kanan, seluruh baris bisa diklik, tinggi baris minimal 44px.
Dipakai untuk `is_agree_privacy_policy` (teks: "Saya menyetujui Kebijakan Privasi", tautan ke halaman kebijakan).

### 6.8 SegmentedControl

Kelompok pilihan eksklusif dalam satu baris. Wadah latar `neutral-100`, radius `radius-md`, padding 4px;
segmen aktif latar putih + `shadow-raised`, teks `neutral-900` 600; tidak aktif teks `neutral-600`.
Tinggi 44px. `role="radiogroup"` dengan tombol `role="radio"` + `aria-checked`, panah kiri/kanan berpindah pilihan.

Dipakai untuk: tipe sewa (Bulanan/Mingguan/Harian, hanya yang harganya ada), periode laporan owner.

### 6.9 Switch

Track 44×24, thumb 20px. Aktif: track `brand-600`; nonaktif: track `neutral-300`.
`role="switch"` + `aria-checked`. Dipakai untuk `is_active` properti dan `is_flash_sale` (owner).

### 6.10 DateField

`<input type="date">` native dengan gaya TextField. `min` = tanggal hari ini **WIB** (format `YYYY-MM-DD`).
Di bawahnya tampilkan tanggal terpilih dalam format panjang ("Senin, 1 Desember 2026") sebagai `caption`.

### 6.11 Chip (filter)

Tinggi 36px (area sentuh 44px), padding X 14px, radius `radius-full`, teks `label` 500.
- Tidak aktif: border `neutral-300`, teks `neutral-700`, latar putih.
- Aktif: border `brand-600`, latar `brand-50`, teks `brand-800`, ikon `Check` 16px di kiri.
- Chip yang membuka sheet (mis. "Harga") memakai ikon `ChevronDown` di kanan.
- Baris chip bisa digeser horizontal (`overflow-x-auto`, `scroll-snap-type: x proximity`, scrollbar disembunyikan),
  dengan padding kiri-kanan sama dengan padding halaman.
- `<button aria-pressed>` untuk chip toggle.

### 6.12 Badge

Tinggi 22px, padding X 8px, radius `radius-sm`, teks `caption-strong`. Selalu berisi teks.

| Jenis | Isi | Warna |
|---|---|---|
| Gender | dari `gender` (§3.4) | warna gender |
| Flash sale | ikon `Zap` 14px + "Promo" | latar `brand-600`, teks putih |
| Sisa kamar | "Sisa {available} kamar" bila `available` 1–3 | `warning` |
| Penuh | "Penuh" bila `available = 0` | latar `neutral-100`, teks `neutral-600` |
| Status | lihat §8.1 | warna semantik |
| Hitungan | angka (maks "99+") | latar `brand-600`, teks putih, `radius-full`, minimal 18×18 |

Badge hitungan notifikasi diambil dari `GET /notifications/counter` → `unread`; disembunyikan bila 0.

### 6.13 Avatar

Lingkaran 40px (list) atau 64px (profil). Tidak ada foto profil di API → selalu **inisial**: huruf pertama dua kata
pertama `name`, latar `brand-100`, teks `brand-800` 600.

### 6.14 Skeleton

Blok `neutral-100`, radius sama dengan elemen yang digantikan, ukuran **sama persis** dengan konten aslinya.
Efek: gradien `neutral-100 → neutral-50 → neutral-100` dengan `animate-shimmer`. `aria-hidden="true"`;
wadahnya `aria-busy="true"`. Skeleton disiapkan untuk: ListingCard, halaman detail, BookingCard, NotificationItem.

### 6.15 Spinner

Ikon `LoaderCircle` dengan `animate-spin`. Hanya di dalam tombol dan aksi kecil; muatan halaman memakai Skeleton.

### 6.16 Toast (global, reusable)

**Satu** `<Toaster />` di root layout. Dipanggil dari mana saja lewat modul `lib/toast.ts`.

```ts
type ToastVariant = "success" | "error" | "warning" | "info";
type ToastOptions = {
  id?: string;              // id sama = ganti toast lama (mencegah dobel)
  description?: string;
  duration?: number;        // ms; Infinity = sampai ditutup
  action?: { label: string; onClick: () => void };
};

toast.success("Booking disetujui.");
toast.error("Kamar tidak tersedia.", { action: { label: "Lihat kamar lain", onClick: goBack } });
toast.warning(meta.message, { id: "rate-limit" });
toast.dismiss(id);
```

Implementasi: store kecil (array toast + subscriber) dibaca `<Toaster />` lewat `useSyncExternalStore`.
Tanpa library tambahan.

| Aspek | Spesifikasi |
|---|---|
| Posisi | Mobile: bawah tengah, `bottom = 64px (bottom nav) + safe area + 12px`; halaman tanpa bottom nav: `16px + safe area`. ≥ 1024px: kanan atas, 24px dari tepi |
| Ukuran | Lebar `min(100% - 32px, 400px)`, padding 12px 14px, radius `radius-lg`, `shadow-overlay` |
| Warna | Latar `neutral-800`, judul putih `body-strong`, deskripsi `neutral-300` `caption`, ikon warna `*-on-dark` (§3.3) |
| Ikon | success `CircleCheck`, error `CircleAlert`, warning `TriangleAlert`, info `Info` — 20px |
| Aksi | Tombol teks `brand-300` 600 di kanan; tombol tutup `X` 20px (`aria-label` "Tutup notifikasi") |
| Durasi | success/info 4 detik, warning 5 detik, error 6 detik, ada aksi = sampai ditutup. Timer berhenti saat hover/fokus |
| Tumpukan | Maksimal 3 terlihat; yang baru paling dekat ke tepi layar; yang lama tergeser |
| Animasi | Masuk `animate-toast-in`, keluar `animate-toast-out` (hapus dari DOM setelah `animationend`) |
| Aksesibilitas | Wadah `<section aria-label="Notifikasi">`; tiap toast `role="status"` (`aria-live="polite"`), error `role="alert"` |

Kapan memakai toast: lihat §8.2. Toast **tidak** dipakai untuk error validasi field.

### 6.17 InlineAlert

Pesan di dalam alur halaman (bukan melayang). Padding 12px, radius `radius-md`, latar `*-bg`, teks `neutral-900`,
ikon 20px warna semantik di kiri, judul opsional `body-strong`. `role="status"` (atau `role="alert"` untuk danger).
Contoh: "Isi rekening untuk menerima pengembalian dana." (refund `awaiting_account`), properti `rejected` + `moderation_note`.

### 6.18 EmptyState

Rata tengah, padding vertikal 32px (mobile) / 48px (≥ 768px). Ikon 40px `neutral-400` dalam lingkaran 72px `neutral-50`
→ judul `title-md` → deskripsi `body` `neutral-600` (maks 2 baris) → satu tombol `neutral` opsional.

| Layar | Judul | Aksi |
|---|---|---|
| Hasil cari kosong | Belum ada kamar yang cocok | Ubah filter |
| Favorit kosong | Belum ada kost favorit | Cari kost |
| Booking kosong | Belum ada booking | Cari kost |
| Notifikasi kosong | Belum ada notifikasi | — |
| Ulasan kosong | Belum ada ulasan | — |

### 6.19 ErrorState

Sama dengan EmptyState, ikon `CircleAlert` warna `danger`. Untuk 404: "Halaman tidak ditemukan" + "Kembali ke beranda".
Untuk gagal muat (5xx/jaringan): pesan API atau "Tidak bisa memuat data." + tombol "Coba lagi" (`RefreshCw`).

### 6.20 BottomSheet

`<dialog class="sheet">` (§5.3). Mobile (< 640px): menempel di bawah, lebar penuh, radius atas `radius-sheet`,
`shadow-sheet`, tinggi maks `90dvh`. ≥ 640px: dialog di tengah layar (lebar maks 560px), semua sudut `radius-sheet`,
`shadow-overlay`, tinggi maks `85dvh`, tanpa pegangan, muncul dengan fade + skala 0,98. Isi selalu bisa di-scroll
(`min-height: 0` pada area isi) sehingga ujung bawah popup panjang tetap terlihat.
Anatomi: pegangan 36×4px `neutral-300` (dekoratif) → header (judul `title-md` + `X`) → isi yang bisa di-scroll →
footer menempel (tombol aksi) dengan `padding-bottom: env(safe-area-inset-bottom)`.
Menutup: tombol `X`, Esc, klik backdrop. Fokus kembali ke pemicu setelah ditutup.
Dipakai: filter, pilih durasi, rincian harga di mobile.

### 6.21 Dialog & ConfirmDialog

`<dialog>` tengah, lebar `min(100% - 32px, 440px)`, radius `radius-lg`, padding 24px, `shadow-overlay`.
**ConfirmDialog** untuk aksi tak bisa dibatalkan: judul pertanyaan, isi konsekuensi, tombol `neutral` "Batal" (atau label lain lewat `cancelLabel`, mis. "Tidak" untuk batalkan booking) +
tombol aksi (`danger` bila merusak) dengan label kata kerja spesifik. Kedua tombol memakai `ButtonPair` (50:50),
"Batal" di kiri, aksi di kanan, keduanya tanpa ikon.

| Aksi | Judul | Tombol aksi |
|---|---|---|
| Tenant batalkan booking `paid` | Batalkan booking? | Batalkan booking (danger) — isi menyebut dana kembali dikurangi biaya admin, dari `admin_fee_label` |
| Owner tolak booking | Tolak booking ini? | Tolak booking (danger) — dengan Textarea alasan opsional |
| Owner hapus foto/kamar | Hapus foto? / Hapus kamar? | Hapus (danger) |
| Keluar | Keluar dari akun? | Keluar |

### 6.22 AppHeader

Tinggi 56px, latar `brand-600` dengan teks dan ikon putih (kontras 4,8:1), `z-sticky`, menempel di atas. Ring fokus
di dalam header berwarna putih; IconButton memakai varian `onBrand`; badge hitungan dibalik (latar putih, angka
`brand-700`); favorit aktif di detail berupa hati putih terisi. `shadow-raised` muncul hanya setelah halaman di-scroll
(pantau dengan `IntersectionObserver` pada elemen penanda, bukan event scroll).
- **Beranda:** logo (kiri) + IconButton `Bell` dengan badge hitungan (kanan).
- **Halaman dalam:** IconButton `ChevronLeft` ("Kembali") + judul `title-md` (1 baris, terpotong, boleh dihilangkan)
  + aksi kanan opsional. Di detail kost: `Heart` lalu `Share2` (IconButton `onBrand`, ukuran dan ikon 20px sama).
  Tanpa judul, ruang tengah diisi spacer di mobile.
- ≥ 1024px: logo + tautan navigasi utama + tombol akun; bottom nav disembunyikan. Isi header di kontainer 1200px
  (§3.8). Halaman dalam: logo + garis pemisah `neutral-200` + judul `body` 500 `neutral-600` (tombol kembali
  disembunyikan). Tautan = item §6.23 tanpa Akun, teks `body-strong` putih; hover dan aktif berlatar `brand-700`
(aktif + `aria-current="page"`); tamu
  yang membuka Favorit/Booking diarahkan ke halaman masuk. Tombol akun: tamu → Button `success` `sm` "Masuk";
  sudah masuk → IconButton `UserRound` ("Akun"). `Bell` hanya di beranda dan halaman legal.
- **Halaman legal** (Syarat & Ketentuan, Kebijakan Privasi): varian beranda, isi di kontainer 1200px seperti halaman lain.

### 6.22a SiteFooter

Di akhir semua halaman grup utama dan halaman legal, latar `neutral-50`, border atas `neutral-200`, isi di kontainer
1200px. Baris atas: logo teks (tautan ke `/`) + deskripsi `caption` `neutral-600`; tautan `body` `neutral-700`
(Cari kost, Syarat & Ketentuan, Kebijakan Privasi), target sentuh 44px. Mobile: tautan sejajar inline dalam
satu baris yang membungkus rapi (jarak 16px, teks `label`); ≥ 1024px sebaris
(tautan di kanan). Baris bawah dipisah border `neutral-200`: "Copyright © 2026 - All Rights Reserved", `caption`
300, rata tengah, `neutral-600`. Footer menempel di dasar layar pada halaman pendek dan tidak tertutup bottom nav
maupun StickyActionBar (ruang bawah mobile = tinggi bar + safe area).

### 6.23 BottomNav (mobile)

Tinggi 64px + safe area, latar putih, border atas `neutral-200`, `z-sticky`. Disembunyikan di ≥ 1024px dan
di halaman dengan StickyActionBar (detail, checkout).

| Role | Item (ikon + label) |
|---|---|
| Tamu / tenant | Beranda (`House`), Cari (`Search`), Favorit (`Heart`), Booking (`ReceiptText`), Akun (`UserRound`) |
| Owner | Beranda (`ChartColumn`), Properti (`Building2`), Booking (`ListChecks`), Keuangan (`Wallet`), Akun (`UserRound`) |

Item: ikon 24px + teks `label` 500. Aktif: ikon & label `brand-700`, garis 3×24px `brand-600` di atas ikon.
Tidak aktif: `neutral-500`. `<nav aria-label="Navigasi utama">`, item aktif `aria-current="page"`.
Tamu yang membuka Favorit/Booking diarahkan ke halaman masuk.

### 6.23a AdminSidebar (panel admin)

Navigasi panel admin, `<nav aria-label="Menu admin">`. Item: ikon 20px + teks `label` 500, tinggi 44px, radius
`radius-md`. Aktif: latar `brand-50`, teks & ikon `brand-800`, 600, `aria-current="page"`. Tidak aktif: `neutral-700`,
hover `neutral-50`. Akun dipisah border atas `neutral-200` di akhir daftar.

| Item | Ikon | Item | Ikon |
|---|---|---|---|
| Properti | `Building2` | Keuangan | `Wallet` |
| Ulasan | `Star` | Rekonsiliasi | `Scale` |
| Booking | `ListChecks` | Riwayat | `History` |
| Tagihan | `ReceiptText` | Pengguna | `Users` |
| Refund | `Undo2` | Area | `MapPin` |
| Payout | `Banknote` | Pengaturan | `Settings` |
| Akun | `UserRound` | | |

- ≥ 1024px: sidebar tetap 240px di kiri, tinggi layar penuh, border kanan `neutral-200`, logo teks di atas, daftar
  bisa di-scroll sendiri.
- < 1024px: bilah atas 56px menempel (`z-sticky`, latar putih, border bawah `neutral-200`) berisi IconButton `Menu`
  ("Buka menu", `aria-expanded`, `aria-controls`) + logo teks. Tombol membuka laci `<dialog class="drawer">` (§5.3)
  dari kiri: lebar `min(280px, 100% - 56px)`, tinggi layar penuh, `shadow-overlay`, header logo + `X` ("Tutup menu").
  Menutup: `X`, Esc, klik backdrop, memilih item, atau layar melebar ke ≥ 1024px. Fokus kembali ke tombol menu.

### 6.24 Tabs

Daftar tab horizontal yang bisa digeser, tinggi 44px, teks 14px/500. Aktif: teks `neutral-900` 600 + garis bawah
2px `brand-600`. Pola ARIA tabs (`role="tablist"`, `tab`, `tabpanel`, panah kiri/kanan).
Dipakai untuk filter status (booking tenant, booking owner, tagihan, refund/payout admin).

### 6.25 LoadMore

Daftar memakai tombol `neutral` lebar penuh "Muat lebih banyak" di bawah daftar, bukan infinite scroll otomatis
(hemat kuota, footer tetap bisa dicapai).
- Search: tampil bila `has-more = true`; request berikutnya memakai `offset = next-offset`.
- Daftar berhalaman (`page`): tampil bila jumlah item yang diterima = `limit`.
- Saat memuat: tombol loading; item baru ditambahkan di bawah tanpa menggeser posisi scroll.

### 6.26 SearchBar & SuggestionList

SearchBar: tinggi 48px, radius `radius-full`, latar `neutral-50`, border `neutral-200`, ikon `Search` 20px di kiri,
placeholder "Cari kota, kampus, atau nama kost". Di beranda bertindak sebagai tombol yang membuka halaman cari
dengan input fokus.

SuggestionList (`GET /suggestion?keyword=`):
- Request dikirim setelah ≥ 3 karakter dan jeda ketik 300ms; request lama dibatalkan (`AbortController`).
- Tiga kelompok berjudul `caption-strong` `neutral-600`: **Area** (`areas[]`: `MapPin`, `title`, `area`),
  **Kost** (`landings[]`: `Building2`, `title`, badge gender), **Tipe kamar** (`room_types[]`: `BedDouble`, `title`).
  Kelompok kosong tidak ditampilkan.
- Item tinggi 52px. Pola ARIA combobox: input `role="combobox"`, `aria-expanded`, `aria-controls`,
  `aria-activedescendant`; daftar `role="listbox"`; panah atas/bawah + Enter.
- Baris pertama (kelompok **Pencarian**, ikon `Search`): "Cari “{teks}”" muncul bila ≥ 3 karakter. Memilihnya atau
  menekan Enter tanpa item aktif → `/cari?q={teks}`, yang mencari listing dengan `filters.keyword` (API.md §3.3) walaupun
  tidak ada yang cocok di saran.
- Pilih area → `/cari?area_id={geocode_id}`. Pilih kost → `/kost/{id}`. Pilih tipe kamar → `/kost/{property_id}?room={id}`.
- Kurang dari 3 karakter: tampilkan "Area populer" dari `GET /area/city?top=true` sebagai chip.

### 6.27 FilterSheet

BottomSheet berisi filter yang dipetakan 1:1 ke `filters` di `POST /listings/search`:

| Filter | Kontrol | Nilai API |
|---|---|---|
| Tipe sewa | SegmentedControl Bulanan/Mingguan/Harian | `rent_type` 3/2/1 |
| Gender | Chip multi-pilih Putra/Putri/Campur | `gender` [1, 2, 0] |
| Rentang harga | Dua TextField Rupiah (min & maks) | `price_range` |
| Promo | Switch "Hanya yang sedang promo" | `flash_sale` |
| Tersedia | Switch "Hanya kamar yang tersedia" | `is_available` |
| Urutkan | Radio: Harga terendah / Harga tertinggi / Kamar terbanyak | `sorting` price `+` / price `-` / availability `-` |

Urutkan dipisah ke sheet sendiri (**SortSheet**): daftar radio satu kolom, langsung diterapkan saat dipilih, sheet menutup.
Baris "Urutkan" di tabel atas hanya ada di SortSheet, bukan di FilterSheet.

Footer FilterSheet: `ButtonPair` — `neutral` "Atur ulang" (kiri) + `success` "Tampilkan" (kanan), tanpa ikon.
Jumlah hasil tidak ditampilkan di tombol (API tidak punya endpoint hitung tanpa memuat hasil).

**FilterBar** di halaman hasil (pemicu sheet):
- `ButtonPair` berisi `neutral` `sm` "Filter" (ikon `SlidersHorizontal`) dan "Urutkan" (ikon `ArrowUpDown`) —
  keduanya berikon, sama lebar. Filter aktif ditandai badge hitungan di tombol "Filter".
- Di bawahnya, **hanya bila ada filter aktif**: satu baris chip aktif (bisa digeser) dengan ikon `X`; menekan chip
  menghapus filter itu.
- Menempel di bawah header dan ikut tersembunyi saat menggulir ke bawah (§0.1-E).

`property_type` tidak ditampilkan (semua properti adalah kost).

### 6.28 ListingCard

Sumber: item `rooms[]` dari `POST /listings/search`.

**Mobile < 640px — horizontal** (hemat ruang, ±4 kartu per layar):
```
┌────────┐ [Campur] Kota Depok         ← badge gender + area (caption)
│ [Promo]│ Kost Asti Beji              ← title-sm, 1 baris
│   foto │ Kamar Tipe A                ← caption, 1 baris
│  4:5   │ ★ 4,5 (12)  Sisa 2 kamar
└────────┘ Rp2.250.000 /bulan          ← price, menempel di dasar (mt-auto)
```
Foto lebar 112px rasio 4:5, radius `radius-md`; kolom info `min-w-0 flex flex-col`; tinggi kartu = tinggi foto.

**≥ 640px — vertikal** (grid sama tinggi, §0.1-D):
```
┌──────────────────────────────┐
│ [Promo]                      │  ← foto 4:3, radius-md
└──────────────────────────────┘
 [Campur]  Kota Depok
 Kost Asti Beji                   ← title-sm, 1 baris
 Kamar Tipe A                     ← caption, 1 baris
 ★ 4,5 (12)        Sisa 2 kamar
 Rp2.250.000 /bulan               ← mt-auto: sejajar di semua kartu sebaris
```

| Elemen | Field | Aturan |
|---|---|---|
| Foto | `thumbnail_url` | `next/image`, `sizes="(min-width:1024px) 25vw, (min-width:640px) 50vw, 100vw"`; `null` → blok `neutral-100` + ikon `Building2` |
| Badge promo | `is_flash_sale` | Kiri atas foto, 8px dari tepi |
| Gender | `gender` | Badge §3.4 |
| Area | `area_name` | `caption` `neutral-600` |
| Nama kost | `property_name` | `title-sm` |
| Nama kamar | `name` | `caption` `neutral-600` |
| Rating | `rating_average`, `rating_count` | Disembunyikan bila `rating_count = 0` |
| Ketersediaan | `available` | Badge sisa (1–3) atau Penuh (0); ≥ 4 tidak ditampilkan |
| Harga | `price_label` | Satuan dari `rent_type` filter: 3 `/bulan`, 2 `/minggu`, 1 `/hari` |

- Seluruh kartu adalah satu tautan ke `/kost/{property_id}`.
- Tanpa border dan bayangan. Daftar: 1 kolom horizontal (< 640, jarak 16px), grid 2 (≥ 640), 3 (≥ 1024), 4 (≥ 1280).
- Kartu `h-full flex flex-col`; baris yang bisa kosong (rating, sisa kamar) tetap memesan tinggi satu baris agar
  harga sejajar antar kartu.
- Skeleton mengikuti bentuk yang sama (horizontal di mobile).
- Harga tidak pernah terpotong; nama kost yang terpotong.
- Varian `vertical`: bentuk ≥ 640px (foto 4:3 di atas) di semua lebar, untuk slot sempit: carousel promo dan grid
  "Kost terbaru" di beranda.

### 6.28b ListingGrid & Kost lainnya

`ListingGrid` membungkus daftar `ListingCard` (mode vertikal) dalam grid yang sama dengan beranda ("Kost terbaru")
dan section "Kost lainnya" di detail kost. Dua varian kolom:
- `home`: 2 kolom (mobile), 3 (≥ 640px), 4 (≥ 1024px).
- `detail`: 2 kolom (mobile), 3 (≥ 1024px).

Section **Kost lainnya** di bawah ulasan detail kost (`/kost/{id}`): judul `title-sm`, 6 kartu acak dari daftar kost
terbaru (`GET /listings/latest?limit=50`) dengan properti yang sedang dibuka dikecualikan dan satu kartu per properti.
Pengacakan dilakukan di server setiap permintaan halaman, sehingga setiap muat ulang menghasilkan kombinasi baru.
Bila kurang dari 1 kartu tersedia, section disembunyikan.

### 6.28a AreaBannerGrid (kota populer)

Grid card foto kota: 2 kolom (< 640), 3 kolom (≥ 640), jarak 12px (16px ≥ 1024). Card = satu tautan ke
`/cari?area_id=&area_name=`, rasio 4:3 (16:9 ≥ 1024), radius `radius-md`. Foto `object-cover` + lapisan `neutral-900`
60% (diukur: teks putih ≥ 4,5:1 di piksel paling terang area label pada semua foto), nama area di tengah `title-md`
(`title-lg` ≥ 640) 700 putih. Hover: foto membesar 5% (`motion-safe`). Foto dipetakan per `slug` area di frontend
(Unsplash License); area tanpa foto tampil sebagai card `brand-600` dengan nama putih.

### 6.29 PropertyGallery

Sumber: `property.photos[]` + `rooms[].photos[]` dari `GET /listings/{id}` (atau `GET /listings/{id}/gallery`).
- Mobile: lebar penuh tanpa padding, rasio 4:3, geser horizontal dengan `scroll-snap-type: x mandatory`.
  Penanda posisi "1/8" (`caption`, latar `neutral-900` 64%, teks putih) di kanan bawah.
- ≥ 1024px: foto utama 16:9 di kiri (2/3 lebar) + dua foto bertumpuk di kanan; tombol "Lihat semua foto" membuka
  dialog galeri layar penuh.
- Foto pertama: `priority` (LCP). Lainnya `loading="lazy"`. Pakai `url` (besar) di galeri, `thumbnail_url` di strip kecil.
- `alt` = `caption` bila ada, selain itu "Foto {nama properti} {n}".
- Tanpa foto: blok 4:3 `neutral-100` dengan ikon `Camera` dan teks "Belum ada foto".

### 6.30 FacilityList

Grid 2 kolom (mobile) / 3 kolom (≥ 768px). Item: ikon 20px (`neutral-600`, pemetaan §4) + teks `body`.
Tampilkan 6 pertama; sisanya di dalam Accordion "Lihat semua fasilitas ({n})". Sumber: `rooms[].facilities`.

### 6.31 RoomOption

Kartu pilihan kamar di halaman detail, satu per item `rooms[]`. Padding 16px, radius `radius-lg`, border `neutral-200`.
Terpilih: border 2px `brand-600`, latar `brand-50`. Seluruh kartu `role="radio"` dalam `role="radiogroup"`.
Isi: nama kamar `title-sm` → FacilityList ringkas (maks 3 ikon + "+n") → harga `price_label` /bulan → badge ketersediaan.
Kamar `available = 0`: tetap tampil, `aria-disabled="true"`, tidak bisa dipilih, badge "Penuh".

### 6.32 PriceBreakdown

Sumber: `GET /booking/draft-price/{room}?is_flash_sale=&duration=` → `data[]` (pilih item sesuai tipe sewa).
Panel border `neutral-200`, radius `radius-lg`, padding 16px. Baris: label `body` `neutral-700` kiri, nilai `body` kanan
(`tabular`).

| Baris | Field | Tampil bila |
|---|---|---|
| Sewa | `price_label` (× durasi untuk harian/mingguan) | selalu |
| Deposit | `deposit_label` + caption "Dikembalikan pemilik di akhir sewa" | selalu |
| Biaya admin | `admin_fee_label` | `admin_fee > 0` |
| Diskon promo | `− {flash_sale_discount_label}` warna `brand-700` | `flash_sale_discount > 0` |
| **Total bayar sekarang** | `total_off_first_payment_label` (`price-lg`) | selalu, dipisah divider |

Harga sebelum diskon (`remaining_payment_label`) ditampilkan dicoret (`line-through`, `neutral-500`) di atas total
hanya bila ada diskon. Sewa bulanan dengan durasi > 1: caption "Bulan berikutnya ditagih {price_label} per bulan".

### 6.33 StickyActionBar

Menempel di bawah (mobile) di halaman detail & form booking. Latar putih, border atas `neutral-200`, `shadow-raised`,
padding 12px 16px + safe area. Kiri: caption "Mulai dari" + harga `price` (+ satuan); kanan: tombol `success` `lg`.
Konten halaman diberi `padding-bottom` setinggi bar agar tidak tertutup. ≥ 1024px: berubah menjadi panel menempel
(`position: sticky; top: 80px`) di kolom kanan.

### 6.34 Accordion

Pola §5.2. Dipakai untuk fasilitas lengkap, kebijakan, FAQ, dan rincian invoice di daftar. Boleh beberapa terbuka bersamaan.
Jangan menaruh aksi utama di dalam accordion yang tertutup. Deskripsi kost memakai §6.34a.

### 6.34a ExpandableText

Deskripsi kost (`description`) tampil langsung sebagai paragraf, dibatasi 3 baris (`line-clamp-3`), `body`
`neutral-700`. Tombol "Selengkapnya" (`ChevronDown`, membalik ke "Tutup" dengan `ChevronDown` terbalik) hanya muncul bila
teks benar-benar terpotong (tinggi konten melebihi batas), diukur setelah render dan saat ukuran layar berubah. Toggle
terjadi di tempat, tanpa memuat ulang halaman. Tombol memakai `aria-expanded`.

### 6.36a ReviewsPopup (detail kost)

Tombol "Lihat semua ulasan ({n})" (teks `label` 600 `brand-700`, ikon `ChevronRight`) adalah tautan ke
`/kost/{id}/ulasan`. Saat diklik dari detail kost, navigasi klien membuka popup `BottomSheet` (§6.20) di atas detail,
dengan RatingSummary dan daftar ulasan (ReviewList, LoadMore), sehingga URL berubah ke `/kost/{id}/ulasan`. Membuka URL
itu langsung atau memuat ulang halaman menampilkan halaman ulasan penuh. Menutup popup kembali ke detail. Popup berbentuk
lembar bawah di mobile dan di tengah (maks 560px) di layar lebih lebar, mengikuti BottomSheet.

### 6.35 RatingStars & RatingSummary

- **Tampilan:** ikon `Star` 16px, terisi (`fill="currentColor"`) warna `star`; angka `body-strong` "4,5" (koma desimal) +
  "(12 ulasan)" `caption` `neutral-600`. Pembaca layar: `aria-label="Rating 4,5 dari 5, 12 ulasan"`.
- **Input (ulasan):** 5 tombol bintang 40×40, `role="radiogroup"`, label per bintang ("1 bintang" … "5 bintang"),
  teks penjelas di bawah: 1 Buruk, 2 Kurang, 3 Cukup, 4 Bagus, 5 Sangat bagus.
- **RatingSummary** (detail): angka besar `title-lg` + bintang + jumlah; tanpa histogram (API tidak menyediakan).
- `rating.average = null` → tampilkan "Belum ada ulasan".

### 6.36 ReviewCard

Sumber: `GET /listings/{id}/reviews` → `reviews[]`. Avatar inisial dari `tenant_name` → nama `body-strong` +
`room_name` `caption` → bintang → `comment` (`body`, dipotong 4 baris dengan tombol "Selengkapnya") →
tanggal relatif `caption` (§8.3). Antar kartu dipisah divider, bukan kotak.

### 6.37 FavoriteButton

IconButton `overlay` dengan ikon `Heart`. Aktif: ikon terisi `brand-600`. Klik → ubah tampilan **langsung**
(optimistic) → `PUT`/`DELETE /favorites/{property}`; gagal → kembalikan tampilan + toast error.
`aria-pressed`, `aria-label` "Simpan ke favorit"/"Hapus dari favorit". Tamu → arahkan ke halaman masuk.
Status awal: di detail dari `property.is_favorite`; di kartu listing tidak ada status awal (API search tidak membawanya),
sehingga tombol favorit **hanya** ditampilkan di halaman detail dan daftar favorit, bukan di ListingCard.
Animasi aktif: `scale` 1 → 1.15 → 1 dalam 200ms (dimatikan oleh reduced motion).

### 6.38 NotificationItem

Sumber: `GET /notifications`. Baris tinggi minimal 72px, padding 16px. Ikon kategori dalam lingkaran 40px
(`booking` → `ReceiptText`, `payment` → `Wallet`, `promo` → `Zap`, `chat` → `Mail`), `title` `title-sm`,
`body` `body` `neutral-700` (maks 3 baris), waktu relatif `caption`.
Belum dibaca (`read_at = null`): latar `brand-50` + titik 8px `brand-600` di kanan. Klik → `PATCH /notifications/{id}/read`
(optimistic), kurangi badge hitungan, lalu buka `link` bila tidak `null`. Tabs di atas: Semua / Booking / Pembayaran / Promo (`category`).

### 6.39 BookingCard & BookingTimeline

**BookingCard** (`GET /booking/my` → `bookings[]`): baris atas badge status (§8.1) + nomor "#{id}" `caption`;
nama kamar `title-sm`; tanggal "1 Des 2026 – 1 Mar 2027" (dari `start_date`, `end_date`); harga `price_label`.
Bila `pending`: InlineAlert warning ringkas "Bayar sebelum {expires_at}". Seluruh kartu tautan ke detail booking.

**BookingTimeline** (detail booking) — ini memang urutan, jadi boleh bernomor/bertitik:

| Langkah | Selesai bila |
|---|---|
| Diajukan | selalu |
| Dibayar | `paid_at` terisi |
| Disetujui pemilik | status `approved`/`completed`/`terminated` |
| Sewa selesai | status `completed` |

Langkah aktif: titik `brand-600` + label 600; selesai: `CircleCheck` `success`; belum: titik `neutral-300`.
Bila `rejected`/`cancelled`/`terminated`: timeline berhenti di langkah terakhir yang tercapai dan ditambah baris
merah berisi status + `reject_reason` (bila ada).

### 6.40 PaymentPanel & Countdown

Muncul di detail booking (`pending`) dan detail invoice (`unpaid`).
- **Countdown:** dari `booking.expires_at` (atau `payment.expires_at` saat transaksi aktif). Format: ≥ 1 jam
  "23 jam 10 menit", < 1 jam "14:32" (`tabular`). Teks: "Selesaikan pembayaran dalam {waktu}". Saat habis:
  muat ulang data. Diperbarui tiap detik hanya bila < 1 jam, selain itu tiap menit. `aria-live` **tidak** dipasang
  pada angka yang terus berubah; umumkan sekali saat sisa 5 menit lewat toast warning.
- **Tombol "Bayar sekarang"** (`success` `lg`): `POST /booking/{id}/checkout` (atau `/invoices/{id}/checkout`) →
  buka Snap dengan `window.snap.pay(snap_token, { onSuccess, onPending, onError, onClose })`.
  Callback Snap **hanya** memicu muat ulang status dari API; status tidak pernah disimpulkan dari callback.
- **Snap.js** dimuat dengan `next/script` (`strategy="afterInteractive"`) hanya di halaman ini:
  sandbox `https://app.sandbox.midtrans.com/snap/snap.js`, production `https://app.midtrans.com/snap/snap.js`,
  atribut `data-client-key={NEXT_PUBLIC_MIDTRANS_CLIENT_KEY}`. Bila skrip gagal dimuat, pakai `redirect_url`.
- Setelah Snap ditutup: tampilkan "Memeriksa status pembayaran…" dan muat ulang data tiap 3 detik maksimal 10 kali.
- Error `502` → toast error dengan aksi "Coba lagi".

### 6.41 InvoiceCard

Sumber: `GET /invoices`. Baris: `number` `caption` + badge status (§8.1); jenis ("Pembayaran pertama", "Tagihan bulanan",
"Perpanjangan") `title-sm`; periode `period_start`–`period_end` (tanggal terakhir ditampilkan = `period_end − 1 hari`);
`total_label` `price`; jatuh tempo `due_date`. Aksi: "Bayar" (bila `unpaid`) dan IconButton `Download` →
`GET /invoices/{id}/pdf` (unduh lewat route handler Next.js agar header Authorization tetap di server).

### 6.42 RefundPanel

Muncul bila `refund ≠ null`. Panel border + InlineAlert sesuai status:
- `awaiting_account`: warning "Isi rekening untuk menerima pengembalian dana {amount_label}." + form (bank, nomor
  rekening, nama pemilik) → `PUT /booking/{id}/refund-account`.
- `ready`: info "Pengembalian dana {amount_label} sedang diproses admin."
- `completed`: success "Dana {amount_label} sudah dikembalikan pada {completed_at}."

### 6.43 DurationPicker

- Bulanan: Chip pilihan dari `monthly_duration_options` kamar ("1 bulan", "3 bulan", …), satu pilihan.
- Harian/mingguan: stepper − angka + (tombol 44px, `aria-label` "Kurangi"/"Tambah"), batas 1–30 hari / 1–12 minggu,
  angka `title-md` `tabular`, input angka juga bisa diketik.
- Di bawahnya caption tanggal selesai: "Sampai {end_date − 1 hari}". Perubahan durasi memuat ulang PriceBreakdown
  (jeda 250ms).

### 6.44 StatCard (owner/admin)

Padding 16px, radius `radius-lg`, latar `neutral-50` (tanpa border, tanpa bayangan). Label `caption` `neutral-600` →
nilai `title-lg` 700 `tabular` → keterangan opsional `caption`. Grid 2 kolom di mobile, 4 di ≥ 1024px.
Semua StatCard sebaris sama tinggi (`h-full`); bila salah satu punya keterangan, yang lain memesan satu baris kosong.
Isi owner: Pendapatan (`earnings_label`), Saldo belum ditransfer (`pending_balance_label`),
Deposit dipegang (`deposits_held_label`), Okupansi (`occupancy.rate` sebagai persen + "{occupied} dari {total} kamar").

### 6.45 MoneyRow (ledger, rincian)

Baris dua sisi: kiri deskripsi `body` + tanggal `caption`; kanan nominal `body-strong` `tabular`.
Nilai positif tanpa tanda, `neutral-900`; negatif diawali "−" (U+2212) warna `neutral-700` (bukan merah — keluar uang
bukan error). `deposit_returned` diberi badge netral "Catatan".

### 6.46 DataTable (owner/admin)

- ≥ 1024px: `<table>` asli, header `caption-strong` `neutral-600` latar `neutral-50`, baris 52px, divider `neutral-200`,
  angka rata kanan `tabular`. Tabel lebar di-scroll dalam wadahnya (`overflow-x-auto`), halaman tidak ikut tergeser.
- < 1024px: setiap baris tampil sebagai item list bertumpuk (label kecil + nilai), bukan tabel menyamping.
- Header kolom yang bisa diurutkan memakai `<button>` + `aria-sort`.
- Ekspor CSV: tombol `neutral` dengan ikon `Download` → `GET /owner/finance/ledger.csv` lewat route handler.

### 6.47 PhotoUploader (owner)

Grid thumbnail 3 kolom (mobile) / 5 (≥ 1024px), rasio 1:1, radius `radius-md`. Petak pertama: tombol tambah
(border putus-putus `neutral-300`, ikon `ImagePlus`, "Tambah foto").

Alur per file (boleh beberapa sekaligus, maks 3 paralel):
1. Validasi klien: JPEG/PNG/WebP, ≤ 10 MB, total foto ≤ 30 → bila gagal, toast error per file.
2. `POST /owner/uploads/presign` → `PUT` ke `upload.url` dengan header persis dari `upload.headers`
   (pakai `XMLHttpRequest` untuk progres).
3. `POST /owner/properties/{id}/photos` (server mengubah ke WebP).

Petak saat proses: pratinjau lokal (`URL.createObjectURL`, dibebaskan setelah selesai) + overlay progres
(lingkaran tipis `brand-600`, persen `caption`). Tahap 3 tampil "Memproses…". Gagal: overlay `danger-bg`, tombol
"Ulangi" & `X`. Selesai: ganti ke `thumbnail_url`. Tiap petak: tombol `Trash2` (ConfirmDialog) dan field caption.
Urutan diubah dengan **seret pegangan**: ikon `GripVertical` (36px, pojok kiri-atas petak, `touch-action: none`,
`aria-label` "Geser foto {n}"). Seret langsung mulai saat pegangan ditekan, di mouse maupun sentuh; gulir halaman tetap
berfungsi di luar pegangan. Saat dilepas: `PATCH /owner/photos/{id}` dengan `sort_order`. Keyboard: fokus pegangan lalu
`Alt` + panah kiri/kanan. Ketuk petak membuka popup "Kelola foto" (tanpa tombol geser).

### 6.48 OtpChannelPicker

Dua pilihan kanal OTP sebaris, **sama lebar (50:50)**, sebelum `POST /auth/otp/request` (`via`):

| Pilihan | Ikon | Label | Nilai `via` |
|---|---|---|---|
| SMS | `MessageSquareText` (Lucide, 20px) | SMS | `"0"` |
| WhatsApp | Logo resmi `/public/brand/whatsapp.svg` (20px) | WhatsApp | `"1"` |

- Bentuk: dua kartu pilihan `role="radio"` dalam `role="radiogroup"` (`aria-label="Kirim kode lewat"`), tinggi 52px,
  border `neutral-300`; terpilih: border 2px `brand-600`, latar `brand-50`. Default: WhatsApp.
- Kedua pilihan sama-sama berikon (konsisten §0.1-B). Logo WhatsApp tidak diwarnai ulang mengikuti tema.
- Saat cooldown aktif, pilihan tetap bisa diganti untuk pengiriman berikutnya.

---

## 7. Pola halaman (wireframe)

### 7.0 Peta route

Path ini juga dipakai backend untuk field `link` notifikasi (SPEC-GO.md §17). Mengubahnya = ubah kedua dokumen.

| Path | Halaman | Akses |
|---|---|---|
| `/` | Beranda | Publik |
| `/cari` | Hasil pencarian (`?area_id=`, filter di query string) | Publik |
| `/kost/{id}` | Detail kost (`?room=` memilih kamar) | Publik |
| `/kost/{id}/ulasan` | Semua ulasan | Publik |
| `/kost/{id}/sewa` | Form ajukan sewa (`?room=`) | Tenant |
| `/masuk`, `/daftar`, `/lupa-password` | Auth | Tamu |
| `/favorit` | Favorit | Tenant |
| `/booking` | Daftar booking | Tenant |
| `/booking/{id}` | Detail booking, pembayaran, tagihan, refund | Tenant |
| `/notifikasi` | Notifikasi | Login |
| `/akun` | Profil, ganti password, ganti nomor, keluar | Login |
| `/owner` | Beranda owner | Owner |
| `/owner/properti`, `/owner/properti/{id}` | Properti, kamar, foto | Owner |
| `/owner/booking`, `/owner/booking/{id}` | Booking masuk | Owner |
| `/owner/keuangan` | Ringkasan (StatCard), lalu tab Buku besar · Payout · Rekening (form rekening hanya di tab Rekening) | Owner |
| `/owner/ulasan` | Ulasan properti | Owner |
| `/admin/...` | Panel admin (struktur mengikuti bagian di API-ADMIN.md) | Admin |

Halaman yang butuh login mengarahkan tamu ke `/masuk?next={path}`. Role yang salah → ErrorState 403
("Halaman ini tidak tersedia untuk akunmu.").

Wireframe mobile 360px. Desktop memperlebar grid sesuai §3.8; struktur sama.

**Beranda (`/`)**
```
[Logo]                    [Bell•]
Cari kost yang pas            ← display, 1 kalimat, tanpa slogan bertumpuk
[ 🔍 Cari kota, kampus, atau nama kost ]
Kota populer
[foto Depok][foto Jakarta Selatan]        ← AreaBannerGrid dari /area/city?top=true, 2 kolom (3 ≥ 640)
Kampus terdekat
[UI Depok] [UGM]                          ← /area/campus?top=true, digeser
Promo minggu ini
[ListingCard][ListingCard]                ← search flash_sale=true, limit 6, digeser horizontal, varian vertical
Kost terbaru
[ListingCard][ListingCard]                ← GET /listings/latest?limit=12, grid 2 kolom (3 ≥ 640, 4 ≥ 1024),
[ListingCard][ListingCard]                   varian vertical, bukan carousel
[SiteFooter]
[BottomNav]
```

**Hasil cari (`/cari?area_id=`)**
```
[<] Kota Depok
[ Filter (2) ][ Urutkan ]            ← FilterBar, ButtonPair 50:50, menempel
[Putri ×][Promo ×]                   ← hanya bila ada filter aktif
120 kamar di 48 kost                 ← caption, dari total & property_total
[ListingCard horizontal]
[ListingCard horizontal]
[ListingCard horizontal]
[Muat lebih banyak]
[BottomNav]
```

**Detail kost (`/kost/{id}`)**
```
[PropertyGallery 4:3]          [<] [♡]   ← tombol mengambang di atas foto
[Campur] [Promo]
Kost Asti Beji                 ← title-lg
Jl. Contoh No. 42, Kota Depok  ← caption + tautan "Buka di peta"
★ 4,5 (12 ulasan)
── Pilih kamar ──
[RoomOption] [RoomOption]
── Fasilitas ──     [FacilityList]
── Tentang kost ──  [Accordion deskripsi]
── Ulasan ──        [RatingSummary][ReviewCard ×3][Lihat semua ulasan]
── Pemilik ──       [avatar inisial] Nama pemilik (`body-strong`)
                    Pemilik Kos · Aktif sejak {bulan tahun} (`caption`)
[StickyActionBar: Mulai dari Rp2.250.000/bulan | Ajukan sewa]
```
Bagian pemilik: pembatas atas `neutral-200` tipis; avatar memakai komponen `Avatar` ukuran `list` (§6.13) seperti
di ulasan, tidak bisa diklik. Data dari `owner.name` dan `owner.active_since`.
Tautan peta: `https://www.google.com/maps/search/?api=1&query={latitude},{longitude}` (tab baru,
ikon `ExternalLink`). Tidak ada peta tertanam di v1 (menjaga bobot halaman).
Setelah halaman tampil, kirim `POST /listings/{id}/view` sekali per sesi per properti.

**Ajukan sewa (`/kost/{id}/sewa?room=`)** — wajib masuk
```
[<] Ajukan sewa
Kamar Tipe A, Kost Asti Beji
Tipe sewa   [Bulanan|Mingguan|Harian]
Durasi      [DurationPicker]
Mulai       [DateField]
[PriceBreakdown]
[StickyActionBar: Total Rp4.250.000 | Ajukan sewa]
```
Sukses → arahkan ke detail booking (status `pending`) dengan toast success dari `meta.message`.

**Detail booking (`/booking/{id}`)**
```
[<] Booking #2                       [Badge status]
[BookingTimeline]
[PaymentPanel]        ← bila pending
[RefundPanel]         ← bila refund ≠ null
Rincian sewa: kamar, tanggal, durasi, harga
Tagihan: daftar InvoiceCard booking ini
[Perpanjang sewa][Tulis ulasan]   ← ButtonPair 50:50 bila keduanya ada; satu saja → lebar penuh
[Batalkan booking] ← bila pending/paid (`danger`)
```

**Masuk & daftar** — rata tengah, maks 400px: logo → judul → form → tombol `success` lebar penuh → tautan alternatif
("Masuk dengan kode OTP", "Lupa password?", "Belum punya akun? Daftar").

**Owner — beranda** — StatCard ×4 → "Perlu direspons" (booking `paid` diurutkan dari `owner_deadline_at` terdekat,
countdown merah bila < 2 jam) → "Properti" (daftar dengan badge moderasi).

---

## 8. Pemetaan API ke UI

### 8.1 Label status

| Entitas | Nilai API | Label | Warna badge |
|---|---|---|---|
| Booking | `pending` | Menunggu pembayaran | warning |
| | `paid` | Menunggu konfirmasi pemilik | info |
| | `approved` | Aktif | success |
| | `completed` | Selesai | netral |
| | `terminated` | Dihentikan | danger |
| | `rejected` | Ditolak | danger |
| | `cancelled` | Dibatalkan | netral |
| Invoice | `unpaid` | Belum dibayar | warning |
| | `paid` | Lunas | success |
| | `void` | Dibatalkan | netral |
| Pembayaran | `pending` | Menunggu pembayaran | warning |
| | `settlement` | Berhasil | success |
| | `expire` | Kedaluwarsa | netral |
| | `cancel` / `deny` / `failure` | Dibatalkan / Ditolak / Gagal | danger |
| Refund | `awaiting_account` | Perlu rekening | warning |
| | `ready` | Diproses | info |
| | `completed` | Selesai | success |
| Payout | `pending` / `completed` | Menunggu transfer / Sudah ditransfer | warning / success |
| Moderasi | `pending` / `approved` / `rejected` / `suspended` | Sedang ditinjau / Tayang / Ditolak / Dinonaktifkan | info / success / danger / danger |

Badge netral: latar `neutral-100`, teks `neutral-600`. Label ini dipusatkan di `lib/labels.ts`; komponen tidak menulis label sendiri.

### 8.2 Respons error ke UI

Semua request lewat satu klien API yang membaca envelope (`status`, `meta.code`, `meta.message`, `messages`).

| Kode | Tampilan |
|---|---|
| `422` | Error per field di bawah input (`messages[field][0]`), fokus ke field error pertama. **Tanpa toast.** Field yang tidak ada di layar → InlineAlert di atas form |
| `400` | Toast error berisi `meta.message`, atau InlineAlert bila terkait langsung dengan isi halaman (mis. kamar penuh di form) |
| `401` (token) | Hapus sesi, arahkan ke `/masuk?next={path}`, toast info `meta.message` |
| `401` (login gagal) | InlineAlert danger di atas form login |
| `403` | Toast error `meta.message`; "Akun dinonaktifkan." → keluar paksa + halaman informasi |
| `404` | ErrorState halaman ("Halaman tidak ditemukan") untuk resource utama; toast untuk aksi |
| `429` | Toast warning `meta.message` (id `rate-limit`), tombol pemicu nonaktif selama `Retry-After` detik |
| `500` / `502` | Toast error `meta.message` dengan aksi "Coba lagi" |
| Jaringan putus | Toast error "Tidak ada koneksi internet. Periksa jaringan lalu coba lagi." |

**Validasi klien:** form memvalidasi dulu di browser dengan aturan yang sama seperti API, memakai pesan bahasa
Indonesia (mis. "Nomor HP harus diawali 08 dan berisi 9–14 digit."). Pesan `messages{}` dari API (bahasa Inggris)
hanya muncul sebagai cadangan bila validasi klien terlewat.

Aksi sukses yang mengubah data → toast success berisi `meta.message` dari API (mis. "Booking disetujui.").
Navigasi biasa dan memuat data → tanpa toast.

### 8.3 Format

| Jenis | Aturan | Contoh |
|---|---|---|
| Uang | Pakai `*_label` dari API. Tanpa label: `"Rp" + n.toLocaleString("id-ID")` (tanpa spasi, sama dengan API) | `Rp2.250.000` |
| Tanggal | `Intl.DateTimeFormat("id-ID", { day: "numeric", month: "short", year: "numeric", timeZone: "Asia/Jakarta" })` | `1 Des 2026` |
| Tanggal panjang | + `weekday: "long"`, `month: "long"` | `Senin, 1 Desember 2026` |
| Jam | `hour: "2-digit", minute: "2-digit"` + " WIB" | `17.05 WIB` |
| Waktu relatif | `Intl.RelativeTimeFormat("id", { numeric: "auto" })`; > 7 hari pakai tanggal | `5 menit yang lalu` |
| Rating | `toLocaleString("id-ID", { minimumFractionDigits: 1, maximumFractionDigits: 1 })` | `4,5` |
| Persen | `Math.round(rate * 100) + "%"` | `40%` |
| Nomor HP tampilan | Kelompok 4-4-sisa | `0812 0000 0002` |

Tanggal `YYYY-MM-DD` dari API diperlakukan sebagai tanggal WIB (jangan di-parse sebagai UTC lalu digeser).
Tanggal berakhir yang ditampilkan ke pengguna = `end_date`/`period_end` **dikurangi 1 hari** (keduanya eksklusif).

---

## 9. Penulisan

- *Sentence case* di semua judul, tombol, dan label. Tidak ada huruf kapital semua.
- Sapaan "kamu" untuk pengguna; kalimat aktif, pendek, tanpa basa-basi dan tanpa tanda seru berlebihan.
- Tombol menyebut hasil aksinya, dan toast memakai kata yang sama: tombol "Simpan kamar" → toast "Kamar diperbarui."
  diambil dari `meta.message` API.
- Error menjelaskan apa yang terjadi dan cara memperbaikinya; tidak meminta maaf, tidak samar.
- Empty state mengajak bertindak (§6.18), bukan sekadar "Tidak ada data".
- Tidak memakai pemisah titik tengah ("A · B") atau tanda panah di label; pakai badge atau baris terpisah.
- Istilah tetap: **kost** (bukan kos/indekos), **sewa**, **tagihan**, **deposit**, **biaya admin**, **pemilik**, **penyewa**.

---

## 10. Checklist aksesibilitas (wajib per komponen)

- [ ] Kontras teks ≥ 4,5:1 (teks ≥ 24px atau ≥ 18,66px tebal: ≥ 3:1); komponen UI & ikon bermakna ≥ 3:1.
- [ ] Target sentuh ≥ 44×44px; jarak antar target ≥ 8px.
- [ ] Semua aksi bisa dicapai keyboard; urutan fokus mengikuti urutan visual; fokus terlihat (outline `brand-600`).
- [ ] Dialog/sheet mengunci fokus dan mengembalikannya ke pemicu.
- [ ] Status tidak hanya dibedakan dengan warna (selalu ada teks/ikon).
- [ ] Gambar bermakna punya `alt`; gambar dekoratif `alt=""`.
- [ ] Input punya `<label>`; error terhubung lewat `aria-describedby`.
- [ ] Perubahan dinamis penting diumumkan (toast `role="status"`/`alert`), angka yang terus berubah tidak.
- [ ] Bekerja pada zoom 200% dan lebar 320px tanpa scroll horizontal halaman.
- [ ] Animasi mati dengan `prefers-reduced-motion`.
- [ ] `lang="id"` di `<html>`; judul halaman (`<title>`) unik per halaman.
- [ ] Kursor pointer di semua elemen yang bisa diklik; `not-allowed` saat nonaktif (§0.1-A).
- [ ] Pasangan tombol 50:50, ikon hanya sesuai daftar (§0.1-B).
- [ ] Logo merek dari file resmi, bukan ikon pengganti (§0.1-C).
- [ ] Kartu/kolom bersebelahan sama tinggi, harga & aksi sejajar (§0.1-D).
- [ ] Tidak ada scroll horizontal halaman di 320–430px (§0.1-E).

---

## 11. Perubahan API yang diminta frontend

Field berikut ditambahkan ke kontrak atas permintaan design system ini dan **sudah tercantum** di `API.md`
serta `SPEC-GO.md`. Bila nanti komponen butuh field lain, ikuti pola yang sama: ubah dokumen API dulu, lalu kode.

| Endpoint | Field | Dipakai oleh |
|---|---|---|
| `POST /listings/search` → `rooms[]` | `property_id`, `thumbnail_url`, `area_name` | ListingCard |
| `GET /suggestion` → `landings[]` | `id` | SuggestionList |
| `GET /suggestion` → `room_types[]` | `property_id` | SuggestionList |
| `GET /listings/{id}` → `property` | `is_favorite` | FavoriteButton |
| `GET /notifications` → item | `link` | NotificationItem |

Tidak termasuk v1: peta interaktif, mode gelap, chat, notifikasi push. Token warna sudah berupa variabel sehingga
mode gelap bisa ditambahkan nanti tanpa mengubah komponen.
