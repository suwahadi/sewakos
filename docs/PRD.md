# PRD — Platform Sewa Kost

> Dokumen kebutuhan produk: **apa** yang dibangun dan **mengapa**. Detail teknis ada di dokumen kontrak API,
> `SPEC-GO.md`, dan `DESIGN.md`; urutan kerja di `IMPLEMENTATION.md`; proses kerja di `AGENT.md`.
> Bila PRD bertentangan dengan dokumen teknis, dokumen teknis yang berlaku dan PRD diperbarui.

## 1. Ringkasan

Aplikasi web *mobile-first* untuk mencari, memesan, dan membayar sewa kost secara online, dengan tiga pengguna:

| Peran | Kebutuhan utama |
|---|---|
| **Penyewa (tenant)** | Menemukan kamar yang cocok di dekat kota/kampus/MRT, melihat harga transparan, membayar aman, mengelola tagihan sewa |
| **Pemilik (owner)** | Menayangkan kost, menerima booking yang sudah dibayar, menerima dana, memantau keuangan dan okupansi |
| **Admin platform** | Menjaga kualitas listing, menangani refund & payout, memantau keuangan dan rekonsiliasi pembayaran |

Pembayaran lewat **Midtrans Snap**. Dana masuk ke rekening platform, lalu diteruskan ke pemilik (payout) secara manual
oleh admin. Platform memperoleh pendapatan dari **biaya admin** (nominal tetap) di setiap tagihan.

## 2. Masalah & tujuan

| Masalah | Tujuan produk |
|---|---|
| Penyewa sulit membandingkan harga total (sewa, deposit, biaya lain) | Rincian harga lengkap sebelum mengajukan sewa |
| Booking tanpa pembayaran membuat kamar "tertahan" tanpa kepastian | Kamar dikunci hanya saat penyewa benar-benar membayar (15 menit) |
| Pemilik lambat merespons | Batas respons 24 jam; lewat batas otomatis ditolak dan dana dikembalikan |
| Tagihan bulanan tercecer | Tagihan otomatis H-3, pengingat, dan akhir sewa otomatis bila telat > 3 hari |
| Keuangan sulit ditelusuri | Buku besar append-only, laporan, rekonsiliasi harian dengan Midtrans |

## 3. Indikator keberhasilan

Diukur dari data aplikasi sendiri (v1 tidak memakai analytics pihak ketiga).

| Indikator | Cara hitung | Target awal |
|---|---|---|
| Konversi bayar | booking `paid` ÷ booking dibuat | ≥ 40% |
| Keberhasilan checkout | payment `settlement` ÷ payment dibuat | ≥ 70% |
| Waktu respons pemilik | median `responded_at − paid_at` | ≤ 6 jam |
| Penolakan otomatis | booking `rejected_by = system` ÷ booking `paid` | ≤ 10% |
| Tagihan tepat waktu | invoice `renewal` dibayar ≤ `due_date` ÷ semua renewal | ≥ 85% |
| Selisih rekonsiliasi | issue `open` > 48 jam | 0 |

## 4. Lingkup rilis

### Rilis 1 — Transaksi inti (MVP)

- Akun: daftar, OTP (SMS/WhatsApp), masuk dengan password/OTP, keluar, lupa & ganti password, profil, ganti nomor.
- Penemuan: area populer, autocomplete, pencarian + filter (area/kampus/MRT, gender, harga, tipe sewa, promo,
  ketersediaan), urutan, detail kost dengan foto, fasilitas, rincian harga.
- Booking: ajukan sewa dengan durasi, checkout Snap (kunci kamar 15 menit), batal, rekening refund, riwayat.
- Pemilik: properti (dengan moderasi admin), kamar, foto (otomatis WebP), booking masuk (setujui/tolak),
  rekening, daftar payout.
- Admin: area, pengguna (nonaktifkan), moderasi properti, booking (baca), refund & payout manual, biaya admin,
  broadcast promo.
- Siklus sewa yang menjaga stok kamar tetap benar: invoice pembayaran pertama & tagihan bulanan (H-3), pengingat jatuh tempo,
  sewa berakhir otomatis bila telat > 3 hari, sewa selesai otomatis di `end_date`.
  (Ini wajib di Rilis 1: tanpa itu kamar sewa harian/mingguan/bulanan tidak pernah kembali tersedia.)
- Notifikasi di aplikasi untuk semua kejadian penting.
- Buku besar (ledger) & audit log **dicatat** sejak transaksi pertama (tampilan laporannya di Rilis 2).

### Rilis 2 — Masa sewa & keuangan

- Perpanjangan sewa oleh penyewa; pemilik mencatat pengembalian deposit.
- PDF invoice/kuitansi.
- Keuangan pemilik (ringkasan, ledger, CSV), laporan & rekonsiliasi admin, statistik pemilik, tampilan audit log.

### Rilis 3 — Keterlibatan

- Ulasan & rating (moderasi admin), favorit, notifikasi email.

### Di luar lingkup v1

Notifikasi push (FCM), chat, peta interaktif, mode gelap, refund/payout otomatis lewat API, potongan deposit,
berhenti sewa lebih awal, aplikasi native, multi-bahasa, analytics pihak ketiga.

## 5. Kebutuhan fungsional

Setiap kebutuhan merujuk ke dokumen yang mengatur detailnya. Kriteria penerimaan = contract test di dokumen itu lolos
**dan** poin di kolom terakhir terpenuhi.

### 5.1 Akun

| ID | Kebutuhan | Rujukan | Kriteria tambahan |
|---|---|---|---|
| ACC-1 | Daftar sebagai penyewa atau pemilik dengan nomor HP | API.md §2.1 | Nomor wajib diverifikasi OTP sebelum dianggap terverifikasi |
| ACC-2 | Kirim OTP via SMS atau WhatsApp, cooldown 60 detik, maks 2/menit per nomor | API.md §2.2 | Pilihan kanal sesuai DESIGN.md §6.48 |
| ACC-3 | Masuk dengan password atau OTP | API.md §2.3–2.4 | Akun dinonaktifkan tidak bisa masuk |
| ACC-4 | Keluar (perangkat ini) | API.md §11.1 | Token lain tetap berlaku |
| ACC-5 | Lupa password via OTP | API.md §11.2 | Semua sesi dicabut |
| ACC-6 | Ubah profil, ganti password, ganti nomor | API.md §11.3–11.6 | Ganti password mencabut sesi lain |

### 5.2 Penemuan

| ID | Kebutuhan | Rujukan | Kriteria tambahan |
|---|---|---|---|
| DSC-1 | Area populer (kota, kampus, MRT) | API.md §3.1 | — |
| DSC-2 | Autocomplete lokasi/kost/tipe kamar | API.md §3.2 | Muncul setelah 3 karakter, jeda 300ms |
| DSC-3 | Pencarian dengan filter, urutan, "muat lebih banyak" | API.md §3.3 | Kampus/MRT memakai radius 3 km |
| DSC-4 | Detail kost: galeri, fasilitas, kamar, rating | API.md §4 | Hanya properti tayang (disetujui admin, aktif, pemilik tidak dinonaktifkan) |
| DSC-5 | Rincian harga sebelum login | API.md §5.1 | Diskon promo dan biaya admin terlihat jelas |

### 5.3 Booking & pembayaran

| ID | Kebutuhan | Rujukan | Kriteria tambahan |
|---|---|---|---|
| BKG-1 | Ajukan sewa dengan tipe, durasi, tanggal mulai | API.md §5.2 | Kamar belum dikunci saat submit |
| BKG-2 | Bayar via Snap, kamar dikunci 15 menit | API.md §5.5, SPEC §13 | Dua penyewa untuk kamar terakhir: hanya satu yang bisa checkout |
| BKG-3 | Status pembayaran dari webhook + cek status Midtrans | SPEC §13.3 | Status tidak pernah diambil dari callback frontend |
| BKG-4 | Pemilik setujui/tolak dalam 24 jam; lewat = tolak otomatis + refund penuh | API-OWNER.md §5 | Pengingat 2 jam sebelum batas |
| BKG-5 | Penyewa batal: sebelum bayar tanpa refund; setelah bayar refund dikurangi biaya admin | API.md §5.6 | Booking disetujui tidak bisa dibatalkan |
| BKG-6 | Refund manual: penyewa isi rekening, admin transfer & tandai selesai | API.md §5.7, API-ADMIN.md §6 | — |

### 5.4 Masa sewa

| ID | Kebutuhan | Rujukan | Rilis |
|---|---|---|---|
| TNC-1 | Invoice untuk setiap pembayaran, nomor unik, daftar & detail | API.md §12.1–12.3 | 1 |
| TNC-2 | Tagihan bulanan H-3, jatuh tempo awal periode, berakhir otomatis bila telat > 3 hari | SPEC §14 | 1 |
| TNC-3 | Sewa selesai otomatis di `end_date`, kamar kembali tersedia | SPEC §14 | 1 |
| TNC-4 | Perpanjangan oleh penyewa, berlaku setelah dibayar | API.md §12.5 | 2 |
| TNC-5 | PDF invoice/kuitansi | API.md §12.4 | 2 |
| TNC-6 | Pemilik mencatat deposit sudah dikembalikan | API-OWNER.md §5.5 | 2 |

### 5.5 Pemilik

| ID | Kebutuhan | Rujukan |
|---|---|---|
| OWN-1 | Kelola properti; properti baru menunggu persetujuan admin | API-OWNER.md §2 |
| OWN-2 | Kelola kamar, harga, stok, pilihan durasi bulanan | API-OWNER.md §3 |
| OWN-3 | Unggah foto (presigned URL), otomatis WebP 1600px + thumbnail 400px | API-OWNER.md §4, SPEC §18 |
| OWN-4 | Rekening & daftar payout | API-OWNER.md §6 |
| OWN-5 | Keuangan: ringkasan, ledger, CSV (Rilis 2) | API-OWNER.md §8 |
| OWN-6 | Statistik & audit log (Rilis 2), ulasan (Rilis 3) | API-OWNER.md §9–11 |

### 5.6 Admin

| ID | Kebutuhan | Rujukan |
|---|---|---|
| ADM-1 | Area, pengguna, moderasi properti | API-ADMIN.md §2–4 |
| ADM-2 | Booking (baca), refund & payout manual | API-ADMIN.md §5–7 |
| ADM-3 | Biaya admin, broadcast promo | API-ADMIN.md §8–9 |
| ADM-4 | Laporan keuangan, ledger, rekonsiliasi Midtrans (Rilis 2) | API-ADMIN.md §10 |
| ADM-5 | Moderasi ulasan (Rilis 3), invoice & audit log | API-ADMIN.md §11–12 |

### 5.7 Notifikasi

| ID | Kebutuhan | Rujukan |
|---|---|---|
| NTF-1 | Notifikasi di aplikasi untuk semua kejadian di tabel teks notifikasi, dengan tautan ke halaman terkait | API.md §6, SPEC §17 |
| NTF-2 | Email untuk kejadian yang sama bila pengguna mengisi email (Rilis 3) | SPEC §17 |

## 6. Aturan bisnis utama

| Aturan | Nilai |
|---|---|
| Batas bayar setelah submit | 24 jam |
| Kunci kamar & masa berlaku transaksi Snap | 15 menit |
| Batas respons pemilik setelah dibayar | 24 jam (pengingat 2 jam sebelumnya) |
| Tagihan bulanan | Dibuat H-3, jatuh tempo awal periode, berakhir otomatis setelah telat 3 hari |
| Durasi | Bulanan: pilihan pemilik per kamar; harian 1–30; mingguan 1–12 (dibayar penuh di depan) |
| Deposit | 1 × harga periode, dipegang pemilik, dikembalikan penuh di akhir sewa |
| Diskon promo | Hanya pembayaran pertama |
| Biaya admin | Nominal tetap global (default Rp0), dikenakan di setiap tagihan |
| Refund | Pemilik tolak/diam/kamar habis: penuh. Penyewa batal setelah bayar: dikurangi biaya admin |
| Payout | Pembayaran pertama: saat disetujui. Tagihan lain: saat dibayar. Nilai = total − biaya admin |

## 7. Kebutuhan non-fungsional

| Area | Kebutuhan |
|---|---|
| Performa API | p95 < 300 ms untuk endpoint baca dan < 800 ms untuk checkout (tanpa waktu Midtrans) pada 10.000 properti |
| Performa web | LCP ≤ 2,5 s di 4G lambat; JS first-load ≤ 150 KB gzip per route; CLS ≤ 0,1 |
| Ketersediaan | Worker aman dijalankan lebih dari satu instance; job idempoten |
| Keamanan | Token di cookie httpOnly; verifikasi signature + cek status Midtrans; rate limit; tidak ada rahasia di repo |
| Integritas uang | Semua uang `int64` rupiah; ledger & audit log append-only di level database |
| Privasi | Metadata/GPS foto dibuang; nomor HP penyewa ke pemilik hanya setelah disetujui; nama di ulasan disamarkan |
| Aksesibilitas | WCAG 2.2 AA (checklist DESIGN.md §10) |
| Perangkat | Layar 320–1440px; Chrome, Safari, Firefox, Samsung Internet dua versi terakhir |
| Bahasa | Antarmuka bahasa Indonesia; zona waktu tampilan WIB |
| Kualitas kode | Aturan `AGENT.md` (lint, test, gaya komentar, kebijakan git) |

## 8. Hal terbuka (perlu keputusan pemilik)

| # | Hal | Dampak bila belum diputuskan | Dibutuhkan sebelum |
|---|---|---|---|
| 1 | Penyedia **SMS & WhatsApp** untuk OTP | OTP hanya tertulis di log; pengguna nyata tidak bisa daftar/masuk | Rilis 1 publik |
| 2 | Penyedia **object storage** produksi (S3-compatible) dan CDN | Foto tidak bisa diunggah di produksi | Rilis 1 publik |
| 3 | Kunci **Midtrans** produksi & URL webhook terdaftar | Pembayaran hanya sandbox | Rilis 1 publik |
| 4 | Penyedia **SMTP** | Email hanya tertulis di log | Rilis 3 |
| 5 | `INSTALL_SH_URL` dan isi/target `deploy/install.sh` (OS server, domain, TLS) | Blok deploy tidak bisa ditampilkan | Deploy pertama |
| 6 | Konfirmasi monorepo (`backend/` + `frontend/` satu repository) | Struktur di AGENT.md §2 dipakai apa adanya | Tugas pertama |
| 7 | Logo aplikasi (`frontend/public/logo.svg`) dan logo WhatsApp resmi | Dipakai teks sementara | UI Rilis 1 |
| 8 | Teks Kebijakan Privasi & Syarat Ketentuan | Checkbox registrasi menaut ke halaman kosong | Rilis 1 publik |

## 9. Glosarium

| Istilah | Arti |
|---|---|
| Properti tayang | `is_active`, disetujui admin, pemilik tidak dinonaktifkan |
| Kunci kamar (*hold*) | Pengurangan `available` sementara selama checkout/menunggu persetujuan |
| Invoice `initial` / `renewal` / `extension` | Tagihan pertama / bulanan berikutnya / perpanjangan |
| Payout | Transfer hak pemilik dari rekening platform ke rekening pemilik |
| Ledger | Buku besar keuangan yang hanya bisa ditambah |
| Rekonsiliasi | Pencocokan status & nominal pembayaran lokal dengan Midtrans |
