# IMPLEMENTATION.md — Daftar implementasi

> Urutan kerja resmi untuk coding agent. Apa yang dibangun: `PRD.md`. Bagaimana: dokumen kontrak API, `SPEC-GO.md`,
> `DESIGN.md`. Proses kerja: `AGENT.md`.

## Cara memakai

- Kerjakan **berurutan** sesuai §1. Jangan memulai tugas yang dependensinya belum selesai.
- Satu tugas boleh beberapa commit (AGENT.md §3). Di commit terakhir tugas, ubah `- [ ]` menjadi `- [x]` di file ini.
- Tugas bertanda **⛔ menunggu pemilik** tidak dikerjakan sebelum keputusan di `PRD.md` §8 tersedia; lewati dan lanjut.
- Menemukan kebutuhan yang tidak tercakup → jangan menambah tugas sendiri; laporkan di catatan laporan (AGENT.md §10).

**Definisi selesai (berlaku untuk setiap tugas):**
1. Pemeriksaan `AGENT.md` §6 lolos (lint, typecheck, test, build untuk bagian yang disentuh).
2. Setiap endpoint yang dibuat/diubah punya contract test sesuai `SPEC-GO.md` §11 dan contoh di dokumen kontrak.
3. Setiap komponen UI mengikuti `DESIGN.md` (token, §0.1, checklist §10) dan punya test perilaku dasar.
4. Tidak ada `TODO` tanpa `(#issue)`, tidak ada kode yang dikomentari, tidak ada rahasia di diff.
5. Di-commit dan di-push ke `main`, blok deploy ditampilkan (AGENT.md §7, §9).

## 1. Urutan

```
R0 → B1 → F0 → B2 → F1 → B3 → F2 → B4 → F3 → B5 → F4 → B6 → F5 → B7 → Q1  (Rilis 1)
   → B8 → F6 → Q2                                                           (Rilis 2)
   → B9 → F7 → Q3                                                           (Rilis 3)
```

---

## R0 — Repository

- [x] **R0.1 Struktur monorepo.** Buat struktur `AGENT.md` §2; pindahkan semua dokumen ke `docs/` kecuali `AGENT.md`
  dan `CLAUDE.md` (root). `.gitignore` untuk Go, Node/Next.js, `.env*` (kecuali `.env.example`), artefak build.
  *Selesai bila:* pohon folder sama dengan AGENT.md §2.
- [x] **R0.2 Hook & konfigurasi agent.** `.githooks/commit-msg`, `.githooks/pre-commit` (executable, isi AGENT.md §8),
  `.claude/settings.json` (cocokkan nama setting dengan dokumentasi Claude Code terbaru), `CLAUDE.md`,
  lalu `git config core.hooksPath .githooks` (lokal).
  *Selesai bila:* pesan berisi `Co-authored-by: Claude` ditolak hook (laporkan hasil uji), file `.env` ditolak pre-commit.
- [x] **R0.3 Lingkungan dev.** `docker-compose.yml` di root: Postgres, Redis, MinIO (+ pembuatan bucket dan lifecycle
  `uploads/` 1 hari), plus `backend/.env.example` dan `frontend/.env.example`.
  *Selesai bila:* `docker compose up -d` sehat dan semua layanan bisa dihubungi dari backend & frontend lokal.
- [ ] **R0.4 Skrip deploy.** ⛔ menunggu pemilik (PRD §8 #5). `deploy/install.sh` milik pemilik; agent hanya membuat/mengubah
  bila diminta dengan spesifikasi target server.

## B1 — Fondasi backend

- [x] **B1.1 Skeleton Go.** `go.mod` (`go 1.27.0`, `toolchain go1.27.3`), staticcheck sebagai tool, konfigurasi env
  (SPEC §3), `cmd/api`, `cmd/worker`, `cmd/seed`, `cmd/admin` dengan shutdown rapi (SIGTERM), Dockerfile
  multi-stage Debian + libvips (SPEC §3). *Selesai bila:* `go build ./...` dan image Docker berhasil dibuat.
- [x] **B1.2 Skema database.** Migration goose untuk **seluruh** skema SPEC §4 (termasuk tabel Rilis 2–3, trigger
  `storage_deletions`, trigger append-only `ledger_entries`/`audit_logs`). Dua role DB: pemilik migration dan role
  aplikasi tanpa hak `UPDATE/DELETE/TRUNCATE` pada ledger & audit.
  *Selesai bila:* `up` dari DB kosong berhasil; test membuktikan UPDATE/DELETE ledger & audit ditolak; hapus kamar
  memasukkan key foto ke `storage_deletions`.
- [x] **B1.3 Envelope & error.** Paket `response`, peta severity, handler 404/405/500, pesan error SPEC §5.3.
- [x] **B1.4 Validasi.** Translator pesan SPEC §5.4 + aturan evaluasi (berhenti setelah `required`/`accepted` gagal).
  *Selesai bila:* test tabel untuk setiap rule dan setiap atribut bertitik (`filters.area_id`, `facilities.0`).
- [x] **B1.5 Uang & waktu.** Format Rupiah, helper WIB, aritmetika periode SPEC §13.6 (penjepitan akhir bulan).
  *Selesai bila:* test mencakup mulai 31 Jan, tahun kabisat, pergantian tahun.
- [x] **B1.6 Middleware.** Request ID, recovery, CORS, client check, limiter `api` & `otp` (Redis) + header 429,
  auth wajib/opsional, cek role, cek akun dinonaktifkan. Pengecualian webhook (SPEC §5.1).
- [x] **B1.7 Ledger & audit.** `finance.Record` dan `audit.Record` dalam transaksi pemanggil, penyamaran field sensitif.
- [x] **B1.8 Notifikasi & email outbox.** Helper pembuat notifikasi (judul, isi, `link` SPEC §17) + insert `email_outbox`
  bila penerima punya email; `email.Sender` dengan `LogSender`.
- [x] **B1.9 Seed & admin CLI.** `cmd/seed` (SPEC §10, menolak di production), `cmd/admin create` (SPEC §16).
- [x] **B1.10 Harness contract test.** Menjalankan API terhadap DB hasil seed; pembanding JSON semantik (SPEC §11);
  helper field dinamis. *Selesai bila:* satu contoh fixture dari API.md berjalan hijau.

## F0 — Fondasi frontend

- [x] **F0.1 Setup Next.js.** TypeScript strict, Tailwind v4 + `globals.css` dari DESIGN §3.10, Poppins, ESLint, Vitest,
  Playwright, script wajib (DESIGN §2), `next/image` remote pattern dari env.
- [x] **F0.2 BFF & klien API.** Route handler yang meneruskan ke backend dengan `x-api-client` + token dari cookie
  httpOnly; login/logout set/hapus cookie; parser envelope; pemetaan error DESIGN §8.2; `lib/labels.ts` (§8.1) dan
  helper format (§8.3). *Selesai bila:* test unit pemetaan setiap kode error & format.
- [x] **F0.3 Komponen dasar.** DESIGN §6.1–6.27 dan §6.48 + `ButtonPair` + Toaster. *Selesai bila:* test peran ARIA,
  navigasi keyboard, `ButtonPair` 50:50, toast (antre maks 3, jeda saat hover).
- [x] **F0.4 Kerangka layout.** AppHeader, BottomNav per role, `template.tsx`, guard route sesuai DESIGN §7.0,
  ErrorState 403/404.

## B2 — Auth, akun, notifikasi

- [x] **B2.1 Registrasi, OTP, login, token.** API.md §2; SPEC §6 (Redis OTP, cooldown, debug echo + guard production).
- [x] **B2.2 Akun.** API.md §11 (logout, reset password, `/me`, ganti password, ganti nomor).
- [x] **B2.3 Notifikasi.** API.md §6 (daftar dengan `link`, counter, tandai dibaca + 403).

## F1 — Auth, akun, notifikasi (UI)

- [x] **F1.1 Halaman auth.** `/daftar`, `/masuk` (password & OTP), `/lupa-password`, OtpInput + OtpChannelPicker.
- [x] **F1.2 Akun.** `/akun`: profil, ganti password, ganti nomor, keluar (ConfirmDialog).
- [x] **F1.3 Notifikasi.** `/notifikasi` + badge hitungan di header; klik membuka `link`.

## B3 — Penemuan

- [x] **B3.1 Area & suggestion.** API.md §3.1–3.2 (termasuk `landings[].id`, `room_types[].property_id`).
- [x] **B3.2 Pencarian.** API.md §3.3: semua filter termasuk `area_id` (radius kampus/MRT), urutan, `has-more`,
  `thumbnail_url`, `area_name`, rating. *Selesai bila:* `EXPLAIN` query utama memakai index pada data seed 10.000 properti
  (generator data uji, bukan seed default).
- [x] **B3.3 Detail.** API.md §4 (`is_favorite` dengan auth opsional, foto, rating), view + `property_views_daily`,
  gallery, A/B test API.md §7.

## F2 — Penemuan (UI)

- [x] **F2.1 Beranda.** DESIGN §7 beranda: SearchBar, chip area populer, promo.
- [x] **F2.2 Pencarian.** `/cari`: SuggestionList, FilterBar, FilterSheet, SortSheet, ListingCard (horizontal di mobile),
  LoadMore, skeleton, empty state.
- [x] **F2.3 Detail kost.** `/kost/{id}`: PropertyGallery, RoomOption, FacilityList, PriceBreakdown, StickyActionBar,
  RatingSummary (tanpa input ulasan), FavoriteButton disembunyikan sampai B9 selesai, tautan peta.

## B4 — Booking & pembayaran

- [x] **B4.1 Draft-price & setting.** API.md §5.1 (`duration`), baca `admin_fee` dari `settings` dengan cache (SPEC §16).
- [x] **B4.2 Submit, daftar, detail booking.** API.md §5.2–5.4: `duration`, `end_date`, invoice `initial` dalam transaksi.
- [x] **B4.3 Klien Midtrans.** SPEC §13.1 di balik interface + mock untuk test.
- [x] **B4.4 Checkout.** SPEC §13.2 untuk booking & invoice. *Selesai bila:* test konkurensi — dua checkout bersamaan
  untuk kamar terakhir, hanya satu berhasil; checkout ulang mengembalikan transaksi aktif yang sama.
- [x] **B4.5 Webhook.** SPEC §13.3: signature, cek status, `ApplyPaymentStatus` semua baris tabel, ledger.
  *Selesai bila:* test notifikasi duplikat, tidak berurutan, nominal beda, bayar setelah kunci lepas.
- [x] **B4.6 Batal & refund.** API.md §5.6–5.7, SPEC §13.4–13.5.
- [x] **B4.7 Invoice tenant.** API.md §12.1–12.3 (tanpa PDF dan tanpa perpanjangan).
- [x] **B4.8 Worker dasar.** SPEC §14: `expire_payments`, `expire_bookings`, `send_emails` (LogSender). Job harian & advisory lock-nya menunggu fitur owner/admin (lihat DECISIONS.md).

## F3 — Booking & pembayaran (UI)

- [x] **F3.1 Form sewa.** `/kost/{id}/sewa`: SegmentedControl tipe sewa, DurationPicker, DateField (min hari ini WIB),
  PriceBreakdown, submit.
- [ ] **F3.2 Booking.** `/booking`, `/booking/{id}`: BookingCard, BookingTimeline, PaymentPanel + Snap.js, Countdown,
  RefundPanel + form rekening, InvoiceCard (tanpa unduh PDF), batal (ConfirmDialog).
- [ ] **F3.3 E2E Rilis 1 tenant.** Playwright: daftar → OTP (debug echo) → cari → detail → sewa → checkout (Snap di-mock
  di level BFF) → status `paid` setelah webhook uji.

## B5 — Pemilik

- [ ] **B5.1 Storage & media.** Klien MinIO/S3, presign, pengolahan gambar SPEC §18 (libvips), foto API-OWNER.md §4,
  job `purge_storage`, `cleanup_uploads`, `reconcile_storage`. *Selesai bila:* test JPEG/PNG/WebP → dua WebP benar,
  EXIF/GPS hilang, file rusak 400, retry hapus saat storage gagal.
- [ ] **B5.2 Properti.** API-OWNER.md §2 (moderasi, slug, ajukan ulang).
- [ ] **B5.3 Kamar.** API-OWNER.md §3 (`held`, `occupied`, `monthly_duration_options`).
- [ ] **B5.4 Booking masuk.** API-OWNER.md §5.1–5.4 + payout & ledger saat disetujui; job `owner_timeout`, `owner_reminder`.
- [ ] **B5.5 Rekening, payout, invoice owner.** API-OWNER.md §5.6, §6.

## F4 — Pemilik (UI)

- [ ] **F4.1 Beranda owner.** `/owner`: daftar "Perlu direspons" dengan countdown + properti (StatCard menyusul di F6).
- [ ] **F4.2 Properti & kamar.** `/owner/properti`, `/owner/properti/{id}`: form, Switch aktif/promo, status moderasi,
  InlineAlert alasan ditolak, form kamar (durasi bulanan, Rupiah).
- [ ] **F4.3 Foto.** PhotoUploader (DESIGN §6.47).
- [ ] **F4.4 Booking masuk & keuangan dasar.** `/owner/booking/{id}` (setujui/tolak), `/owner/keuangan` bagian rekening
  & payout.

## B6 — Admin

- [ ] **B6.1 Area & pengguna.** API-ADMIN.md §2–3.
- [ ] **B6.2 Moderasi, booking, invoice.** API-ADMIN.md §4–5, §12 (invoice).
- [ ] **B6.3 Refund, payout, setting, broadcast.** API-ADMIN.md §6–9.

## F5 — Admin (UI)

- [ ] **F5.1 Panel admin.** Layout desktop-first yang tetap responsif, DataTable, halaman untuk setiap bagian B6.

## B7 — Siklus sewa (wajib Rilis 1)

- [ ] **B7.1 Tagihan bulanan & akhir sewa.** Job `issue_renewals`, `renewal_due_reminder`, `terminate_overdue`,
  `complete_tenancies` (SPEC §14) + `ApplyPaymentStatus` untuk invoice `renewal`.
  *Selesai bila:* test mencakup H-3, H+3, pelepasan kamar, pembayaran setelah `void` → refund.

## Q1 — Gerbang Rilis 1

- [ ] **Q1.1 Contract test lengkap** untuk semua endpoint Rilis 1.
- [ ] **Q1.2 E2E** tenant (F3.3), owner (buat properti → admin setujui → terima booking), admin (refund & payout).
- [ ] **Q1.3 Uji beban ringan** pencarian, detail, checkout, unggah foto; target PRD §7.
- [ ] **Q1.4 Tinjauan keamanan:** signature webhook, rate limit, cookie httpOnly/secure/SameSite, tidak ada rahasia di repo,
  header keamanan dasar.
- [ ] **Q1.5 Siap produksi.** ⛔ menunggu pemilik: PRD §8 #1, #2, #3, #5, #7, #8.

## B8 — Rilis 2 backend

- [ ] **B8.1 Perpanjangan.** API.md §12.5, job `void_extensions`, `ApplyPaymentStatus` untuk `extension`.
- [ ] **B8.2 PDF invoice.** API.md §12.4, SPEC §13.7.
- [ ] **B8.3 Deposit.** API-OWNER.md §5.5.
- [ ] **B8.4 Keuangan & statistik owner.** API-OWNER.md §8–9 (CSV streaming).
- [ ] **B8.5 Keuangan admin & rekonsiliasi.** API-ADMIN.md §10, job `reconcile_payments`.
- [ ] **B8.6 Audit log.** API-OWNER.md §11, API-ADMIN.md §12.

## F6 — Rilis 2 UI

- [ ] **F6.1 Tenant.** Perpanjang sewa (DurationPicker + PaymentPanel), unduh PDF di InvoiceCard.
- [ ] **F6.2 Owner.** StatCard di beranda, `/owner/keuangan` (ringkasan, ledger MoneyRow, CSV), catat deposit, audit log.
- [ ] **F6.3 Admin.** Laporan keuangan, ledger, rekonsiliasi, audit log.

## Q2 — Gerbang Rilis 2

- [ ] **Q2.1** Contract test & E2E fitur Rilis 2; uji perpanjangan bulanan & harian end-to-end dengan jam yang dimajukan.

## B9 — Rilis 3 backend

- [ ] **B9.1 Ulasan.** API.md §13.1–13.2, API-OWNER.md §10, API-ADMIN.md §11, cache rating, `can_review`.
- [ ] **B9.2 Favorit.** API.md §13.3.
- [ ] **B9.3 Email produksi.** ⛔ menunggu pemilik (PRD §8 #4): pengirim SMTP nyata.

## F7 — Rilis 3 UI

- [ ] **F7.1 Ulasan.** RatingStars input, ReviewCard, `/kost/{id}/ulasan`, `/owner/ulasan`, moderasi admin.
- [ ] **F7.2 Favorit.** FavoriteButton di detail, `/favorit`.

## Q3 — Gerbang Rilis 3

- [ ] **Q3.1** Contract test & E2E ulasan dan favorit; uji email lewat SMTP sandbox setelah B9.3.
