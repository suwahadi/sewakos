# SPEC — Kos API (Go)

> Dokumen ini untuk **Claude Code / AI coding agent**. Letakkan di `docs/` bersama dokumen kontrak.
> Proses kerja (git, gaya kode, pemeriksaan) diatur `AGENT.md`; urutan kerja di `IMPLEMENTATION.md`.
> Path kode di dokumen ini relatif terhadap folder `backend/`.
> Ini adalah **proyek baru dari nol**. Kontrak API terdiri dari tiga dokumen yang tidak boleh diubah:
> `API.md` (publik & tenant), `API-OWNER.md` (pemilik kost), `API-ADMIN.md` (admin);
> dokumen ini melengkapi semua hal yang tidak tercatat di sana. Semua keputusan di sini sudah dikunci.

## 0. Aturan untuk agent

1. Jangan mengubah URL, method, nama field, tipe data, status code, atau teks pesan yang ada di dokumen kontrak.
2. Kalau dokumen kontrak dan dokumen ini bertentangan, **dokumen kontrak menang**, kecuali disebut lain di §2.
3. Jangan menambah endpoint atau field respons yang tidak disebut di dokumen kontrak atau dokumen ini.
4. Sebuah endpoint dianggap selesai hanya jika contract test-nya (§11) lolos.
5. Keputusan teknis kecil (nama paket, struktur internal) dicatat di `docs/DECISIONS.md`. Keputusan yang mengubah
   kontrak, skema yang sudah di-push, atau perilaku uang → berhenti dan tanyakan (AGENT.md §3).
6. Komentar di contoh SQL/kode dalam dokumen ini adalah penjelasan; di kode asli, ikuti aturan komentar AGENT.md §4
   (bahasa Inggris, hanya untuk hal yang tidak terlihat dari kode).

## 1. Definisi "sesuai kontrak"

Respons harus **semantik JSON identik** dengan contoh di dokumen kontrak dan dokumen ini:

- Urutan key dan escaping karakter **boleh berbeda**.
- Yang **wajib sama**: nama key, tipe data (angka/string/bool/null/array/object), nilai string persis,
  status code, dan kehadiran key.
- `null`, `[]`, `{}`, dan key yang tidak ada adalah **empat hal berbeda**.
  - Slice kosong wajib diserialisasi sebagai `[]`, jangan `null`. Inisialisasi semua slice.
  - Field yang di kontrak bernilai `null` wajib tetap dikirim (jangan pakai `omitempty`).

## 2. Keputusan yang sudah dikunci

| Topik | Keputusan |
|---|---|
| Bahasa | Go 1.27.3 (`go 1.27.0` + `toolchain go1.27.3` di go.mod) |
| Database | PostgreSQL, skema dibuat lewat migration (§4) |
| Cache / rate limit / OTP store | Redis |
| `{property}` di `/listings/{property}` | ID properti (integer), sesuai API.md §9. Slug → 404 |
| `limit` di luar batas | Clamp: > 50 jadi 50; < 1 jadi default endpoint |
| Rate limit `api` | 60/menit per IP |
| Rate limit `otp` | 2/menit per nomor tujuan; `otp/request` dan `otp/check` berbagi kuota |
| Error yang tidak tercatat (token invalid, 404, 405, 500) | Envelope `status`/`meta` (§5.3) |
| Bahasa pesan | Semua `meta.message` berbahasa Indonesia (kecuali `OK` untuk sukses tanpa teks khusus). **Hanya** pesan validasi per field di `messages{}` yang berbahasa Inggris (§5.4) |
| `severity` | 2xx `OK`, 4xx `WARNING`, 5xx `ERROR` |
| OTP `for=login`, nomor belum terverifikasi | 422 |
| Login password, nomor belum terverifikasi | Boleh, `is_verified: false` |
| Minta OTP saat cooldown 60 detik | 422 `Silakan tunggu sebelum meminta kode baru.` |
| Pengiriman OTP | Hanya log (sender interface); provider SMS/WA menyusul |
| Submit booking | **Tidak** mengunci kamar; hanya role `tenant` |
| Alur booking | `pending` (bayar) → `paid` (tunggu owner) → `approved` / `rejected`; `cancelled` |
| Pembayaran | Midtrans **Snap**; yang dibayar = `price + deposit + admin_fee` |
| Kunci stok | Saat checkout (`available - 1`), berlaku 15 menit = masa berlaku transaksi Snap |
| Stok saat `paid` | Tetap terkunci; dilepas (`available + 1`) jika `rejected` atau `cancelled` |
| Batas respons owner | 24 jam setelah `paid`; lewat → otomatis `rejected` + refund penuh |
| Pengingat owner | 2 jam sebelum batas respons |
| Refund | Manual oleh admin. Owner tolak/timeout → penuh; tenant batal setelah bayar → dikurangi `admin_fee` |
| Dana | Masuk rekening platform; payout ke owner dicatat admin, dibuat saat `approved`; `amount = price + deposit` |
| `admin_fee` | Nominal tetap global, diatur admin (default `0`); disalin ke booking saat submit |
| Rekening owner | Wajib sebelum payout pertama diproses |
| Moderasi properti | Properti baru wajib disetujui admin; perubahan setelah disetujui tidak ditinjau ulang |
| Foto | Object storage S3-compatible, upload via presigned URL; tampil ke tenant (`photos`) |
| Admin | Penuh (area, user, moderasi, booking baca, refund, payout, setting, broadcast); akun hanya via CLI/seed |
| Notifikasi properti | Kategori `booking` |
| Stok oleh owner | Owner mengisi `available` (absolut); sistem tetap mengunci/melepas di atasnya |
| Foto | Diproses **sinkron** saat didaftarkan: WebP 1600px + thumbnail 400px, metadata dibuang |
| File yatim | Antrean `storage_deletions` (diisi trigger DB), worker retry, rekonsiliasi bucket mingguan |
| Ganti foto | Hapus + tambah (tidak ada endpoint ganti) |
| Keuangan | Buku besar append-only; owner: ringkasan, ledger, CSV; admin: laporan + rekonsiliasi Midtrans harian |
| Audit log | Semua aksi owner & admin, append-only |
| Masa sewa | Tenant memilih `duration` saat submit; `end_date` dihitung |
| Durasi bulanan | Pilihan per kamar (`monthly_duration_options`, default `[1]`) |
| Durasi harian/mingguan | 1–30 hari / 1–12 minggu, dibayar penuh di depan |
| Tagihan bulanan | Dibuat H-3, jatuh tempo awal periode; telat > 3 hari → `terminated` |
| Harga tagihan berikutnya | Harga normal saat booking (`period_price`), tanpa diskon; `admin_fee` sesuai setting saat tagihan dibuat |
| Perpanjangan | Oleh tenant sebelum `end_date`, berlaku setelah dibayar, tanpa persetujuan owner |
| Berhenti lebih awal | Tidak ada |
| Deposit | Dipegang owner (ikut payout), tanpa potongan; owner mencatat "sudah dikembalikan" |
| Payout tagihan berikutnya | Dibuat saat tagihan dibayar; `amount = total − admin_fee` |
| Logout | Perangkat ini saja; ganti/reset password mencabut token lain |
| Lupa password | OTP `for = reset_password` + endpoint reset |
| Ganti nomor | Boleh, OTP ke nomor baru |
| Email | Ke email apa pun yang diisi (tanpa verifikasi); isi ringkas + tautan; dikirim async lewat outbox |
| Filter lokasi | `area_id`: city = `properties.area_id`; campus/mrt = radius 3 km |
| Ulasan | Rating 1–5 + komentar, sekali per booking; owner tidak membalas; admin bisa menyembunyikan |
| Favorit | Role tenant |
| Invoice | PDF dibuat server |
| Fase berikutnya | Notifikasi push (FCM), chat |
| Diskon flash sale saat booking | Otomatis jika properti flash sale; `price` disimpan setelah diskon |
| Booking `pending` lewat `expires_at` | Otomatis `cancelled` oleh worker, kecuali sedang ada transaksi Snap aktif |
| `remaining_payment` | `price + deposit + admin_fee` |
| `next-offset` saat `has-more = false` | `null` |
| `GET /booking/my` | Pagination `page`/`limit` |
| `property_type` | Semua properti dianggap `kost`; `apartemen` → hasil kosong |
| Properti non-aktif | Tidak muncul di search; detail/view/gallery → 404 |
| Draft-price | Satu item per harga yang terisi (bulanan, mingguan, harian) |
| "Hari ini" untuk `start_date` | WIB (`Asia/Jakarta`); semua timestamp output tetap UTC |
| Token | Tidak kedaluwarsa; tidak ada endpoint logout (tidak ada di kontrak) |
| Data seed | Ada, sesuai contoh API.md (§10); hanya untuk dev/test |
| Halaman uji di `/` | **Tidak dibuat**. Hanya API |

**Di luar cakupan:** pengembalian deposit di akhir masa sewa, perpanjangan sewa, check-out penghuni,
chat, refund/payout otomatis lewat API Midtrans atau bank, dan pengaturan A/B lewat admin.

**Properti tayang** (dipakai di search, suggestion, detail, view, gallery, submit, checkout) =
`is_active = true` **dan** `moderation_status = 'approved'` **dan** pemilik `suspended_at IS NULL`.

## 3. Stack dan struktur

| Komponen | Pilihan |
|---|---|
| Router | `go-chi/chi/v5` + `net/http` |
| DB | `jackc/pgx/v5` + `sqlc` |
| Migration | `pressly/goose/v3` (file SQL) |
| Validasi | `go-playground/validator/v10` + translator pesan sendiri (§5.4) |
| Password | `golang.org/x/crypto/bcrypt`, cost 12 |
| Redis | `redis/go-redis/v9` |
| Object storage | `minio/minio-go/v7` (S3-compatible) |
| Pengolahan gambar | `davidbyttow/govips/v2` (libvips via cgo) |
| PDF | Library PDF pure-Go yang masih aktif dipelihara (mis. `johnfercher/maroto/v2`); pilih saat implementasi |
| Email | SMTP lewat `wneessen/go-mail` di balik interface `email.Sender` (+ `LogSender` untuk dev) |
| Midtrans | Klien HTTP sendiri di `internal/payment/midtrans` di balik interface (mudah di-mock) |
| Logging | `log/slog` (JSON di production) |
| Test | `testing` + `testify`, Postgres, Redis & MinIO via Docker Compose |
| Lint | `gofmt`, `go vet`, `staticcheck` (dipin sebagai tool di `go.mod`, AGENT.md §5.1) |

Pin versi library secara eksak di `go.mod`. Pengiriman OTP **sinkron** di dalam request.
**Docker:** karena cgo + libvips, build stage memakai image Debian dengan `libvips-dev`, runtime memakai Debian slim
dengan paket `libvips` (bukan `scratch`/distroless static). `CGO_ENABLED=1`.

```
cmd/api/main.go
cmd/worker/main.go
cmd/seed/main.go
cmd/admin/main.go    (buat akun admin: --name --phone --password)
internal/
  config/
  server/            router.go, middleware.go
  response/          envelope, severity, helper error
  validation/        translator pesan
  auth/              handler, service, token
  otp/               service, sender (interface + LogSender)
  area/  search/  listing/  booking/  notification/  abtest/
  payment/           checkout, webhook, midtrans client
  owner/             properti, kamar, foto, booking masuk, rekening, payout
  admin/             area, user, moderasi, refund, payout, setting, broadcast
  jobs/              worker terjadwal
  invoice/           tagihan, PDF
  media/             pengolahan gambar, antrean hapus file
  finance/           ledger, ringkasan, CSV, rekonsiliasi
  audit/             audit log
  account/           profil, password, nomor HP, logout
  review/  favorite/  stats/
  email/             sender, outbox, template
  platform/          postgres, redis, storage
  money/             format Rupiah
db/migrations/*.sql
db/queries/*.sql     (sqlc)
tests/contract/
Dockerfile           (multi-stage, image yang sama untuk api & worker)
.env.example
```

Di root repository (bukan di `backend/`): `docker-compose.yml` untuk dev berisi Postgres, Redis, MinIO
(IMPLEMENTATION.md R0.3), dan `docs/DECISIONS.md`. Di dev, api & worker dijalankan dengan `go run`;
di produksi memakai image dari `Dockerfile` lewat `deploy/install.sh`.

### Konfigurasi (env)

| Variabel | Default | Keterangan |
|---|---|---|
| `APP_ENV` | `development` | `development` / `production` |
| `HTTP_ADDR` | `:8121` | Sesuai base URL di API.md |
| `DATABASE_URL` | — | Wajib |
| `REDIS_URL` | — | Wajib |
| `API_CLIENT_ID` | `KOSAPI WEB:WEB` | Nilai header `x-api-client` |
| `CORS_ALLOWED_ORIGINS` | kosong | Daftar dipisah koma; kosong = tidak ada CORS |
| `OTP_LENGTH` | `4` | |
| `OTP_TTL_SECONDS` | `300` | |
| `OTP_RESEND_AFTER_SECONDS` | `60` | |
| `OTP_MAX_PER_MINUTE` | `2` | |
| `OTP_DEBUG_ECHO` | `false` | `true` hanya di dev. **Server menolak start jika `true` dan `APP_ENV=production`** |
| `MIDTRANS_SERVER_KEY` | — | Wajib, rahasia. Dipakai untuk memanggil API Midtrans (Snap, GET status, expire) |
| `MIDTRANS_SIGNATURE_KEY` | = `MIDTRANS_SERVER_KEY` | Rahasia. "Signature Secret Key" dari dashboard Midtrans, dipakai **hanya** untuk memverifikasi `signature_key` webhook (§13.3). Kosong = pakai server key |
| `MIDTRANS_CLIENT_KEY` | — | Untuk frontend (Snap.js) |
| `MIDTRANS_IS_PRODUCTION` | `false` | Pilih base URL sandbox/production. Satu-satunya penentu environment; jangan menyimpulkan environment dari prefix kunci (`SB-`, `Mid-`) |
| `MIDTRANS_FINISH_URL` | kosong | `callbacks.finish` Snap (halaman frontend setelah bayar) |
| `MIDTRANS_BASE_URL` | kosong | Mengganti base URL Snap & API Midtrans (Midtrans tiruan untuk E2E). **Server menolak start jika diisi dan `APP_ENV=production`** |
| `PAYMENT_EXPIRY_MINUTES` | `15` | Kunci stok = masa berlaku transaksi |
| `OWNER_RESPONSE_HOURS` | `24` | Batas respons owner |
| `OWNER_REMINDER_BEFORE_MINUTES` | `120` | Pengingat sebelum batas |
| `STORAGE_ENDPOINT` | — | Host S3-compatible |
| `STORAGE_REGION` | `us-east-1` | |
| `STORAGE_BUCKET` | — | |
| `STORAGE_ACCESS_KEY` / `STORAGE_SECRET_KEY` | — | Rahasia |
| `STORAGE_USE_SSL` | `true` | |
| `STORAGE_PUBLIC_BASE_URL` | — | Prefix URL publik foto (CDN/bucket publik) |
| `UPLOAD_MAX_BYTES` | `10485760` | 10 MB (file asli sebelum konversi) |
| `IMAGE_MAX_PIXELS` | `40000000` | Tolak gambar > 40 MP |
| `IMAGE_LARGE_PX` / `IMAGE_THUMB_PX` | `1600` / `400` | Sisi terpanjang |
| `IMAGE_LARGE_QUALITY` / `IMAGE_THUMB_QUALITY` | `80` / `75` | Kualitas WebP |
| `IMAGE_MAX_CONCURRENCY` | jumlah CPU | Batas proses gambar paralel |
| `AREA_RADIUS_KM` | `3` | Radius filter kampus/MRT |
| `RENEWAL_ISSUE_DAYS_BEFORE` | `3` | Tagihan bulanan dibuat H-n |
| `RENEWAL_GRACE_DAYS` | `3` | Telat > n hari → `terminated` |
| `SMTP_HOST` / `SMTP_PORT` / `SMTP_USERNAME` / `SMTP_PASSWORD` | — / `587` / — / — | Kosong = `LogSender`; login tanpa password juga `LogSender` (peringatan di log). Dengan login wajib STARTTLS (port 465 = TLS langsung) |
| `EMAIL_FROM` | — | Mis. `Kos API <no-reply@domain>`. Wajib bila `SMTP_HOST` diisi (worker gagal start tanpa ini) |
| `APP_PUBLIC_URL` | — | Dasar tautan di email |

## 4. Skema database (migration)

Semua timestamp `timestamptz`, semua tabel punya `created_at` dan `updated_at` (default `now()`).
Uang disimpan sebagai `bigint` rupiah utuh.

```sql
users
  id bigserial PK
  name varchar(255) NOT NULL
  role varchar(10) NOT NULL DEFAULT 'tenant' CHECK (role IN ('tenant','owner','admin'))
  phone varchar(20) NOT NULL UNIQUE
  email varchar(255) NULL UNIQUE
  password varchar(255) NULL            -- hash bcrypt
  phone_verified_at timestamptz NULL
  email_verified_at timestamptz NULL
  suspended_at timestamptz NULL

areas
  id bigserial PK
  name varchar(255) NOT NULL
  slug varchar(255) NOT NULL UNIQUE
  type varchar(10) NOT NULL CHECK (type IN ('city','campus','mrt'))
  is_top boolean NOT NULL DEFAULT false
  latitude double precision NOT NULL
  longitude double precision NOT NULL
  place_id varchar(255) NULL
  INDEX (type, is_top)

properties
  id bigserial PK
  owner_id bigint NOT NULL → users(id)
  area_id bigint NOT NULL → areas(id)
  name varchar(255) NOT NULL
  slug varchar(255) NOT NULL UNIQUE
  gender varchar(10) NOT NULL CHECK (gender IN ('campur','putra','putri'))
  description text NOT NULL DEFAULT ''
  address text NOT NULL DEFAULT ''
  latitude double precision NOT NULL
  longitude double precision NOT NULL
  is_active boolean NOT NULL DEFAULT true
  moderation_status varchar(10) NOT NULL DEFAULT 'pending'
         CHECK (moderation_status IN ('pending','approved','rejected','suspended'))
  moderation_note text NULL
  moderated_at timestamptz NULL
  is_flash_sale boolean NOT NULL DEFAULT false
  flash_sale_discount bigint NOT NULL DEFAULT 0
  views bigint NOT NULL DEFAULT 0
  INDEX (is_active, moderation_status)
  INDEX (owner_id)

rooms
  id bigserial PK
  property_id bigint NOT NULL → properties(id) ON DELETE CASCADE
  name varchar(255) NOT NULL
  price_monthly bigint NOT NULL
  price_weekly bigint NULL
  price_daily bigint NULL
  available smallint NOT NULL DEFAULT 0
  capacity smallint NOT NULL DEFAULT 1
  facilities jsonb NOT NULL DEFAULT '[]'
  monthly_duration_options jsonb NOT NULL DEFAULT '[1]'
  INDEX (property_id)

bookings
  id bigserial PK
  tenant_id bigint NOT NULL → users(id)
  room_id bigint NOT NULL → rooms(id)
  rent_type varchar(10) NOT NULL CHECK (rent_type IN ('daily','weekly','monthly'))
  start_date date NOT NULL
  status varchar(10) NOT NULL DEFAULT 'pending'
         CHECK (status IN ('pending','paid','approved','completed','terminated','rejected','cancelled'))
  duration smallint NOT NULL DEFAULT 1      -- jumlah periode (termasuk perpanjangan yang sudah dibayar)
  end_date date NOT NULL                    -- eksklusif
  period_price bigint NOT NULL              -- harga normal per periode saat booking
  flash_sale_discount bigint NOT NULL DEFAULT 0
  price bigint NOT NULL                     -- harga periode pertama setelah diskon
  deposit bigint NOT NULL
  admin_fee bigint NOT NULL DEFAULT 0       -- biaya admin tagihan pertama
  expires_at timestamptz NOT NULL          -- batas bayar
  stock_held boolean NOT NULL DEFAULT false -- true = 1 unit kamar sedang dikunci booking ini
  paid_at timestamptz NULL
  owner_deadline_at timestamptz NULL        -- paid_at + 24 jam
  reminder_sent_at timestamptz NULL
  responded_at timestamptz NULL
  reject_reason text NULL
  rejected_by varchar(10) NULL CHECK (rejected_by IN ('owner','system'))
  cancelled_by varchar(10) NULL CHECK (cancelled_by IN ('tenant','system'))
  ended_at timestamptz NULL                 -- completed/terminated
  deposit_returned_at timestamptz NULL
  deposit_return_note text NULL
  INDEX (status, expires_at)
  INDEX (status, owner_deadline_at)
  INDEX (tenant_id, created_at DESC)

user_notifications
  id bigserial PK
  user_id bigint NOT NULL → users(id) ON DELETE CASCADE
  category varchar(10) NOT NULL CHECK (category IN ('booking','payment','promo','chat'))
  title varchar(255) NOT NULL
  body text NOT NULL
  link varchar(255) NULL                    -- path frontend, mis. '/booking/2'
  read_at timestamptz NULL
  INDEX (user_id, created_at DESC)
  INDEX (user_id) WHERE read_at IS NULL

access_tokens
  id bigserial PK
  user_id bigint NOT NULL → users(id) ON DELETE CASCADE
  token_hash char(64) NOT NULL UNIQUE     -- hex SHA-256
  last_used_at timestamptz NULL
```

```sql
property_photos
  id bigserial PK
  property_id bigint NOT NULL → properties(id) ON DELETE CASCADE
  room_id bigint NULL → rooms(id) ON DELETE CASCADE
  object_key varchar(500) NOT NULL UNIQUE   -- WebP besar
  thumb_key varchar(500) NOT NULL UNIQUE    -- WebP thumbnail
  width integer NOT NULL
  height integer NOT NULL
  size_bytes integer NOT NULL               -- ukuran WebP besar
  caption varchar(255) NULL
  sort_order integer NOT NULL DEFAULT 0
  INDEX (property_id, sort_order, id)

payments
  id bigserial PK
  invoice_id bigint NOT NULL → invoices(id)
  order_id varchar(50) NOT NULL UNIQUE      -- 'KOS-{invoice_id}-{payment_id}'
  gross_amount bigint NOT NULL
  status varchar(10) NOT NULL DEFAULT 'pending'
         CHECK (status IN ('pending','settlement','expire','cancel','deny','failure'))
  snap_token varchar(100) NULL
  redirect_url text NULL
  payment_type varchar(50) NULL
  midtrans_transaction_id varchar(100) NULL
  expires_at timestamptz NOT NULL
  paid_at timestamptz NULL
  last_notification jsonb NULL
  UNIQUE INDEX (invoice_id) WHERE status = 'pending'   -- maksimal 1 transaksi aktif per invoice
  INDEX (status, expires_at)

refunds
  id bigserial PK
  booking_id bigint NOT NULL → bookings(id)
  invoice_id bigint NOT NULL UNIQUE → invoices(id)
  amount bigint NOT NULL
  reason varchar(20) NOT NULL CHECK (reason IN ('owner_rejected','owner_timeout','tenant_cancelled','stock_unavailable','invoice_void'))
  status varchar(20) NOT NULL DEFAULT 'awaiting_account' CHECK (status IN ('awaiting_account','ready','completed'))
  bank_name varchar(100) NULL
  account_number varchar(20) NULL
  account_name varchar(255) NULL
  transfer_reference varchar(100) NULL
  note text NULL
  completed_by bigint NULL → users(id)
  completed_at timestamptz NULL

payouts
  id bigserial PK
  booking_id bigint NOT NULL → bookings(id)
  invoice_id bigint NOT NULL UNIQUE → invoices(id)
  owner_id bigint NOT NULL → users(id)
  amount bigint NOT NULL
  status varchar(10) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','completed'))
  bank_name varchar(100) NULL               -- salinan saat transfer
  account_number varchar(20) NULL
  account_name varchar(255) NULL
  transfer_reference varchar(100) NULL
  note text NULL
  completed_by bigint NULL → users(id)
  completed_at timestamptz NULL
  INDEX (owner_id, created_at DESC)

owner_bank_accounts
  user_id bigint PK → users(id) ON DELETE CASCADE
  bank_name varchar(100) NOT NULL
  account_number varchar(20) NOT NULL
  account_name varchar(255) NOT NULL

settings
  key varchar(50) PK
  value jsonb NOT NULL                      -- 'admin_fee' → 0

invoices
  id bigserial PK
  booking_id bigint NOT NULL → bookings(id)
  number varchar(30) NOT NULL UNIQUE        -- 'INV/YYYY/MM/{id 6 digit}', bulan = issued_at WIB
  type varchar(10) NOT NULL CHECK (type IN ('initial','renewal','extension'))
  period_start date NOT NULL
  period_end date NOT NULL                  -- eksklusif
  periods smallint NOT NULL
  rent_amount bigint NOT NULL
  discount bigint NOT NULL DEFAULT 0
  deposit bigint NOT NULL DEFAULT 0
  admin_fee bigint NOT NULL DEFAULT 0
  total bigint NOT NULL                     -- rent_amount - discount + deposit + admin_fee
  status varchar(10) NOT NULL DEFAULT 'unpaid' CHECK (status IN ('unpaid','paid','void'))
  due_date date NOT NULL                    -- WIB
  issued_at timestamptz NOT NULL
  paid_at timestamptz NULL
  void_reason varchar(30) NULL
  reminder_sent_at timestamptz NULL
  UNIQUE INDEX (booking_id, type, period_start) WHERE type = 'renewal'
  UNIQUE INDEX (booking_id) WHERE type = 'initial'
  INDEX (status, due_date)

storage_deletions
  id bigserial PK
  object_key varchar(500) NOT NULL
  attempts smallint NOT NULL DEFAULT 0
  next_attempt_at timestamptz NOT NULL DEFAULT now()
  last_error text NULL
  INDEX (next_attempt_at)

ledger_entries                              -- APPEND-ONLY
  id bigserial PK
  occurred_at timestamptz NOT NULL DEFAULT now()
  type varchar(30) NOT NULL                 -- lihat API-ADMIN.md §10.2
  amount bigint NOT NULL                    -- bertanda (+/−); deposit_returned bernilai positif sebagai catatan
  owner_id bigint NULL → users(id)
  tenant_id bigint NULL → users(id)
  booking_id bigint NULL → bookings(id)
  invoice_id bigint NULL → invoices(id)
  payment_id bigint NULL → payments(id)
  refund_id bigint NULL → refunds(id)
  payout_id bigint NULL → payouts(id)
  description varchar(255) NOT NULL
  created_by bigint NULL → users(id)        -- NULL = sistem
  INDEX (owner_id, occurred_at DESC)
  INDEX (type, occurred_at)

audit_logs                                  -- APPEND-ONLY
  id bigserial PK
  created_at timestamptz NOT NULL DEFAULT now()
  actor_id bigint NULL → users(id)          -- NULL = sistem
  actor_role varchar(10) NULL
  action varchar(50) NOT NULL               -- 'booking.approve', 'property.update', ...
  entity_type varchar(30) NOT NULL
  entity_id bigint NOT NULL
  owner_id bigint NULL                      -- pemilik entitas, untuk filter owner
  changes jsonb NOT NULL DEFAULT '{}'
  ip inet NULL
  user_agent varchar(255) NULL
  INDEX (owner_id, created_at DESC)
  INDEX (entity_type, entity_id)

reconciliation_issues
  id bigserial PK
  payment_id bigint NOT NULL → payments(id)
  kind varchar(30) NOT NULL CHECK (kind IN ('status_mismatch','amount_mismatch','not_found_at_midtrans'))
  local_status varchar(20) NOT NULL
  midtrans_status varchar(20) NULL
  local_amount bigint NOT NULL
  midtrans_amount bigint NULL
  status varchar(10) NOT NULL DEFAULT 'open' CHECK (status IN ('open','resolved'))
  note text NULL
  detected_at timestamptz NOT NULL DEFAULT now()
  resolved_by bigint NULL → users(id)
  resolved_at timestamptz NULL
  UNIQUE INDEX (payment_id, kind) WHERE status = 'open'

reviews
  id bigserial PK
  booking_id bigint NOT NULL UNIQUE → bookings(id)
  property_id bigint NOT NULL → properties(id)
  tenant_id bigint NOT NULL → users(id)
  rating smallint NOT NULL CHECK (rating BETWEEN 1 AND 5)
  comment text NULL
  is_hidden boolean NOT NULL DEFAULT false
  hidden_reason text NULL
  INDEX (property_id, is_hidden, created_at DESC)

favorites
  user_id bigint → users(id) ON DELETE CASCADE
  property_id bigint → properties(id) ON DELETE CASCADE
  created_at timestamptz NOT NULL DEFAULT now()
  PK (user_id, property_id)

property_views_daily
  property_id bigint → properties(id) ON DELETE CASCADE
  view_date date                            -- WIB
  views integer NOT NULL DEFAULT 0
  PK (property_id, view_date)

email_outbox
  id bigserial PK
  to_email varchar(255) NOT NULL
  subject varchar(255) NOT NULL
  body_text text NOT NULL
  attempts smallint NOT NULL DEFAULT 0
  next_attempt_at timestamptz NOT NULL DEFAULT now()
  sent_at timestamptz NULL
  last_error text NULL
  INDEX (sent_at, next_attempt_at)
```

**Trigger wajib:**
- `property_photos` AFTER DELETE → insert `object_key` dan `thumb_key` ke `storage_deletions`.
  Ini menangkap juga penghapusan berantai (kamar dihapus → foto ikut terhapus).
- `ledger_entries` dan `audit_logs` BEFORE UPDATE OR DELETE → `RAISE EXCEPTION` (append-only).
  User database aplikasi juga tidak diberi hak `UPDATE`/`DELETE`/`TRUNCATE` pada kedua tabel.
- Rata-rata rating properti dihitung dengan query agregat (`WHERE NOT is_hidden`), di-cache Redis 5 menit,
  cache dihapus saat ulasan dibuat/disembunyikan.

OTP **tidak** disimpan di Postgres; hanya di Redis (§6.3).
Semua uang dihitung dengan `int64` (rupiah utuh), tidak pernah float.

## 5. Konvensi global

### 5.1 Base path dan middleware

Semua route di bawah prefix `/api`. Urutan middleware:

1. Request ID + logging
2. Recovery (panic → 500 envelope)
3. CORS (hanya jika `CORS_ALLOWED_ORIGINS` diisi). Preflight `OPTIONS` dijawab 204 sebelum cek header klien.
   Header yang diizinkan: `x-api-client`, `Content-Type`, `Authorization`. Method: `GET, POST, PUT, PATCH, DELETE, OPTIONS`.
4. **Client check**: header `x-api-client` harus persis `API_CLIENT_ID` (nama header case-insensitive).
   **Pengecualian:** `POST /api/payments/midtrans/notification` (dipanggil Midtrans) tidak melewati client check
   maupun limiter `api`; endpoint ini diamankan dengan signature (§13.3).
   Gagal → 401 `Klien tidak dikenal.`
5. Rate limiter `api` (60/menit per IP)
6. Auth Bearer, **hanya** untuk endpoint `[auth]`, `/owner/*`, dan `/admin/*`.
   User dengan `suspended_at` terisi → 403 `Akun dinonaktifkan.` (tokennya juga sudah dihapus saat suspend).
7. Role check: `/owner/*` hanya `owner`, `/admin/*` hanya `admin`; submit/checkout/cancel/refund-account booking
   hanya `tenant`; begitu juga extend, review, dan favorit. `/me/*` dan `/auth/logout` untuk semua role.
   Gagal → 403 `Akses ditolak.`
8. Handler

### 5.2 Envelope

```json
{ "status": true, "meta": { "response_code": 200, "code": 200, "severity": "OK", "message": "..." }, "...": "data" }
```

- `response_code` dan `code` selalu sama dengan HTTP status.
- `meta.message` sukses memakai teks dari API.md bila ada; selain itu `"OK"`.
- Semua respons dibuat lewat satu paket `response`; handler tidak boleh menulis JSON sendiri.

### 5.3 Error

| Kasus | Status | `meta.message` | `messages` |
|---|---|---|---|
| Validasi | 422 | `Validasi gagal.` (kecuali disebut lain) | object `{ field: [pesan, ...] }` |
| Login gagal | 401 | `Nomor handphone atau password salah.` | `null` |
| Header klien salah | 401 | `Klien tidak dikenal.` | tidak dikirim |
| Token tidak ada/invalid | 401 | `Sesi tidak valid. Silakan login kembali.` | tidak dikirim |
| Akses milik user lain | 403 | `Akses ditolak.` | tidak dikirim |
| Route/resource tidak ada | 404 | `Data tidak ditemukan.` | tidak dikirim |
| Method salah | 405 | `Metode tidak diizinkan.` | tidak dikirim |
| Rate limit | 429 | `Terlalu banyak permintaan. Silakan coba lagi nanti.` | tidak dikirim |
| Error internal | 500 | `Terjadi kesalahan pada server.` | tidak dikirim |
| Midtrans gagal | 502 | `Layanan pembayaran sedang bermasalah. Silakan coba lagi.` | tidak dikirim |
| Akun dinonaktifkan | 403 | `Akun dinonaktifkan.` | tidak dikirim |
| Error bisnis 400 | 400 | sesuai endpoint | tidak dikirim |

Respons 429 juga mengirim header `Retry-After`, `X-RateLimit-Limit`, dan `X-RateLimit-Remaining`.
Detail error internal hanya masuk log, tidak pernah ke respons.

### 5.4 Pesan validasi (bagian dari kontrak)

Pesan per field di `messages{}` **sengaja berbahasa Inggris** dan merupakan satu-satunya teks non-Indonesia di API.
`meta.message` untuk 422 tetap `Validasi gagal.` (kecuali endpoint menyebut pesan khusus).

**Nama atribut**: nama field dengan `_` diganti spasi (`phone_number` → `phone number`).

| Rule | Pesan |
|---|---|
| required | `The :attribute field is required.` |
| accepted | `The :attribute field must be accepted.` |
| string | `The :attribute field must be a string.` |
| integer | `The :attribute field must be an integer.` |
| boolean | `The :attribute field must be true or false.` |
| array | `The :attribute field must be an array.` |
| email | `The :attribute field must be a valid email address.` |
| min (string) | `The :attribute field must be at least :min characters.` |
| max (string) | `The :attribute field must not be greater than :max characters.` |
| regex | `The :attribute field format is invalid.` |
| in / exists | `The selected :attribute is invalid.` |
| unique | `The :attribute has already been taken.` |
| date | `The :attribute field must be a valid date.` |
| after_or_equal:today | `The :attribute field must be a date after or equal to today.` |
| numeric | `The :attribute field must be a number.` |
| min (angka) | `The :attribute field must be at least :min.` |
| between (angka) | `The :attribute field must be between :min and :max.` |
| max (array) | `The :attribute field must not have more than :max items.` |
| max (string item array, mis. `facilities.0`) | atribut ditulis `facilities.0`: `The facilities.0 field must not be greater than :max characters.` |

**Aturan evaluasi:**
- Rule diperiksa berurutan sesuai tabel rule per endpoint.
- Jika `required` atau `accepted` gagal, rule berikutnya untuk field itu **tidak** dicek.
- Field opsional yang tidak dikirim/kosong dilewati seluruhnya.
- Selain itu, **semua** rule yang gagal dilaporkan (satu field bisa punya beberapa pesan).
- Body JSON yang rusak diperlakukan seperti body kosong.

### 5.5 Format nilai

| Jenis | Format | Contoh |
|---|---|---|
| Rupiah | `"Rp"` + ribuan dengan titik, tanpa spasi | `Rp2.250.000`, `Rp0` |
| `expires_at` booking | `2006-01-02T15:04:05-07:00` dalam UTC | `2026-11-10T17:36:44+00:00` |
| `created_at`, `read_at` | `2006-01-02T15:04:05.000000Z` dalam UTC | `2026-11-10T17:36:44.000000Z` |
| `start_date` | `2006-01-02` | `2026-12-01` |
| Latitude/longitude | float JSON | `-6.3878486` |

Jangan pakai `time.RFC3339` untuk `expires_at`, karena hasilnya `Z`, bukan `+00:00`.

## 6. Auth dan OTP

### 6.1 Token

**Membuat:**
1. `random` = 40 karakter acak `[A-Za-z0-9]` dari `crypto/rand`.
2. `plain` = `random` + hex CRC32 IEEE (8 karakter huruf kecil) dari `random`. Panjang total 48, sesuai contoh API.md.
3. Insert `access_tokens(user_id, token_hash = hex SHA-256 dari plain)`.
4. Kembalikan `"{id}|{plain}"`, contoh `2|VjGS6DZX8K096pOaQG3XKq7BkVLtZV8eqj1pNXpn5c3c9053`.

**Memverifikasi:** pisah di `|` pertama, ambil baris berdasarkan `id`, bandingkan SHA-256 bagian kedua
dengan `token_hash` secara constant-time. Format salah atau tidak cocok → 401 `Sesi tidak valid. Silakan login kembali.`
Sukses → update `last_used_at`. Token tidak kedaluwarsa.

### 6.2 `POST /auth/register`

| Field | Rule (berurutan) |
|---|---|
| name | required, string, max:255 |
| phone_number | required, string, regex `^08\d{7,12}$`, unique (`users.phone`) |
| email | opsional, email, unique (`users.email`) |
| password | required, string, min:8 |
| role | opsional, in:tenant,owner (default `tenant`) |
| is_agree_privacy_policy | accepted |

`phone_number` disimpan di kolom `users.phone`. Respons persis API.md §2.1.

### 6.3 `POST /auth/otp/request` — limiter otp

| Field | Rule |
|---|---|
| for | required, in:tenant_verification,login,reset_password |
| via | required, in:0,1 (`"0"`=SMS, `"1"`=WhatsApp) |
| destination | required, regex `^08\d{7,12}$` |

Urutan proses:
1. Validasi (422).
2. `tenant_verification`: user tidak ada **atau** sudah terverifikasi → 404 `Data tidak ditemukan.`.
3. `reset_password`: user tidak ada → 404; dinonaktifkan → 403. Status verifikasi tidak dicek.
4. `login`: user tidak ada → 404; user belum terverifikasi → 422,
   `meta.message` dan `messages.destination[0]` = `Nomor handphone belum terverifikasi.`
5. Cooldown aktif → 422, `meta.message` dan `messages.destination[0]` = `Silakan tunggu sebelum meminta kode baru.`
6. Buat kode 4 digit (`crypto/rand`, boleh diawali 0). Simpan di Redis:
   `otp:code:{for}:{destination}` TTL 300 detik, `otp:cooldown:{destination}` TTL 60 detik.
7. Kirim lewat `otp.Sender` (sinkron). Saat ini hanya `LogSender` yang menulis ke log:
   `channel`, `destination`, dan pesan berisi kode. Gagal kirim → 500.

```go
type Sender interface {
    Send(ctx context.Context, channel Channel, destination, message string) error
}
```

Respons sukses: pesan `Kode verifikasi telah dikirim.`,
`verification_code{can_resend_at (unix detik = now+60), can_resend_after_seconds: 60}`, `data: null`,
dan `debug_code` **hanya** jika `OTP_DEBUG_ECHO=true` (jika false, key tidak dikirim).

### 6.4 `POST /auth/otp/check` — limiter otp (kuota sama dengan request)

Rule: for (required, in), destination (required, regex), code (required, string).

- Kode cocok → hapus key kode dari Redis.
  - `tenant_verification`: isi `phone_verified_at = now`.
  - Terbitkan token. Respons: pesan `Nomor handphone terverifikasi.`, `verified: true`, `token`, `user{id,name,role}`.
  - `for=login` memakai bentuk dan pesan respons yang sama.
- Salah/expired → 400 `Kode verifikasi salah atau kedaluwarsa.`
- `for=reset_password` **tidak bisa** diverifikasi lewat `otp/check` (→ 400 pesan yang sama); kodenya hanya
  dipakai oleh `POST /auth/password/reset`.
- Ganti nomor (`/me/phone/*`) memakai key Redis `otp:code:change_phone:{user_id}:{nomor_baru}`,
  cooldown dan limiter per nomor baru.

### 6.5 `POST /auth/login`

Rule: phone_number (required), password (required).
User tidak ada, password NULL, atau hash tidak cocok → 401 (§5.3).
Sukses: pesan `Login berhasil.`, `token`, `user{id,name,role,is_verified}` dengan `is_verified = phone_verified_at != null`.
User yang belum terverifikasi tetap boleh login.
User `suspended_at` terisi → 403 `Akun dinonaktifkan.` (dicek setelah password cocok). Sama untuk OTP login
(`otp/request` dan `otp/check` dengan `for=login`).

## 7. Pencarian dan listing

### 7.1 `GET /area/{type}?top=true`

`type` di luar `city|campus|mrt` → 404. `top=true` → hanya `is_top = true`.
Urut `id ASC`. Item: `id, name, slug, latitude, longitude`.

### 7.2 `GET /suggestion?keyword=`

- keyword: required, string, min:3.
- `areas`: `areas.name ILIKE %keyword%`, item `{title, area, latitude, longitude, geocode_id (= areas.id), type: "area"}`.
- `area_source`: selalu `"db"`.
- `landings`: properti tayang, nama ILIKE keyword, item `{id, title, slug, gender}`.
- `room_types`: kamar dari properti tayang, nama ILIKE keyword, item `{id, property_id, title, slug (slug properti)}`.
- Maksimal 5 item per kelompok, urut `id ASC`. Kelompok kosong tetap `[]`.

### 7.3 `POST /listings/search`

| Input | Makna |
|---|---|
| `filters.gender` | array int: 0=campur, 1=putra, 2=putri |
| `filters.rent_type` | 1=harian (`price_daily`), 2=mingguan (`price_weekly`), 3=bulanan (`price_monthly`). Default 3 |
| `filters.price_range` | `[min, max]` inklusif terhadap kolom harga aktif |
| `filters.property_type` | `kost` → tanpa filter; `apartemen` → hasil kosong |
| `filters.flash_sale` | `true` → hanya `is_flash_sale = true`; `false` → tanpa filter |
| `filters.is_available` | `true` → `available >= 1`; `false` → tanpa filter |
| `sorting.field` | `price` (kolom harga aktif) / `availability` (`available`) |
| `sorting.direction` | `-` = DESC, `+` = ASC. Tanpa sorting → `rooms.id ASC`. Selalu tambah `rooms.id ASC` sebagai tie-breaker |
| `limit` | default 20, clamp 1..50 |
| `offset` | default 0, minimal 0 |
| `filters.area_id` | opsional, exists (`areas.id`). `city` → `properties.area_id = X`; `campus`/`mrt` → jarak haversine ≤ `AREA_RADIUS_KM` dari titik area (prafilter bounding box lat/lng agar memakai index) |

Selalu: hanya properti tayang (§2), dan kamar dengan harga aktif NULL tidak ikut.

Respons:
- `source: "db"`
- `property_total`: jumlah properti distinct yang cocok
- `total`: jumlah kamar yang cocok
- `limit`: limit setelah clamp
- `has-more`: `offset + jumlah_item < total`
- `next-offset`: `offset + limit` jika has-more, selain itu `null`
- `rooms[]`: `{id, property_id, name, property_name, area_name, thumbnail_url, gender, price, price_label, available, is_flash_sale, rating_average, rating_count}`
  (`thumbnail_url` = thumbnail foto properti pertama tanpa `room_id`, urut `sort_order, id`; diambil dengan satu
  subquery `LATERAL`/`DISTINCT ON`, bukan query per baris)

### 7.4 `GET /listings/{id}`

`{id}` bukan integer positif, tidak ditemukan, atau properti tidak tayang (§2) → 404.
Respons persis API.md §4.1. `facilities` = array dari jsonb.
`photos` = `{id, url, caption}`; `url = STORAGE_PUBLIC_BASE_URL + "/" + object_key`.
Foto dengan `room_id NULL` → `property.photos`; selain itu → `rooms[].photos`. Urut `sort_order, id`.
`is_favorite`: middleware auth **opsional** di endpoint ini — token valid role `tenant` → cek `favorites`;
tanpa token/token invalid/role lain → `false`, tanpa error.

### 7.5 `POST /listings/{id}/view`

`UPDATE properties p SET views = views + 1 WHERE id = $1 AND <properti tayang> RETURNING views` → `{ "views": n }`.
Tidak ada baris → 404.

### 7.6 `GET /listings/{id}/gallery`

Tidak ditemukan atau tidak tayang → 404. `gallery[]`: `{room_id, name, facilities, photos}`, urut `rooms.id ASC`.

## 8. Booking

### 8.1 `GET /booking/draft-price/{room}?is_flash_sale=` (PUBLIK)

- Kamar tidak ditemukan → 404. Status aktif properti tidak diperiksa di sini.
- `diskon = property.is_flash_sale && query is_flash_sale == "true" ? property.flash_sale_discount : 0`
- Satu item `data[]` per harga yang tidak NULL:

| sort | label | type | kolom |
|---|---|---|---|
| 1 | Per Bulan | monthly | price_monthly |
| 2 | Per Minggu | weekly | price_weekly |
| 3 | Per Hari | daily | price_daily |

Per item:
- `deposit = price`, `admin_fee = settings.admin_fee`, `flash_sale_discount = diskon`
- `total_off_first_payment = price + deposit + admin_fee - diskon`
- `remaining_payment = price + deposit + admin_fee`
- Setiap angka punya pasangan `*_label` berformat Rupiah.

Cek dengan contoh API.md: 2.250.000 + 2.250.000 + 0 − 250.000 = **4.250.000**; remaining = **4.500.000**.

### 8.2 `POST /booking/submit` `[auth]`

| Field | Rule |
|---|---|
| room_id | required, integer, exists (`rooms.id`) |
| rent_type | required, in:daily,weekly,monthly |
| start_date | required, date (`YYYY-MM-DD`), after_or_equal:today (tanggal hari ini di Asia/Jakarta) |

Hanya role `tenant` (403 untuk role lain, dicek sebelum validasi).

Cek bisnis setelah validasi, berurutan:
1. Properti tidak tayang → 400 `Properti tidak aktif.`
2. `available < 1` → 400 `Kamar tidak tersedia.`
3. Harga untuk rent_type NULL → 400 `Tipe sewa tidak tersedia.`

Insert:
- `harga_normal` = kolom harga sesuai rent_type
- `diskon` = `property.is_flash_sale ? property.flash_sale_discount : 0`
- `price = max(harga_normal - diskon, 0)` (periode pertama), `deposit = harga_normal`, `admin_fee = settings.admin_fee` (disalin)
- `status = pending`, `stock_held = false`, `expires_at = now + 24 jam`
- `duration` sesuai aturan API.md §5.2; `period_price = harga_normal`; `flash_sale_discount = diskon`
- `end_date = start_date + duration periode` (§13.6)
- Dalam transaksi yang sama, buat invoice `initial` (§13.6), `due_date` = tanggal WIB dari `expires_at`

**Jangan** mengubah `rooms.available` (kunci baru terjadi saat checkout). Respons dan pesan persis API.md §5.2.

### 8.3 `GET /booking/my` `[auth]`

Query: `page` default 1; `limit` default 20, clamp 1..50. Hanya milik user, urut `created_at DESC, id DESC`.
Respons: `bookings[]` + `current_page`.
Item: `{id, status, rent_type, room{id,name}, start_date, price, price_label, expires_at, created_at}`.

### 8.4 Endpoint booking lain

`GET /booking/{booking}`, `POST /booking/{booking}/checkout`, `POST /booking/{booking}/cancel`,
dan `PUT /booking/{booking}/refund-account`: kontrak di API.md §5.4–5.7, implementasi di §13.

## 9. Notifikasi `[auth]` dan A/B test

### 9.1 `GET /notifications?category=&page=&limit=`

category opsional (in:booking,payment,promo,chat); page default 1; limit default 20, clamp 1..50.
Hanya milik user, urut `created_at DESC, id DESC`.
Respons: `notifications[]` `{id, category, title, body, link, read_at, created_at}` + `current_page`.
Kolom `user_notifications.link` diisi mengikuti tabel §17 (path frontend, DESIGN.md §7.0).

### 9.2 `GET /notifications/counter` → `{ "unread": n }`

### 9.3 `PATCH /notifications/{id}/read`

Tidak ada → 404; milik user lain → 403. Sudah dibaca → `read_at` tidak diubah. Respons: envelope saja.

### 9.4 `GET /config/ab-test/{experiment}?visitor_id=` (PUBLIK)

- `visitor_id` dari query (atau body JSON), required → 422. Divalidasi sebelum cek eksperimen.
- Eksperimen tidak ada → 404.
- `index = crc32.ChecksumIEEE([]byte(experiment + "|" + visitor_id)) % len(variants)`.
- Daftar eksperimen di config Go, **urutan varian tidak boleh diubah** (mengubahnya memindahkan visitor ke varian lain):
  - `discovery-communication`: `control`, `chat-first`, `booking-first`
  - `home-hero`: `image`, `video`

## 10. Data seed (`cmd/seed`, hanya dev/test)

Seed harus idempoten dan **menolak jalan jika `APP_ENV=production`**. Data minimal sesuai contoh dokumen kontrak:
(Foto tidak di-seed; `photos` bernilai `[]`.)

| Data | Isi |
|---|---|
| User 1 (owner) | `Pemilik Contoh`, `081200000001`, password `password123`, terverifikasi |
| User 2 (tenant) | `Anak Kos Contoh`, `081200000002`, password `password123`, terverifikasi |
| User 3 (admin) | `Admin Contoh`, `081200000000`, password `password123`, terverifikasi |
| Settings | `admin_fee = 0` (agar contoh draft-price di API.md tetap berlaku) |
| Rekening owner | User 1: `BCA` / `1234567890` / `Pemilik Contoh` |
| Area | `Kota Depok`, slug `kota-depok`, type `city`, `is_top = true`, lat `-6.3878486`, lng `106.8177948` |
| Properti 1 | `Kost Asti Beji`, slug `kost-asti-beji-123`, `campur`, `Kost dekat stasiun UI Depok`, `Jl. Contoh No. 42`, lat `-6.38`, lng `106.81`, aktif, `moderation_status = approved`, flash sale, diskon `250000` |
| Kamar 1 | `Kamar Tipe A`, bulanan `2250000`, available `2`, fasilitas `WiFi, AC, K. Mandi Dalam, Kasur, Akses 24 Jam` |
| Properti 2–4 | Tiga properti aktif & approved lain di Kota Depok (gender `putra`, `putri`, `campur`), masing-masing satu kamar, agar `property_total` = 4 seperti contoh search |
| Notifikasi user 2 | 2 belum dibaca, salah satunya `promo` / `Promo Ngebut!` / `Diskon s.d. Rp250.000 untuk bulan pertama.` |

## 11. Contract testing

- Setiap contoh respons di API.md menjadi fixture di `tests/contract/`, dijalankan terhadap DB hasil seed.
- Pembanding: parse kedua JSON lalu bandingkan **semantik** (abaikan urutan key; ketat soal tipe,
  `null` vs key tidak ada, `[]` vs `null`).
- Field dinamis (`token`, `can_resend_at`, `expires_at`, `created_at`, `id` baru) dicek format/regex-nya.
- Wajib ada test untuk:
  - semua 422 per field (teks pesan persis)
  - header `x-api-client` salah/kosong; token valid, invalid, dan format rusak
  - OTP: salah, expired, cooldown (422), login untuk nomor belum terverifikasi (422), limit 2/menit per nomor
  - limiter `api` 60/menit dan header 429
  - clamp limit, `has-more` / `next-offset`, `property_type: "apartemen"`
  - draft-price dengan/tanpa flash sale dan dengan harga mingguan/harian
  - harga booking setelah diskon, deposit tetap harga normal
  - `start_date` hari ini (WIB) diterima pada pukul 00:30 WIB
  - properti non-aktif: tidak muncul di search, 404 di detail/view/gallery, 400 di submit
  - kepemilikan notifikasi (403) dan pagination booking/notifikasi
  - varian A/B konsisten untuk visitor yang sama
  - worker: booking belum dibayar → `cancelled`; hold kedaluwarsa → stok kembali; owner timeout → `rejected` + refund penuh; pengingat terkirim sekali
  - checkout: idempoten (transaksi aktif dikembalikan), stok habis (400), dua checkout bersamaan untuk kamar terakhir → hanya satu berhasil
  - webhook: signature salah (403), notifikasi duplikat/terlambat/tidak berurutan tidak mengubah hasil, `settlement` setelah hold lepas
  - cancel tenant: `pending` tanpa refund; `paid` → refund `total − admin_fee`
  - owner: approve/reject hanya `paid` dan sebelum batas; payout dibuat saat approve; data owner lain → 404
  - admin: transisi moderasi, suspend user (token hilang, login 403, properti tidak tayang), refund/payout complete
  - role: tenant ke `/owner/*` atau `/admin/*` → 403; owner submit booking → 403
  - presign + daftarkan foto (pakai MinIO di Docker Compose)
  - Midtrans di-mock di test; satu test integrasi opsional ke sandbox (dijalankan manual)
  - foto: JPEG/PNG/WebP → dua WebP dengan ukuran benar, EXIF/GPS hilang, file asli terhapus, file rusak → 400
  - hapus foto & hapus kamar → key masuk `storage_deletions`; `purge_storage` retry saat storage gagal
  - ledger & audit log: UPDATE/DELETE ditolak database
  - invoice: perhitungan initial bulanan/harian/mingguan, renewal H-3, terminate H+3, perpanjangan bulanan & harian,
    penjepitan akhir bulan (mulai 31 Jan), pembayaran setelah `void` → refund
  - `complete_tenancies` melepas kamar; perpanjangan dibayar sebelum `end_date` mencegah `completed`
  - PDF: bisa dibuka tenant/owner/admin, 403 untuk lainnya, header Content-Type benar
  - akun: logout mencabut token ini saja; reset/ganti password mencabut token lain; ganti nomor via OTP
  - filter `area_id` city vs campus (radius), ulasan sekali per booking, ulasan tersembunyi tidak dihitung
  - CSV ledger: header, rentang > 366 hari → 422
  - `OTP_DEBUG_ECHO=true` + `APP_ENV=production` → server gagal start
  - `MIDTRANS_BASE_URL` terisi + `APP_ENV=production` → server gagal start

## 12. Urutan pengerjaan

Urutan resmi, beserta definisi selesai tiap tugas, ada di **`IMPLEMENTATION.md`**. Dokumen ini tidak mengatur urutan.

## 13. Pembayaran (Midtrans Snap)

Semua pembayaran menempel ke **invoice**. `POST /booking/{b}/checkout` = checkout invoice `initial` booking itu.
Langkah §13.2 berlaku untuk invoice `initial`; untuk `renewal`/`extension` langkah kunci kamar (4) dilewati,
dan syaratnya: invoice `unpaid`, booking `approved`.

### 13.1 Klien Midtrans

| Kebutuhan | Sandbox | Production |
|---|---|---|
| Buat transaksi Snap | `POST https://app.sandbox.midtrans.com/snap/v1/transactions` | `POST https://app.midtrans.com/snap/v1/transactions` |
| Cek status | `GET https://api.sandbox.midtrans.com/v2/{order_id}/status` | `GET https://api.midtrans.com/v2/{order_id}/status` |
| Expire transaksi pending | `POST https://api.sandbox.midtrans.com/v2/{order_id}/expire` | `POST https://api.midtrans.com/v2/{order_id}/expire` |

Header: `Authorization: Basic base64(MIDTRANS_SERVER_KEY + ":")`, `Accept` dan `Content-Type: application/json`.
Timeout 15 detik. Semua pemanggilan lewat interface `midtrans.Client` agar bisa di-mock.
Verifikasi ulang detail request/response terhadap dokumentasi resmi Midtrans saat implementasi.

Body pembuatan transaksi:
```json
{
  "transaction_details": { "order_id": "KOS-2-1", "gross_amount": 4250000 },
  "item_details": [
    { "id": "rent", "name": "Sewa Kamar Tipe A (Per Bulan)", "price": 2000000, "quantity": 1 },
    { "id": "deposit", "name": "Deposit", "price": 2250000, "quantity": 1 },
    { "id": "admin_fee", "name": "Biaya Admin", "price": 0, "quantity": 1 }
  ],
  "customer_details": { "first_name": "Anak Kos Contoh", "phone": "081200000002" },
  "expiry": { "start_time": "2026-11-10 17:00:00 +0700", "unit": "minutes", "duration": 15 },
  "callbacks": { "finish": "<MIDTRANS_FINISH_URL>" }
}
```
- Item `admin_fee` hanya dikirim jika > 0. Jumlah `item_details` wajib sama dengan `gross_amount`.
- `start_time` memakai waktu WIB saat transaksi dibuat; `expiry` berlaku untuk halaman dan semua metode bayar.
- `callbacks` hanya dikirim jika `MIDTRANS_FINISH_URL` diisi.
- Nama item dipotong maksimal 50 karakter.

### 13.2 Checkout (`POST /booking/{booking}/checkout`)

1. Transaksi DB: `SELECT ... FROM bookings WHERE id = $1 FOR UPDATE`.
2. Cek kepemilikan (403), status `pending` (400), `expires_at > now` (400), properti tayang (400).
3. Ada payment `pending` dengan `expires_at > now` → commit, kembalikan payment itu (200).
4. Jika `stock_held = false`: `UPDATE rooms SET available = available - 1 WHERE id = $room AND available > 0`.
   0 baris → rollback, 400 `Kamar tidak tersedia.` Berhasil → `stock_held = true`.
5. Insert payment (`pending`, `gross_amount = price + deposit + admin_fee`, `expires_at = now + 15 menit`),
   lalu isi `order_id = 'KOS-{booking_id}-{payment_id}'`. Commit.
6. **Di luar transaksi**, panggil Midtrans. Sukses → simpan `snap_token`, `redirect_url`, kembalikan 200.
7. Gagal → transaksi DB baru: payment `failure`, lepas kunci (`available + 1`, `stock_held = false`) → 502.

### 13.3 Webhook (`POST /api/payments/midtrans/notification`)

1. Parse body. Hitung `SHA512(order_id + status_code + gross_amount + MIDTRANS_SIGNATURE_KEY)` (hex huruf kecil,
   `gross_amount` persis string yang diterima, mis. `"4250000.00"`). Bandingkan dengan `signature_key`
   secara constant-time. Salah → 403 `Akses ditolak.`
2. `order_id` tidak dikenal (signature sudah valid) → catat di log lalu balas 200 `"OK"` tanpa memproses apa pun.
   Notifikasi uji dari dashboard Midtrans memakai `order_id` contoh dan menganggap non-2xx sebagai URL gagal;
   balasan 200 juga menghentikan pengulangan kiriman untuk order yang memang tidak ada.
3. Ambil status otoritatif lewat **GET status** (jangan percaya body saja). Simpan body ke `last_notification`.
4. Cocokkan `gross_amount` dengan `payments.gross_amount`; beda → log error, tandai `failure`, jangan proses.
5. Proses dengan fungsi bersama `ApplyPaymentStatus(order_id, status)` (dipakai juga oleh worker):
   - **Sukses**: `transaction_status = settlement`, atau `capture` dengan `fraud_status = accept`.
   - **Gagal**: `expire`, `cancel`, `deny`, `failure`.
   - `pending` atau `capture` + `challenge` → tidak ada perubahan.
6. Balas 200 envelope (`"OK"`) untuk semua notifikasi yang valid, termasuk yang tidak mengubah apa pun.
   Midtrans akan mengulang jika menerima non-2xx.

**`ApplyPaymentStatus` — wajib idempoten**, dalam satu transaksi DB dengan `FOR UPDATE` pada booking:

| Kondisi | Tindakan |
|---|---|
| Payment sudah final (`settlement`/gagal) | Abaikan (notifikasi duplikat atau tidak berurutan) |
| Sukses, booking `pending`, `stock_held = true` | Payment `settlement`; invoice `initial` `paid` (`paid_at = now`); booking `paid`, `paid_at = now`, `owner_deadline_at = now + 24 jam`; notifikasi tenant & owner |
| Sukses, kunci sudah lepas (bayar di detik terakhir) | Coba kunci ulang (`available - 1 WHERE available > 0`). Berhasil → seperti baris atas. Gagal → booking `rejected` (`rejected_by = system`), refund penuh alasan `stock_unavailable`, notifikasi tenant |
| Sukses, booking sudah `cancelled` oleh tenant | Payment `settlement`; refund penuh alasan `tenant_cancelled`; jangan ubah status booking |
| Gagal | Payment = status gagal; jika invoice `initial`, booking `pending` & `stock_held` → lepas kunci |
| Sukses, invoice `renewal` `unpaid` | Invoice `paid`; payout `amount = total − admin_fee`; ledger; notifikasi tenant & owner |
| Sukses, invoice `extension` `unpaid` | Invoice `paid`; `bookings.duration += periods`, `end_date = invoice.period_end`; payout; ledger; notifikasi |
| Sukses, invoice sudah `void` | Payment `settlement`; refund penuh alasan `invoice_void`; notifikasi tenant |

Setiap pembayaran sukses menulis ledger `payment_received` (+total) dan `admin_fee_earned` (+admin_fee).

### 13.4 Cancel oleh tenant (`POST /booking/{booking}/cancel`)

- `pending`: jika ada payment `pending`, panggil **GET status** dulu.
  - Ternyata `settlement` → jalankan `ApplyPaymentStatus`, lalu lanjutkan sebagai kasus `paid`.
  - Masih `pending` → panggil **expire** (best effort), tandai payment `expire`.
  - Lalu booking `cancelled` (`cancelled_by = tenant`), lepas kunci jika `stock_held`. Tanpa refund.
- `paid`: booking `cancelled`, lepas kunci, refund `amount = price + deposit` (total − `admin_fee`),
  alasan `tenant_cancelled`, status `awaiting_account`; notifikasi owner.
- Lainnya → 400.

### 13.5 Refund

- Dibuat dengan status `awaiting_account`. Tenant mengisi rekening (`API.md` §5.7) → `ready`.
- Admin mentransfer manual lalu menandai `completed` (`API-ADMIN.md` §6) → notifikasi tenant.
- Satu invoice maksimal satu refund (`UNIQUE invoice_id`). `GET /booking/{b}` menampilkan refund invoice `initial`.
- Ledger: `refund_due` saat dibuat; `refund_completed` saat selesai; refund penuh juga menulis `admin_fee_reversed`.

| Penyebab | `reason` | `amount` |
|---|---|---|
| Owner menolak | `owner_rejected` | `price + deposit + admin_fee` |
| Owner tidak merespons 24 jam | `owner_timeout` | `price + deposit + admin_fee` |
| Tenant batal setelah bayar | `tenant_cancelled` | `price + deposit` |
| Pembayaran masuk saat kamar sudah habis | `stock_unavailable` | `price + deposit + admin_fee` |
| Pembayaran masuk setelah invoice `void` | `invoice_void` | `invoice.total` |

### 13.6 Periode, tanggal, dan invoice

- **Tambah periode:** `daily` +1 hari, `weekly` +7 hari, `monthly` +1 bulan kalender dengan penjepitan akhir bulan
  dari **tanggal mulai asli** (periode ke-n = `start_date + n bulan`, dijepit ke hari terakhir bulan itu;
  mis. mulai 31 Jan → 28/29 Feb → 31 Mar).
- **Invoice `initial`:**
  - `monthly`: periode 1, `rent_amount = period_price`, `discount = flash_sale_discount`.
  - `daily`/`weekly`: seluruh `duration`, `rent_amount = period_price × duration`, `discount = flash_sale_discount`.
  - `deposit = period_price`, `admin_fee = booking.admin_fee`.
- **Invoice `renewal`** (hanya `monthly`, periode ke-2 dst.): `rent_amount = period_price`, `deposit = 0`,
  `admin_fee = setting saat dibuat`, `due_date = period_start`.
- **Invoice `extension`:** `monthly` → 1 periode mulai `end_date` saat ini (periode lanjutan ditagih lewat `renewal`,
  dan `periods = duration` yang diminta, tetapi `rent_amount` hanya 1 periode); `daily`/`weekly` → seluruh tambahan.
  `due_date = end_date − 1 hari`. Saat dibayar: `duration += periods`, `end_date` digeser sesuai jumlah periode.
- **Nomor invoice:** diisi setelah insert: `INV/{YYYY}/{MM}/{id 6 digit}`.

### 13.7 PDF invoice

Dibuat on-the-fly (tidak disimpan), A4, berisi: identitas platform, nomor, tanggal terbit, tenant (nama, nomor HP
disamarkan `0812****0002`), properti & kamar, periode, rincian baris (sewa, diskon, deposit, biaya admin), total,
status (cap **LUNAS** + tanggal bayar + metode bila `paid`), dan `order_id` Midtrans. Rupiah memakai format §5.5.
Hak akses: tenant pemilik, owner properti, admin; selain itu 403.

## 14. Worker (`cmd/worker`)

Satu proses, setiap 60 detik menjalankan job berikut berurutan. Setiap job memproses baris dengan
`SELECT ... FOR UPDATE SKIP LOCKED LIMIT 100` agar aman dijalankan lebih dari satu instance.
Berhenti rapi saat SIGTERM.

| Job | Pilih | Tindakan |
|---|---|---|
| `expire_payments` | payment `pending` dan `expires_at < now - 1 menit` | GET status Midtrans → `ApplyPaymentStatus`. Jika masih `pending`, panggil expire lalu tandai `expire` dan lepas kunci |
| `expire_bookings` | booking `pending`, `expires_at < now`, **tanpa** payment `pending` | `cancelled` (`cancelled_by = system`); lepas kunci jika `stock_held` |
| `owner_timeout` | booking `paid`, `owner_deadline_at < now` | `rejected` (`rejected_by = system`), lepas kunci, refund penuh `owner_timeout`, notifikasi tenant |
| `owner_reminder` | booking `paid`, `reminder_sent_at IS NULL`, `owner_deadline_at - 2 jam <= now < owner_deadline_at` | Notifikasi owner, isi `reminder_sent_at` |
| `issue_renewals` | booking `approved` `monthly`; periode berikutnya `p` dengan `p < end_date` dan `p − 3 hari ≤ hari ini (WIB)`; belum ada invoice `renewal` untuk `p` | Buat invoice `renewal`, notifikasi tenant |
| `renewal_due_reminder` | invoice `renewal` `unpaid`, `due_date = hari ini`, `reminder_sent_at IS NULL` | Notifikasi tenant |
| `terminate_overdue` | invoice `renewal` `unpaid`, `hari ini > due_date + 3 hari` (WIB) | Booking `terminated`, `ended_at`; invoice `void` (`overdue`); expire payment pending; lepas kunci kamar; notifikasi tenant & owner |
| `void_extensions` | invoice `extension` `unpaid` dan `hari ini ≥ booking.end_date` | Invoice `void` (`extension_expired`); expire payment pending |
| `complete_tenancies` | booking `approved`, `end_date ≤ hari ini` (WIB). Dijalankan setelah `void_extensions` | `completed`, `ended_at`; lepas kunci kamar; notifikasi tenant & owner (owner: pengingat catat deposit) |
| `purge_storage` | `storage_deletions` dengan `next_attempt_at ≤ now` | Hapus objek (404 = sukses), hapus baris; gagal → `attempts + 1`, backoff `2^attempts` menit (maks 24 jam); setelah 10 kali log error |
| `send_emails` | `email_outbox` belum terkirim, `next_attempt_at ≤ now` | Kirim; gagal → backoff seperti di atas, maks 8 kali |

Job harian (pukul 02:00 WIB, dikunci dengan advisory lock Postgres agar hanya satu instance):

| Job | Tindakan |
|---|---|
| `reconcile_payments` | Cek GET status Midtrans untuk payment yang berubah dalam 3 hari terakhir dan payment `pending` yang sudah kedaluwarsa; selisih status/nominal → `reconciliation_issues`. Jika Midtrans `settlement` tapi lokal belum, jalankan `ApplyPaymentStatus` lalu tetap catat issue |
| `cleanup_uploads` | Hapus objek `uploads/` berumur > 24 jam (cadangan bila lifecycle rule tidak didukung) |

Job mingguan (Minggu 03:00 WIB): `reconcile_storage` — daftar objek prefix `properties/` berumur > 24 jam yang
tidak ada di `property_photos` → masukkan ke `storage_deletions`. Baris foto yang objeknya hilang → log error.

**Melepas kunci** selalu: `UPDATE rooms SET available = available + 1 WHERE id = $room` dan
`stock_held = false`, dalam transaksi yang sama dengan perubahan status. Jangan pernah melepas jika `stock_held = false`.

## 15. Implementasi owner (`API-OWNER.md`)

- Semua query owner menyertakan `owner_id = <user login>`; data milik owner lain → 404.
- `held` per kamar = jumlah booking kamar itu dengan `stock_held = true`.
- Daftar booking owner hanya menampilkan booking dengan `paid_at IS NOT NULL` (pernah dibayar).
- `tenant.phone_number` diisi hanya jika status `approved`.
- Approve: booking `FOR UPDATE`; status `paid` dan `now < owner_deadline_at`; set `approved`, `responded_at`;
  insert payout untuk invoice `initial` (`amount = invoice.total − admin_fee`, `pending`), ledger `owner_earning`;
  notifikasi tenant. `stock_held` dibiarkan `true` sampai sewa `completed`/`terminated`.
- Reject: sama, set `rejected`, `rejected_by = owner`, `reject_reason`; lepas kunci; refund penuh `owner_rejected`.
- Slug properti: huruf kecil, non-alfanumerik jadi `-`, `-` ganda diringkas, lalu `-{id}` (update setelah insert).
- Deposit dikembalikan: booking `completed`/`terminated`, `deposit_returned_at IS NULL` → isi kolom,
  ledger `deposit_returned` (amount = `deposit`), audit log.
- `occupied` per kamar = booking `approved` dengan `stock_held = true`; `held` = booking `pending`/`paid` dengan `stock_held = true`.
- Foto:
  - Presign: `object_key = uploads/{owner_id}/{uuid v4}.{jpg|png|webp}`, presigned PUT 10 menit, `Content-Type` dikunci.
  - Daftar: lihat §18.
  - Hapus foto: hapus baris DB saja; trigger memasukkan kedua key ke `storage_deletions` (§14 `purge_storage`).
  - Pasang lifecycle rule bucket: objek di prefix `uploads/` dihapus otomatis setelah 1 hari.

## 16. Implementasi admin (`API-ADMIN.md`)

- `cmd/admin create --name "..." --phone 08... --password "..."`: membuat user `role = admin`,
  `phone_verified_at = now`. Menolak jika nomor sudah dipakai.
- Suspend user: `suspended_at = now` dan hapus semua `access_tokens` user itu dalam satu transaksi.
- Complete payout: `FOR UPDATE`; wajib ada `owner_bank_accounts`; salin rekening ke baris payout; ledger `payout_completed`; notifikasi owner.
- Complete refund: hanya `ready`; notifikasi tenant.
- Broadcast: satu `INSERT ... SELECT` ke `user_notifications` untuk user target yang tidak dinonaktifkan;
  `sent` = jumlah baris yang dimasukkan.
- `admin_fee` di-cache maksimal 60 detik (Redis/in-memory); PATCH settings menghapus cache.

## 17. Teks notifikasi

Waktu di teks ditulis dalam WIB, format `11 Nov 2026 17:05 WIB`. `{room}` dan `{property}` = nama.

**`link` per penerima** (path DESIGN.md §7.0): tenant → `/booking/{booking_id}`; owner untuk booking/tagihan/sewa →
`/owner/booking/{booking_id}`; owner untuk properti → `/owner/properti/{property_id}`; payout → `/owner/keuangan`;
ulasan baru → `/owner/ulasan`; broadcast promo → `null`.

| Penerima | Kejadian | Kategori | Judul | Isi |
|---|---|---|---|---|
| Tenant | Pembayaran sukses | `payment` | `Pembayaran berhasil` | `Pembayaran untuk {room} di {property} sudah kami terima. Menunggu persetujuan pemilik.` |
| Tenant | Owner approve | `booking` | `Booking disetujui` | `Pemilik menyetujui booking {room} di {property}. Mulai sewa {start_date}.` |
| Tenant | Owner reject | `booking` | `Booking ditolak` | `Pemilik menolak booking {room} di {property}. Dana akan dikembalikan penuh. Silakan isi rekening pengembalian dana.` + ` Alasan: {reason}` jika ada |
| Tenant | Owner timeout | `booking` | `Booking ditolak otomatis` | `Pemilik tidak merespons dalam 24 jam. Dana akan dikembalikan penuh. Silakan isi rekening pengembalian dana.` |
| Tenant | Kamar habis saat bayar | `booking` | `Booking dibatalkan` | `Kamar sudah tidak tersedia saat pembayaran diterima. Dana akan dikembalikan penuh. Silakan isi rekening pengembalian dana.` |
| Tenant | Refund selesai | `payment` | `Pengembalian dana selesai` | `Dana {amount} untuk booking #{id} sudah ditransfer.` |
| Owner | Booking dibayar | `booking` | `Booking baru` | `{tenant} sudah membayar booking {room} di {property}. Mohon respons sebelum {deadline}.` |
| Owner | Pengingat | `booking` | `Segera respons booking` | `Booking #{id} akan otomatis ditolak pada {deadline} jika tidak direspons.` |
| Owner | Tenant batal setelah bayar | `booking` | `Booking dibatalkan penyewa` | `{tenant} membatalkan booking {room} di {property}. Kamar kembali tersedia.` |
| Owner | Properti disetujui | `booking` | `Properti disetujui` | `{property} sudah disetujui dan dapat tayang.` |
| Owner | Properti ditolak | `booking` | `Properti ditolak` | `{property} ditolak admin. Alasan: {reason}` |
| Owner | Properti dinonaktifkan | `booking` | `Properti dinonaktifkan` | `{property} dinonaktifkan admin. Alasan: {reason}` |
| Owner | Payout selesai | `payment` | `Dana sudah ditransfer` | `Dana {amount} untuk booking #{id} sudah ditransfer ke rekening kamu.` |
| Tenant | Tagihan bulanan dibuat | `payment` | `Tagihan baru` | `Tagihan {number} sebesar {total} jatuh tempo {due_date}.` |
| Tenant | Hari jatuh tempo | `payment` | `Tagihan jatuh tempo hari ini` | `Bayar tagihan {number} paling lambat {due_date + 3 hari} agar sewa tidak berakhir.` |
| Tenant | Tagihan/perpanjangan dibayar | `payment` | `Pembayaran berhasil` | `Pembayaran tagihan {number} sudah kami terima.` |
| Owner | Tagihan/perpanjangan dibayar | `payment` | `Tagihan dibayar` | `{tenant} membayar tagihan {number} untuk {room} di {property}.` |
| Tenant | Sewa dihentikan | `booking` | `Sewa berakhir` | `Sewa {room} di {property} berakhir karena tagihan {number} tidak dibayar.` |
| Owner | Sewa dihentikan | `booking` | `Sewa penyewa berakhir` | `Sewa {tenant} di {room} berakhir karena tagihan tidak dibayar. Kamar kembali tersedia.` |
| Tenant | Sewa selesai | `booking` | `Masa sewa selesai` | `Masa sewa {room} di {property} sudah selesai. Bagikan pengalamanmu lewat ulasan.` |
| Owner | Sewa selesai | `booking` | `Masa sewa selesai` | `Sewa {tenant} di {room} selesai. Jangan lupa kembalikan deposit {deposit} dan catat di aplikasi.` |
| Owner | Ulasan baru | `booking` | `Ulasan baru` | `{property} mendapat ulasan bintang {rating}.` |

Notifikasi dibuat dalam transaksi DB yang sama dengan perubahan statusnya.
Jika penerima punya email, dalam transaksi yang sama juga dimasukkan ke `email_outbox`:
subjek = judul, isi = isi notifikasi + tautan `APP_PUBLIC_URL` (tanpa nomor rekening atau data pribadi lain).
Broadcast promo admin tidak dikirim lewat email.

## 18. Pengolahan gambar

Dijalankan **sinkron** di `POST /owner/properties/{p}/photos`, dibatasi semaphore `IMAGE_MAX_CONCURRENCY`
dan timeout 30 detik.

1. Validasi `object_key` (prefix milik owner), `StatObject`: ada dan `size ≤ UPLOAD_MAX_BYTES`.
2. Unduh ke memori. Deteksi tipe dari **isi file** (magic bytes), bukan dari header; hanya JPEG/PNG/WebP.
3. Cek dimensi sebelum decode penuh; `width × height > IMAGE_MAX_PIXELS` → 400.
4. libvips: auto-rotate EXIF → buang semua metadata → resize *fit* (tanpa crop, tanpa upscale) ke
   1600px dan 400px → encode WebP (kualitas 80 / 75, strip metadata).
5. Upload `properties/{property_id}/{uuid}.webp` dan `{uuid}_thumb.webp` (`Content-Type: image/webp`,
   `Cache-Control: public, max-age=31536000, immutable`).
6. Transaksi DB: insert `property_photos`. Jika gagal → masukkan kedua key baru ke `storage_deletions`.
7. Hapus objek `uploads/...` asli (gagal → masukkan ke `storage_deletions`).

File yang key-nya selalu baru (UUID) tidak pernah ditimpa, sehingga cache CDN aman.

## 19. Ledger & audit log

- Setiap perubahan uang menulis ledger **di transaksi DB yang sama** lewat satu fungsi `finance.Record(...)`.
- Ringkasan owner/admin dihitung dari `SUM(amount)` ledger per `type` (bukan dari tabel status), sehingga angka
  di laporan selalu bisa ditelusuri ke entri.
- Audit log ditulis lewat helper `audit.Record(ctx, action, entity, before, after)` di service layer, dalam transaksi
  yang sama dengan perubahannya. `changes` hanya field yang berubah. Field sensitif (`password`, `account_number`,
  `token_hash`) disamarkan `***`.
- Aksi yang wajib dicatat: semua `POST/PUT/PATCH/DELETE` di `/owner/*` dan `/admin/*`, approve/reject booking,
  perubahan status oleh worker (actor `NULL`), suspend user, perubahan setting.
- CSV ditulis streaming (`encoding/csv`) langsung ke response, tidak dimuat semua ke memori.

## 20. Statistik, ulasan, favorit

- `POST /listings/{id}/view` juga melakukan upsert `property_views_daily` (tanggal WIB) dalam transaksi yang sama.
- `stats.views` = `SUM(views)` dalam rentang; angka booking dihitung dari `paid_at`/`responded_at` dalam rentang;
  `earnings` dari ledger `owner_earning`.
- Ulasan: `can_review` = memenuhi syarat API.md §13.1 dan belum ada review. Nama tenant disamarkan saat
  ditampilkan publik, disimpan utuh.
- Favorit: `PUT`/`DELETE` idempoten (`ON CONFLICT DO NOTHING` / delete tanpa error jika tidak ada);
  properti harus ada (404), boleh tidak tayang saat dihapus.
