# IMPLEMENTATION.md â€” Daftar implementasi

> Urutan kerja resmi untuk coding agent. Apa yang dibangun: `PRD.md`. Bagaimana: dokumen kontrak API, `SPEC-GO.md`,
> `DESIGN.md`. Proses kerja: `AGENT.md`.

## Cara memakai

- Kerjakan **berurutan** sesuai Â§1. Jangan memulai tugas yang dependensinya belum selesai.
- Satu tugas boleh beberapa commit (AGENT.md Â§3). Di commit terakhir tugas, ubah `- [ ]` menjadi `- [x]` di file ini.
- Tugas bertanda **â›” menunggu pemilik** tidak dikerjakan sebelum keputusan di `PRD.md` Â§8 tersedia; lewati dan lanjut.
- Menemukan kebutuhan yang tidak tercakup â†’ jangan menambah tugas sendiri; laporkan di catatan laporan (AGENT.md Â§10).

**Definisi selesai (berlaku untuk setiap tugas):**
1. Pemeriksaan `AGENT.md` Â§6 lolos (lint, typecheck, test, build untuk bagian yang disentuh).
2. Setiap endpoint yang dibuat/diubah punya contract test sesuai `SPEC-GO.md` Â§11 dan contoh di dokumen kontrak.
3. Setiap komponen UI mengikuti `DESIGN.md` (token, Â§0.1, checklist Â§10) dan punya test perilaku dasar.
4. Tidak ada `TODO` tanpa `(#issue)`, tidak ada kode yang dikomentari, tidak ada rahasia di diff.
5. Di-commit dan di-push ke `main`, blok deploy ditampilkan (AGENT.md Â§7, Â§9).

## 1. Urutan

```
R0 â†’ B1 â†’ F0 â†’ B2 â†’ F1 â†’ B3 â†’ F2 â†’ B4 â†’ F3 â†’ B5 â†’ F4 â†’ B6 â†’ F5 â†’ B7 â†’ Q1  (Rilis 1)
   â†’ B8 â†’ F6 â†’ Q2                                                           (Rilis 2)
   â†’ B9 â†’ F7 â†’ Q3                                                           (Rilis 3)
```

---

## R0 â€” Repository

- [x] **R0.1 Struktur monorepo.** Buat struktur `AGENT.md` Â§2; pindahkan semua dokumen ke `docs/` kecuali `AGENT.md`
  dan `CLAUDE.md` (root). `.gitignore` untuk Go, Node/Next.js, `.env*` (kecuali `.env.example`), artefak build.
  *Selesai bila:* pohon folder sama dengan AGENT.md Â§2.
- [x] **R0.2 Hook & konfigurasi agent.** `.githooks/commit-msg`, `.githooks/pre-commit` (executable, isi AGENT.md Â§8),
  `.claude/settings.json` (cocokkan nama setting dengan dokumentasi Claude Code terbaru), `CLAUDE.md`,
  lalu `git config core.hooksPath .githooks` (lokal).
  *Selesai bila:* pesan berisi `Co-authored-by: Claude` ditolak hook (laporkan hasil uji), file `.env` ditolak pre-commit.
- [x] **R0.3 Lingkungan dev.** `docker-compose.yml` di root: Postgres, Redis, MinIO (+ pembuatan bucket dan lifecycle
  `uploads/` 1 hari), plus `backend/.env.example` dan `frontend/.env.example`.
  *Selesai bila:* `docker compose up -d` sehat dan semua layanan bisa dihubungi dari backend & frontend lokal.
- [ ] **R0.4 Skrip deploy.** â›” menunggu pemilik (PRD Â§8 #5). `deploy/install.sh` milik pemilik; agent hanya membuat/mengubah
  bila diminta dengan spesifikasi target server.

## B1 â€” Fondasi backend

- [x] **B1.1 Skeleton Go.** `go.mod` (`go 1.27.0`, `toolchain go1.27.3`), staticcheck sebagai tool, konfigurasi env
  (SPEC Â§3), `cmd/api`, `cmd/worker`, `cmd/seed`, `cmd/admin` dengan shutdown rapi (SIGTERM), Dockerfile
  multi-stage Debian + libvips (SPEC Â§3). *Selesai bila:* `go build ./...` dan image Docker berhasil dibuat.
- [x] **B1.2 Skema database.** Migration goose untuk **seluruh** skema SPEC Â§4 (termasuk tabel Rilis 2â€“3, trigger
  `storage_deletions`, trigger append-only `ledger_entries`/`audit_logs`). Dua role DB: pemilik migration dan role
  aplikasi tanpa hak `UPDATE/DELETE/TRUNCATE` pada ledger & audit.
  *Selesai bila:* `up` dari DB kosong berhasil; test membuktikan UPDATE/DELETE ledger & audit ditolak; hapus kamar
  memasukkan key foto ke `storage_deletions`.
- [x] **B1.3 Envelope & error.** Paket `response`, peta severity, handler 404/405/500, pesan error SPEC Â§5.3.
- [x] **B1.4 Validasi.** Translator pesan SPEC Â§5.4 + aturan evaluasi (berhenti setelah `required`/`accepted` gagal).
  *Selesai bila:* test tabel untuk setiap rule dan setiap atribut bertitik (`filters.area_id`, `facilities.0`).
- [x] **B1.5 Uang & waktu.** Format Rupiah, helper WIB, aritmetika periode SPEC Â§13.6 (penjepitan akhir bulan).
  *Selesai bila:* test mencakup mulai 31 Jan, tahun kabisat, pergantian tahun.
- [x] **B1.6 Middleware.** Request ID, recovery, CORS, client check, limiter `api` & `otp` (Redis) + header 429,
  auth wajib/opsional, cek role, cek akun dinonaktifkan. Pengecualian webhook (SPEC Â§5.1).
- [x] **B1.7 Ledger & audit.** `finance.Record` dan `audit.Record` dalam transaksi pemanggil, penyamaran field sensitif.
- [x] **B1.8 Notifikasi & email outbox.** Helper pembuat notifikasi (judul, isi, `link` SPEC Â§17) + insert `email_outbox`
  bila penerima punya email; `email.Sender` dengan `LogSender`.
- [x] **B1.9 Seed & admin CLI.** `cmd/seed` (SPEC Â§10, menolak di production), `cmd/admin create` (SPEC Â§16).
- [x] **B1.10 Harness contract test.** Menjalankan API terhadap DB hasil seed; pembanding JSON semantik (SPEC Â§11);
  helper field dinamis. *Selesai bila:* satu contoh fixture dari API.md berjalan hijau.

## F0 â€” Fondasi frontend

- [x] **F0.1 Setup Next.js.** TypeScript strict, Tailwind v4 + `globals.css` dari DESIGN Â§3.10, Poppins, ESLint, Vitest,
  Playwright, script wajib (DESIGN Â§2), `next/image` remote pattern dari env.
- [x] **F0.2 BFF & klien API.** Route handler yang meneruskan ke backend dengan `x-api-client` + token dari cookie
  httpOnly; login/logout set/hapus cookie; parser envelope; pemetaan error DESIGN Â§8.2; `lib/labels.ts` (Â§8.1) dan
  helper format (Â§8.3). *Selesai bila:* test unit pemetaan setiap kode error & format.
- [x] **F0.3 Komponen dasar.** DESIGN Â§6.1â€“6.27 dan Â§6.48 + `ButtonPair` + Toaster. *Selesai bila:* test peran ARIA,
  navigasi keyboard, `ButtonPair` 50:50, toast (antre maks 3, jeda saat hover).
- [x] **F0.4 Kerangka layout.** AppHeader, BottomNav per role, `template.tsx`, guard route sesuai DESIGN Â§7.0,
  ErrorState 403/404.

## B2 â€” Auth, akun, notifikasi

- [x] **B2.1 Registrasi, OTP, login, token.** API.md Â§2; SPEC Â§6 (Redis OTP, cooldown, debug echo + guard production).
- [x] **B2.2 Akun.** API.md Â§11 (logout, reset password, `/me`, ganti password, ganti nomor).
- [x] **B2.3 Notifikasi.** API.md Â§6 (daftar dengan `link`, counter, tandai dibaca + 403).

## F1 â€” Auth, akun, notifikasi (UI)

- [x] **F1.1 Halaman auth.** `/daftar`, `/masuk` (password & OTP), `/lupa-password`, OtpInput + OtpChannelPicker.
- [x] **F1.2 Akun.** `/akun`: profil, ganti password, ganti nomor, keluar (ConfirmDialog).
- [x] **F1.3 Notifikasi.** `/notifikasi` + badge hitungan di header; klik membuka `link`.

## B3 â€” Penemuan

- [x] **B3.1 Area & suggestion.** API.md Â§3.1â€“3.2 (termasuk `landings[].id`, `room_types[].property_id`).
- [x] **B3.2 Pencarian.** API.md Â§3.3: semua filter termasuk `area_id` (radius kampus/MRT), urutan, `has-more`,
  `thumbnail_url`, `area_name`, rating. *Selesai bila:* `EXPLAIN` query utama memakai index pada data seed 10.000 properti
  (generator data uji, bukan seed default).
- [x] **B3.3 Detail.** API.md Â§4 (`is_favorite` dengan auth opsional, foto, rating), view + `property_views_daily`,
  gallery, A/B test API.md Â§7.

## F2 â€” Penemuan (UI)

- [x] **F2.1 Beranda.** DESIGN Â§7 beranda: SearchBar, chip area populer, promo.
- [x] **F2.2 Pencarian.** `/cari`: SuggestionList, FilterBar, FilterSheet, SortSheet, ListingCard (horizontal di mobile),
  LoadMore, skeleton, empty state.
- [x] **F2.3 Detail kost.** `/kost/{id}`: PropertyGallery, RoomOption, FacilityList, PriceBreakdown, StickyActionBar,
  RatingSummary (tanpa input ulasan), FavoriteButton disembunyikan sampai B9 selesai, tautan peta.

## B4 â€” Booking & pembayaran

- [x] **B4.1 Draft-price & setting.** API.md Â§5.1 (`duration`), baca `admin_fee` dari `settings` dengan cache (SPEC Â§16).
- [x] **B4.2 Submit, daftar, detail booking.** API.md Â§5.2â€“5.4: `duration`, `end_date`, invoice `initial` dalam transaksi.
- [x] **B4.3 Klien Midtrans.** SPEC Â§13.1 di balik interface + mock untuk test.
- [x] **B4.4 Checkout.** SPEC Â§13.2 untuk booking & invoice. *Selesai bila:* test konkurensi â€” dua checkout bersamaan
  untuk kamar terakhir, hanya satu berhasil; checkout ulang mengembalikan transaksi aktif yang sama.
- [x] **B4.5 Webhook.** SPEC Â§13.3: signature, cek status, `ApplyPaymentStatus` semua baris tabel, ledger.
  *Selesai bila:* test notifikasi duplikat, tidak berurutan, nominal beda, bayar setelah kunci lepas.
- [x] **B4.6 Batal & refund.** API.md Â§5.6â€“5.7, SPEC Â§13.4â€“13.5.
- [x] **B4.7 Invoice tenant.** API.md Â§12.1â€“12.3 (tanpa PDF dan tanpa perpanjangan).
- [x] **B4.8 Worker dasar.** SPEC Â§14: `expire_payments`, `expire_bookings`, `send_emails` (LogSender). Job harian & advisory lock-nya menunggu fitur owner/admin (lihat DECISIONS.md).

## F3 â€” Booking & pembayaran (UI)

- [x] **F3.1 Form sewa.** `/kost/{id}/sewa`: SegmentedControl tipe sewa, DurationPicker, DateField (min hari ini WIB),
  PriceBreakdown, submit.
- [x] **F3.2 Booking.** `/booking`, `/booking/{id}`: BookingCard, BookingTimeline, PaymentPanel + Snap.js, Countdown,
  RefundPanel + form rekening, InvoiceCard (tanpa unduh PDF), batal (ConfirmDialog).
- [x] **F3.3 E2E Rilis 1 tenant.** Playwright: daftar â†’ OTP (debug echo) â†’ cari â†’ detail â†’ sewa â†’ checkout (Snap di-mock
  di level BFF) â†’ status `paid` setelah webhook uji.

## B5 â€” Pemilik

- [x] **B5.1 Storage & media.** Klien MinIO/S3, presign, pengolahan gambar SPEC Â§18 (libvips), foto API-OWNER.md Â§4,
  job `purge_storage`, `cleanup_uploads`, `reconcile_storage`. *Selesai bila:* test JPEG/PNG/WebP â†’ dua WebP benar,
  EXIF/GPS hilang, file rusak 400, retry hapus saat storage gagal.
- [x] **B5.2 Properti.** API-OWNER.md Â§2 (moderasi, slug, ajukan ulang).
- [x] **B5.3 Kamar.** API-OWNER.md Â§3 (`held`, `occupied`, `monthly_duration_options`).
- [x] **B5.4 Booking masuk.** API-OWNER.md Â§5.1â€“5.4 + payout & ledger saat disetujui; job `owner_timeout`, `owner_reminder`.
- [x] **B5.5 Rekening, payout, invoice owner.** API-OWNER.md Â§5.6, Â§6.

## F4 â€” Pemilik (UI)

- [x] **F4.1 Beranda owner.** `/owner`: daftar "Perlu direspons" dengan countdown + properti (StatCard menyusul di F6).
- [x] **F4.2 Properti & kamar.** `/owner/properti`, `/owner/properti/{id}`: form, Switch aktif/promo, status moderasi,
  InlineAlert alasan ditolak, form kamar (durasi bulanan, Rupiah).
- [x] **F4.3 Foto.** PhotoUploader (DESIGN Â§6.47).
- [x] **F4.4 Booking masuk & keuangan dasar.** `/owner/booking/{id}` (setujui/tolak), `/owner/keuangan` bagian rekening
  & payout.

## B6 â€” Admin

- [x] **B6.1 Area & pengguna.** API-ADMIN.md Â§2â€“3.
- [x] **B6.2 Moderasi, booking, invoice.** API-ADMIN.md Â§4â€“5, Â§12 (invoice).
- [x] **B6.3 Refund, payout, setting, broadcast.** API-ADMIN.md Â§6â€“9.

## F5 â€” Admin (UI)

- [x] **F5.1 Panel admin.** Layout desktop-first yang tetap responsif, DataTable, halaman untuk setiap bagian B6.

## B7 â€” Siklus sewa (wajib Rilis 1)

- [x] **B7.1 Tagihan bulanan & akhir sewa.** Job `issue_renewals`, `renewal_due_reminder`, `terminate_overdue`,
  `complete_tenancies` (SPEC Â§14) + `ApplyPaymentStatus` untuk invoice `renewal`.
  *Selesai bila:* test mencakup H-3, H+3, pelepasan kamar, pembayaran setelah `void` â†’ refund.

## Q1 â€” Gerbang Rilis 1

- [x] **Q1.1 Contract test lengkap** untuk semua endpoint Rilis 1.
- [x] **Q1.2 E2E** tenant (F3.3), owner (buat properti â†’ admin setujui â†’ terima booking), admin (refund & payout).
- [x] **Q1.3 Uji beban ringan** pencarian, detail, checkout, unggah foto; target PRD Â§7.
- [x] **Q1.4 Tinjauan keamanan:** signature webhook, rate limit, cookie httpOnly/secure/SameSite, tidak ada rahasia di repo,
  header keamanan dasar.
- [ ] **Q1.5 Siap produksi.** â›” menunggu pemilik: PRD Â§8 #1, #2, #3, #5, #7, #8.

## B8 â€” Rilis 2 backend

- [x] **B8.1 Perpanjangan.** API.md Â§12.5, job `void_extensions`, `ApplyPaymentStatus` untuk `extension`.
- [x] **B8.2 PDF invoice.** API.md Â§12.4, SPEC Â§13.7.
- [x] **B8.3 Deposit.** API-OWNER.md Â§5.5.
- [x] **B8.4 Keuangan & statistik owner.** API-OWNER.md Â§8â€“9 (CSV streaming).
- [x] **B8.5 Keuangan admin & rekonsiliasi.** API-ADMIN.md Â§10, job `reconcile_payments`.
- [x] **B8.6 Audit log.** API-OWNER.md Â§11, API-ADMIN.md Â§12.

## F6 â€” Rilis 2 UI

- [x] **F6.1 Tenant.** Perpanjang sewa (DurationPicker + PaymentPanel), unduh PDF di InvoiceCard.
- [x] **F6.2 Owner.** StatCard di beranda, `/owner/keuangan` (ringkasan, ledger MoneyRow, CSV), catat deposit, audit log.
- [x] **F6.3 Admin.** Laporan keuangan, ledger, rekonsiliasi, audit log.

## Q2 â€” Gerbang Rilis 2

- [x] **Q2.1** Contract test & E2E fitur Rilis 2; uji perpanjangan bulanan & harian end-to-end dengan jam yang dimajukan.

## B9 â€” Rilis 3 backend

- [x] **B9.1 Ulasan.** API.md Â§13.1â€“13.2, API-OWNER.md Â§10, API-ADMIN.md Â§11, cache rating, `can_review`.
- [x] **B9.2 Favorit.** API.md Â§13.3.
- [x] **B9.3 Email produksi.** â›” menunggu pemilik (PRD Â§8 #4): pengirim SMTP nyata.

## F7 â€” Rilis 3 UI

- [x] **F7.1 Ulasan.** RatingStars input, ReviewCard, `/kost/{id}/ulasan`, `/owner/ulasan`, moderasi admin.
- [x] **F7.2 Favorit.** FavoriteButton di detail, `/favorit`.

## Q3 â€” Gerbang Rilis 3

- [x] **Q3.1** Contract test & E2E ulasan dan favorit; uji email lewat SMTP sandbox setelah B9.3.
