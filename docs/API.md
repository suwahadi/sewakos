# 📚 Dokumentasi API — Kos API (Publik & Tenant)

Referensi teknis endpoint publik dan penyewa (tenant), kontrak request/response, model data, dan alur.
Kontrak lain: `API-OWNER.md` (pemilik kost) dan `API-ADMIN.md` (admin).
Semua contoh response di bawah adalah **kontrak acuan** yang wajib dipenuhi implementasi.
Detail implementasi (skema, aturan bisnis, konfigurasi) ada di `SPEC-GO.md`.

---

## 1. Konvensi Umum

### 1.1 Base URL

```
http://127.0.0.1:8121/api
```

### 1.2 Header Wajib (semua endpoint)

| Header | Nilai | Keterangan |
|---|---|---|
| `x-api-client` | `KOSAPI WEB:WEB` | Penanda klien resmi. Salah/tanpa ini → `401` |
| `Content-Type` | `application/json` | Untuk request ber-body |
| `Authorization` | `Bearer <token>` | **Hanya endpoint `[auth]`** — token akses format `{id}|{token}` |

> Desain: penanda klien dipisah dari `Authorization` karena header itu dipakai
> untuk Bearer token user.

### 1.3 Format Envelope Response

**Sukses:**
```json
{
  "status": true,
  "meta": { "response_code": 200, "code": 200, "severity": "OK", "message": "..." },
  "...": "data spesifik endpoint"
}
```

**Gagal:**
```json
{
  "status": false,
  "meta": { "response_code": 422, "code": 422, "severity": "WARNING", "message": "Validasi gagal." },
  "messages": { "field": ["pesan per-field"] }
}
```

**Bahasa pesan:** semua `meta.message` berbahasa Indonesia (sukses tanpa teks khusus memakai `"OK"`).
Pesan per-field di `messages{}` berbahasa Inggris.

### 1.4 Kode Status

| Kode | Arti | Kapan |
|---|---|---|
| `200` | Sukses | Standard |
| `400` | Permintaan ditolak aturan bisnis | Kode OTP salah/kedaluwarsa, kamar tidak tersedia, properti non-aktif |
| `401` | Klien tidak dikenal / kredensial salah | Header `x-api-client` salah, token tidak valid, login gagal |
| `403` | Terlarang | Role tidak sesuai, akun dinonaktifkan, mengakses data user lain |
| `404` | Tidak ditemukan | Resource/eksperimen tidak ada |
| `422` | Validasi gagal | Body tidak memenuhi aturan, `messages{}` per-field |
| `405` | Method tidak didukung | Method salah untuk endpoint yang ada |
| `429` | Terlalu banyak request | Lewat rate limit (tetap memakai envelope) |
| `500` | Kesalahan server | Error internal (envelope, `severity: ERROR`) |
| `502` | Layanan pembayaran bermasalah | Midtrans gagal membuat transaksi (envelope, `severity: ERROR`) |

### 1.5 Rate Limit

| Limiter | Batas | Kunci | Berlaku |
|---|---|---|---|
| `api` | 60 req/menit | IP | Seluruh endpoint |
| `otp` | 2 req/menit | **Nomor tujuan** | `/auth/otp/request`, `/auth/otp/check` (berbagi kuota) |

Melebihi batas → `429` dengan envelope `{status:false, meta.message:"Terlalu banyak permintaan. Silakan coba lagi nanti."}`.

---

## 2. Modul Auth

### 2.1 `POST /auth/register` — daftar akun

Membuat user (phone **belum** terverifikasi). Verifikasi lewat OTP (2.2 – 2.3).

**Body:**
```json
{
  "name": "Test User",
  "phone_number": "081399900099",      // format: ^08\d{7,12}$, unik
  "email": "baru@x.test",              // unik
  "password": "Rahasia123",            // min 8 karakter
  "role": "tenant",                    // opsional: tenant (default) | owner
  "is_agree_privacy_policy": true      // wajib true
}
```

**Sukses `200`:**
```json
{
  "status": true,
  "meta": { "response_code": 200, "code": 200, "severity": "OK",
            "message": "Registrasi berhasil. Silakan verifikasi nomor handphone." },
  "user": { "id": 3, "name": "Test User", "phone_number": "081399900099" },
  "next_step": "otp_verification"
}
```

**Validasi gagal `422` (contoh nyata):**
```json
{
  "status": false,
  "meta": { "response_code": 422, "code": 422, "severity": "WARNING", "message": "Validasi gagal." },
  "messages": {
    "name": ["The name field is required."],
    "phone_number": ["The phone number field is required."],
    "password": ["The password field is required."],
    "is_agree_privacy_policy": ["The is agree privacy policy field must be accepted."]
  }
}
```

### 2.2 `POST /auth/otp/request` — kirim OTP `[throttle otp: 2/menit]`

**Body:**
```json
{
  "for": "tenant_verification",   // tenant_verification | login | reset_password
  "via": "0",                     // "0" = SMS, "1" = WhatsApp
  "destination": "081399900099"
}
```

**Sukses `200`:**
```json
{
  "status": true,
  "meta": { "...": "OK", "message": "Kode verifikasi telah dikirim." },
  "verification_code": {
    "can_resend_at": 1794245548,          // unix timestamp boleh kirim ulang
    "can_resend_after_seconds": 60
  },
  "debug_code": "1125",                   // HANYA saat OTP_DEBUG_ECHO=true (dev)
  "data": null
}
```

**Gagal:**
- `404` — `tenant_verification`: nomor tidak ditemukan / sudah terverifikasi; `login`/`reset_password`: nomor tidak ditemukan
- `422` — validasi gagal; `login` untuk nomor yang **belum** terverifikasi; minta ulang sebelum 60 detik (cooldown)
- `429` — lewat batas 2/menit

> Kode OTP: 4 digit, berlaku 300 detik, boleh kirim ulang setelah 60 detik. Saat ini pengiriman hanya ditulis ke log aplikasi.

### 2.3 `POST /auth/otp/check` — verifikasi OTP `[throttle otp: 2/menit]`

**Body:**
```json
{ "for": "tenant_verification", "destination": "081399900099", "code": "1125" }
```

**Sukses `200`** (untuk `tenant_verification` sekaligus mengisi `phone_verified_at` + terbitkan token):
```json
{
  "status": true, "meta": { "...": "OK", "message": "Nomor handphone terverifikasi." },
  "verified": true,
  "token": "1|j5knAw0v2mvolbQHdruOsjwXVFa0...",
  "user": { "id": 3, "name": "Test User", "role": "tenant" }
}
```

**Kode salah `400`:** `{ "status": false, "meta": { "...": "Kode verifikasi salah atau kedaluwarsa." } }`

### 2.4 `POST /auth/login` — login password

**Body:**
```json
{ "phone_number": "081200000002", "password": "password123" }
```

**Sukses `200`:**
```json
{
  "status": true, "meta": { "...": "OK", "message": "Login berhasil." },
  "token": "2|VjGS6DZX8K096pOaQG3XKq7BkVLtZV8eqj1pNXpn5c3c9053",
  "user": { "id": 2, "name": "Anak Kos Contoh", "role": "tenant", "is_verified": true }
}
```

**Kredensial salah `401`:** `messages: null`, message `"Nomor handphone atau password salah."`

**Akun dinonaktifkan admin `403`:** message `"Akun dinonaktifkan."` (berlaku juga untuk OTP login).

`role` bernilai `tenant`, `owner`, atau `admin`. Registrasi hanya bisa membuat `tenant` atau `owner`.

Alternatif login tanpa password: `otp/request` dengan `for: "login"` → `otp/check`.
Lupa password: `otp/request` dengan `for: "reset_password"` → `POST /auth/password/reset` (§11.2).

---

## 3. Modul Pencarian

### 3.1 `GET /area/{type}?top=true` — area populer

`{type}` ∈ `city | campus | mrt`. `top=true` hanya area bertanda `is_top`.

**Sukses `200`:**
```json
{
  "status": true, "meta": { "...": "OK" },
  "areas": [
    { "id": 1, "name": "Kota Depok", "slug": "kota-depok",
      "latitude": -6.3878486, "longitude": 106.8177948 }
  ]
}
```

### 3.2 `GET /suggestion?keyword=` — autocomplete

Keyword minimal 3 karakter (`422` jika kurang).

**Sukses `200`** (contoh keyword `depok`):
```json
{
  "status": true, "meta": { "...": "OK" },
  "suggestion": {
    "areas": [
      { "title": "Kota Depok", "area": "Kota Depok",
        "latitude": -6.3878486, "longitude": 106.8177948,
        "geocode_id": 1, "type": "area" }
    ],
    "area_source": "db",
    "landings": [ { "id": 1, "title": "Kost Asti Beji", "slug": "kost-asti-beji-123", "gender": "campur" } ],
    "room_types": [ { "id": 1, "property_id": 1, "title": "Kamar Tipe A", "slug": "kost-asti-beji-123" } ]
  }
}
```

### 3.3 `POST /listings/search` — pencarian listing

**Body:**
```json
{
  "filters": {
    "gender": [0, 1, 2],              // 0=campur, 1=putra, 2=putri (opsional)
    "price_range": [500000, 20000000],// [min, max] (opsional)
    "rent_type": 3,                   // 1=harian 2=mingguan 3=bulanan (opsional)
    "property_type": "kost",          // kost | apartemen (opsional)
    "flash_sale": false,              // hanya properti flash sale (opsional)
    "is_available": false,            // hanya kamar tersedia (opsional)
    "area_id": 1                      // area dari suggestion/area populer (opsional)
  },
  "sorting": { "field": "price", "direction": "-" },  // field: price|availability, direction: -|+
  "limit": 20,    // 1..50, default 20
  "offset": 0
}
```

**Sukses `200`:**
```json
{
  "status": true, "meta": { "...": "OK" },
  "source": "db",
  "property_total": 4,
  "total": 4,
  "limit": 3,
  "has-more": true,
  "next-offset": 3,
  "rooms": [
    { "id": 1, "property_id": 1, "name": "Kamar Tipe A", "property_name": "Kost Asti Beji",
      "area_name": "Kota Depok",
      "thumbnail_url": "https://cdn.example.com/properties/1/a1b2_thumb.webp",
      "gender": "campur", "price": 2250000, "price_label": "Rp2.250.000",
      "available": 2, "is_flash_sale": true,
      "rating_average": 4.5, "rating_count": 12 }
  ]
}
```

`filters.area_id`: area bertipe `city` → properti di kota itu; `campus`/`mrt` → properti dalam radius 3 km
dari titik area. Area tidak ada → `422`. `rating_average` (1 desimal) bernilai `null` bila belum ada ulasan.
`thumbnail_url` = thumbnail foto properti pertama (urut `sort_order`, `id`, hanya foto tanpa `room_id`), atau `null`.

### 3.4 `GET /listings/latest?limit=` — kost terbaru

Satu item per properti tayang, urut properti terbaru (`created_at` lalu `id`, menurun). Item memakai kamar dengan
harga bulanan termurah di properti itu (seri → `id` terkecil). `limit` default 12, dibatasi 1..50.

**Sukses `200`:**
```json
{
  "status": true, "meta": { "...": "OK" },
  "rooms": [
    { "id": 7, "property_id": 3, "name": "Kamar Standar", "property_name": "Kost Melati Tebet",
      "area_name": "Jakarta Selatan",
      "thumbnail_url": "https://cdn.example.com/properties/3/c3d4_thumb.webp",
      "gender": "putri", "price": 1950000, "price_label": "Rp1.950.000",
      "available": 3, "is_flash_sale": false,
      "rating_average": null, "rating_count": 0 }
  ]
}
```

Bentuk item `rooms[]` sama dengan §3.3; `price` = harga bulanan.

---

## 4. Modul Detail Properti

`{property}` = **ID properti** (integer). Properti yang tidak tayang dianggap tidak ada (`404`).
Properti tayang = `is_active = true`, sudah **disetujui admin**, dan pemiliknya tidak dinonaktifkan.

### 4.1 `GET /listings/{property}` — detail + kamar

**Sukses `200`:**
```json
{
  "status": true, "meta": { "...": "OK" },
  "property": {
    "id": 1, "name": "Kost Asti Beji", "slug": "kost-asti-beji-123",
    "gender": "campur", "description": "Kost dekat stasiun UI Depok",
    "address": "Jl. Contoh No. 42", "latitude": -6.38, "longitude": 106.81,
    "is_active": true, "is_flash_sale": true, "is_favorite": false,
    "owner": { "name": "Pemilik Contoh" },
    "area": { "name": "Kota Depok" },
    "photos": [
      { "id": 1, "url": "https://cdn.example.com/properties/1/a1b2.webp",
        "thumbnail_url": "https://cdn.example.com/properties/1/a1b2_thumb.webp", "caption": "Tampak depan" }
    ],
    "rating": { "average": 4.5, "count": 12 }
  },
  "rooms": [
    { "id": 1, "name": "Kamar Tipe A", "price_monthly": 2250000,
      "price_label": "Rp2.250.000", "available": 2,
      "facilities": ["WiFi", "AC", "K. Mandi Dalam", "Kasur", "Akses 24 Jam"],
      "monthly_duration_options": [1, 3, 6, 12],
      "photos": [] }
  ]
}
```

`property.photos` = foto umum properti; `rooms[].photos` = foto per kamar. Urut `sort_order`, lalu `id`.
`caption` bisa `null`. Tanpa foto → `[]`. Semua foto berformat WebP: `url` maks 1600px, `thumbnail_url` maks 400px.
`rating.average` bernilai `null` bila belum ada ulasan. `monthly_duration_options` = pilihan durasi sewa bulanan (periode).
`is_favorite`: endpoint ini publik; bila request membawa token `tenant` yang valid, bernilai `true` jika properti ada di
favoritnya. Tanpa token, token tidak valid, atau role lain → `false` (token tidak valid **tidak** menghasilkan `401` di sini).

### 4.2 `POST /listings/{property}/view` — counter kunjungan

**Sukses `200`:** `{ "status": true, "meta": { "...": "OK" }, "views": 1 }`

### 4.3 `GET /listings/{property}/gallery` — galeri per kamar

**Sukses `200`:**
```json
{
  "status": true, "meta": { "...": "OK" },
  "gallery": [
    { "room_id": 1, "name": "Kamar Tipe A",
      "facilities": ["WiFi", "AC"],
      "photos": [ { "id": 2, "url": "https://cdn.example.com/properties/1/c3d4.webp",
                   "thumbnail_url": "https://cdn.example.com/properties/1/c3d4_thumb.webp", "caption": null } ] }
  ]
}
```

---

## 5. Modul Booking

### 5.1 `GET /booking/draft-price/{room}?is_flash_sale=` — kalkulasi harga (PUBLIK)

Dipanggil frontend saat halaman detail dibuka, sebelum user login.
`is_flash_sale=true` hanya efektif bila properti memang sedang flash sale.
`admin_fee` diambil dari pengaturan admin (nominal tetap, default `0`).
Query opsional `duration` (default 1) hanya memengaruhi item `daily` (1–30) dan `weekly` (1–12), karena keduanya
dibayar penuh di depan: `total_off_first_payment = price × duration + deposit + admin_fee − diskon`,
`remaining_payment = price × duration + deposit + admin_fee`. Item `monthly` selalu menghitung 1 periode.

**Sukses `200`** (contoh nyata, harga 2.250.000 + flash sale 250.000):
```json
{
  "status": true, "meta": { "...": "OK" },
  "data": [{
    "sort": 1, "label": "Per Bulan", "type": "monthly",
    "price": 2250000, "price_label": "Rp2.250.000",
    "deposit": 2250000, "deposit_label": "Rp2.250.000",
    "admin_fee": 0, "admin_fee_label": "Rp0",
    "flash_sale_discount": 250000, "flash_sale_discount_label": "Rp250.000",
    "total_off_first_payment": 4250000, "total_off_first_payment_label": "Rp4.250.000",
    "remaining_payment": 4500000, "remaining_payment_label": "Rp4.500.000"
  }]
}
```

### 5.2 `POST /booking/submit` — ajukan sewa `[auth]`

**Body:**
```json
{ "room_id": 1, "rent_type": "monthly", "start_date": "2026-12-01", "duration": 3 }
```
Hanya role `tenant` (selain itu `403`). `start_date` wajib ≥ hari ini (WIB).

`duration` (opsional, jumlah periode):
- `monthly`: harus salah satu dari `monthly_duration_options` kamar; default = pilihan terkecil.
  Pembayaran pertama = periode 1; periode berikutnya ditagih bulanan (§12).
- `daily`: 1–30, `weekly`: 1–12; default 1. Seluruh durasi dibayar di pembayaran pertama.
- Nilai di luar ketentuan → `422` (`The selected duration is invalid.`).
Gagal bila `available < 1` (`400`) atau properti tidak tayang (`400`).
Submit **tidak** mengunci kamar. Kamar baru dikunci saat checkout (§5.5).

**Sukses `200`:**
```json
{
  "status": true, "meta": { "response_code": 200, "code": 200, "severity": "OK",
            "message": "Pengajuan sewa dibuat. Silakan lakukan pembayaran." },
  "booking": {
    "id": 2, "status": "pending",
    "room": { "id": 1, "name": "Kamar Tipe A" },
    "start_date": "2026-11-16",
    "expires_at": "2026-11-10T17:36:44+00:00"   // batas pembayaran 24 jam
  }
}
```

### 5.3 `GET /booking/my?page=&limit=` — daftar booking user `[auth]`

`page` default 1; `limit` default 20, maksimal 50. Urut terbaru.

**Sukses `200`:**
```json
{
  "status": true, "meta": { "...": "OK" },
  "bookings": [
    { "id": 2, "status": "paid", "rent_type": "monthly",
      "room": { "id": 1, "name": "Kamar Tipe A" },
      "start_date": "2026-12-01", "duration": 3, "end_date": "2027-03-01",
      "price": 2000000, "price_label": "Rp2.000.000",
      "expires_at": "2026-11-11T10:00:00+00:00",
      "created_at": "2026-11-10T10:00:00.000000Z" }
  ],
  "current_page": 1
}
```

### Status booking

| Status | Arti |
|---|---|
| `pending` | Menunggu pembayaran (batas `expires_at`, 24 jam sejak submit) |
| `paid` | Sudah dibayar, kamar terkunci, menunggu respons pemilik (batas 24 jam) |
| `approved` | Disetujui pemilik; masa sewa berjalan |
| `completed` | Masa sewa selesai normal (`end_date` tercapai) |
| `terminated` | Sewa berakhir otomatis karena tagihan telat > 3 hari |
| `rejected` | Ditolak pemilik, atau otomatis oleh sistem; dana dikembalikan penuh |
| `cancelled` | Dibatalkan tenant, atau tidak dibayar sampai batas waktu |

`price` adalah harga sewa **periode pertama setelah** diskon flash sale.
`end_date` = tanggal berakhir (eksklusif: hari terakhir menginap adalah sehari sebelumnya).
Untuk `daily`/`weekly`, `price` = harga per periode setelah diskon; total sewa pembayaran pertama ada di `total_payment`.

### 5.4 `GET /booking/{booking}` — detail booking `[auth]`

Hanya pemilik booking (`403` bila milik user lain, `404` bila tidak ada).

**Sukses `200`:**
```json
{
  "status": true, "meta": { "...": "OK" },
  "booking": {
    "id": 2, "status": "paid", "rent_type": "monthly",
    "room": { "id": 1, "name": "Kamar Tipe A" },
    "property": { "id": 1, "name": "Kost Asti Beji" },
    "start_date": "2026-12-01", "duration": 3, "end_date": "2027-03-01",
    "price": 2000000, "price_label": "Rp2.000.000",
    "deposit": 2250000, "deposit_label": "Rp2.250.000",
    "admin_fee": 0, "admin_fee_label": "Rp0",
    "total_payment": 4250000, "total_payment_label": "Rp4.250.000",
    "expires_at": "2026-11-11T10:00:00+00:00",
    "paid_at": "2026-11-10T10:05:00+00:00",
    "owner_deadline_at": "2026-11-11T10:05:00+00:00",
    "reject_reason": null,
    "deposit_returned_at": null,
    "can_extend": false, "can_review": false,
    "created_at": "2026-11-10T10:00:00.000000Z"
  },
  "payment": {
    "order_id": "KOS-2-1", "status": "settlement",
    "snap_token": "d16236bb-c101-4fcd-8108-72795bd232b7",
    "redirect_url": "https://app.sandbox.midtrans.com/snap/v4/redirection/d16236bb-c101-4fcd-8108-72795bd232b7",
    "expires_at": "2026-11-10T10:15:00+00:00"
  },
  "refund": null
}
```

- `paid_at`, `owner_deadline_at` → `null` selama belum dibayar.
- `total_payment` = total tagihan pertama (invoice `initial`). Tagihan lain lihat `GET /invoices?booking_id=` (§12).
- `deposit_returned_at` diisi saat pemilik menandai deposit sudah dikembalikan.
- `can_extend` / `can_review` menandakan tombol perpanjang / ulas boleh ditampilkan.
- `payment` = transaksi Midtrans terakhir, atau `null` jika belum pernah checkout.
  `status`: `pending`, `settlement`, `expire`, `cancel`, `deny`, `failure`.
- `refund` = `null`, atau `{ "amount", "amount_label", "status", "completed_at" }` dengan
  `status`: `awaiting_account` (tenant perlu isi rekening), `ready` (diproses admin), `completed`.

### 5.5 `POST /booking/{booking}/checkout` — bayar via Midtrans Snap `[auth]`

Tanpa body. Hanya role `tenant` dan pemilik booking.
Mengunci 1 kamar dan membuat transaksi Snap yang berlaku **15 menit**.
Jika masih ada transaksi `pending` yang belum kedaluwarsa, transaksi yang sama dikembalikan (tidak membuat baru).

**Sukses `200`:**
```json
{
  "status": true, "meta": { "...": "OK", "message": "Silakan selesaikan pembayaran." },
  "payment": {
    "order_id": "KOS-2-1",
    "gross_amount": 4250000, "gross_amount_label": "Rp4.250.000",
    "snap_token": "d16236bb-c101-4fcd-8108-72795bd232b7",
    "redirect_url": "https://app.sandbox.midtrans.com/snap/v4/redirection/d16236bb-c101-4fcd-8108-72795bd232b7",
    "expires_at": "2026-11-10T10:15:00+00:00"
  }
}
```

Frontend membuka Snap memakai `snap_token` (Snap.js) atau `redirect_url`.
Status akhir **tidak** diambil dari callback frontend; pantau lewat `GET /booking/{booking}`.

**Gagal:**

| Kondisi | Status | Pesan |
|---|---|---|
| Status bukan `pending` | `400` | `Booking tidak dapat dibayar.` |
| Lewat `expires_at` | `400` | `Batas waktu pembayaran sudah lewat.` |
| Properti tidak tayang | `400` | `Properti tidak aktif.` |
| Tidak ada kamar kosong | `400` | `Kamar tidak tersedia.` |
| Midtrans gagal | `502` | `Layanan pembayaran sedang bermasalah. Silakan coba lagi.` |

### 5.6 `POST /booking/{booking}/cancel` — batalkan booking `[auth]`

Tanpa body. Hanya role `tenant` dan pemilik booking.

| Status awal | Hasil | Refund | Pesan |
|---|---|---|---|
| `pending` | `cancelled`, kunci kamar dilepas | Tidak ada | `Booking dibatalkan.` |
| `paid` | `cancelled`, kunci kamar dilepas | Total dibayar − `admin_fee` | `Booking dibatalkan. Dana akan dikembalikan setelah dikurangi biaya admin.` |
| lainnya | — | — | `400` `Booking tidak dapat dibatalkan.` |

**Sukses `200`:**
```json
{
  "status": true, "meta": { "...": "OK", "message": "Booking dibatalkan. Dana akan dikembalikan setelah dikurangi biaya admin." },
  "booking": { "id": 2, "status": "cancelled" },
  "refund": { "amount": 4250000, "amount_label": "Rp4.250.000", "status": "awaiting_account", "completed_at": null }
}
```
`refund` bernilai `null` bila tidak ada pengembalian dana.

### 5.7 `PUT /booking/{booking}/refund-account` — rekening pengembalian dana `[auth]`

Hanya pemilik booking.

**Body:**
```json
{ "bank_name": "BCA", "account_number": "1234567890", "account_name": "Anak Kos Contoh" }
```

| Field | Rule |
|---|---|
| bank_name | required, string, max:100 |
| account_number | required, regex `^\d{5,20}$` |
| account_name | required, string, max:255 |

Status refund `awaiting_account` atau `ready` → disimpan dan status menjadi `ready`.

**Sukses `200`:** message `"Rekening pengembalian dana disimpan."`, berisi `refund` (bentuk sama seperti §5.4).

**Gagal `400`:** tidak ada refund → `"Tidak ada pengembalian dana untuk booking ini."`;
refund sudah `completed` → `"Pengembalian dana sudah selesai."`

---

## 6. Modul Notifikasi `[auth]`

| Endpoint | Method | Keterangan |
|---|---|---|
| `/notifications?category=&page=&limit=` | GET | Daftar (maks limit 50), filter kategori: booking/payment/promo/chat |
| `/notifications/counter` | GET | Jumlah belum dibaca → `{ "unread": 2 }` |
| `/notifications/{id}/read` | PATCH | Tandai terbaca (`403` bila milik user lain) |

**Contoh daftar `200`:**
```json
{
  "status": true, "meta": { "...": "OK" },
  "notifications": [
    { "id": 1, "category": "promo", "title": "Promo Ngebut!",
      "body": "Diskon s.d. Rp250.000 untuk bulan pertama.",
      "link": "/booking/2",
      "read_at": null, "created_at": "2026-11-10T10:00:00.000000Z" }
  ],
  "current_page": 1
}
```

`link` = path halaman frontend yang relevan (relatif, diawali `/`), atau `null` (mis. promo). Daftar path ada di
`DESIGN.md` §7.0. Frontend tidak boleh membangun link sendiri dari isi `body`.

---

## 7. Modul A/B Testing

### `GET /config/ab-test/{experiment}?visitor_id=` — penugasan varian (PUBLIK)

Varian **deterministik**: `crc32(experiment|visitor_id) % jumlah_varian` — visitor yang sama
selalu mendapat varian yang sama. Daftar eksperimen (urutan varian tidak boleh diubah):

| Eksperimen | Varian |
|---|---|
| `discovery-communication` | control, chat-first, booking-first |
| `home-hero` | image, video |

**Sukses `200`:**
```json
{ "status": true, "meta": { "...": "OK" },
  "experiment": "discovery-communication", "variant": "booking-first" }
```

**Eksperimen tidak ada `404`.** Body/parameter wajib: `visitor_id` (`422` jika kosong).

---

## 8. Model Data (ringkas)

Skema lengkap dan tabel pembayaran/payout ada di `SPEC-GO.md` §4.

```
users
  id, name, role(tenant|owner|admin), phone(unique), email(unique), password(nullable),
  phone_verified_at, email_verified_at, suspended_at?, timestamps

areas
  id, name, slug(unique), type(city|campus|mrt), is_top(bool),
  latitude, longitude, place_id(nullable), timestamps

properties
  id, owner_id → users, area_id → areas, name, slug(unique),
  gender(campur|putra|putri), description, address,
  latitude, longitude, is_active(bool), moderation_status(pending|approved|rejected|suspended),
  is_flash_sale(bool), flash_sale_discount(int), views(int), timestamps

property_photos
  id, property_id → properties (cascade), room_id? → rooms (cascade),
  object_key, caption?, sort_order, timestamps

rooms
  id, property_id → properties (cascade), name,
  price_monthly, price_weekly?, price_daily?, available(tinyint),
  capacity, facilities(json), timestamps

bookings
  id, tenant_id → users, room_id → rooms,
  rent_type(daily|weekly|monthly), start_date,
  status(pending|paid|approved|rejected|cancelled),
  price, deposit, admin_fee, expires_at, paid_at?, owner_deadline_at?, reject_reason?, timestamps

payments · refunds · payouts · owner_bank_accounts · settings   (lihat SPEC-GO.md)

user_notifications
  id, user_id → users (cascade), category(booking|payment|promo|chat),
  title, body, read_at?, timestamps

access_tokens
  id, user_id → users (cascade), token_hash(unique), last_used_at, timestamps
```

---

## 9. Alur Utama

```
┌─ REGISTRASI ──────────────────────────────────────────────┐
│ POST /auth/register        → user baru (phone belum verified)
│ POST /auth/otp/request     → OTP via SMS/WA (cooldown 60s, limit 2/mnt)
│ POST /auth/otp/check       → phone_verified_at + TOKEN
└───────────────────────────────────────────────────────────┘
┌─ LOGIN ───────────────────────────────────────────────────┐
│ POST /auth/login           → TOKEN                (password)
│   atau  /auth/otp/request (for=login) + /auth/otp/check
└───────────────────────────────────────────────────────────┘
┌─ CARI & BOOKING ──────────────────────────────────────────┐
│ GET  /area/{type}          → dropdown area
│ GET  /suggestion           → autocomplete lokasi
│ POST /listings/search      → daftar kamar (filter+sort+offset)
│ GET  /listings/latest      → kost terbaru (1 kamar termurah/kost)
│ GET  /listings/{id}        → detail
│ POST /listings/{id}/view   → counter
│ GET  /booking/draft-price  → rincian harga (PUBLIK)
│ POST /booking/submit [auth]→ pending (batas bayar 24 jam)
│ GET  /booking/my     [auth]→ riwayat
└───────────────────────────────────────────────────────────┘
┌─ BAYAR & PERSETUJUAN ─────────────────────────────────────┐
│ POST /booking/{id}/checkout → kunci kamar + Snap (15 menit)
│ (Midtrans webhook)          → paid, pemilik diberi tahu
│ Pemilik approve             → approved, dana masuk antrean payout
│ Pemilik tolak / diam 24 jam → rejected, refund penuh
│ POST /booking/{id}/cancel   → cancelled (refund − admin_fee jika paid)
│ PUT  /booking/{id}/refund-account → rekening refund
└───────────────────────────────────────────────────────────┘
┌─ MASA SEWA ───────────────────────────────────────────────┐
│ Bulanan: tagihan periode berikut dibuat H-3, jatuh tempo awal periode
│ Telat > 3 hari              → terminated, kamar kembali tersedia
│ POST /booking/{id}/extend   → tagihan perpanjangan, berlaku setelah dibayar
│ end_date tercapai           → completed, pemilik mencatat deposit kembali
│ POST /booking/{id}/review   → ulasan (sekali per booking)
└───────────────────────────────────────────────────────────┘
```

Status booking: `pending → paid → approved → completed` (jalur bahagia) · `rejected` · `cancelled` · `terminated`.

**Webhook pembayaran:** `POST /api/payments/midtrans/notification` dipanggil oleh Midtrans, bukan frontend.
Endpoint ini tidak memakai header `x-api-client` dan diverifikasi dengan `signature_key` (detail di `SPEC-GO.md`).

---

## 10. Konfigurasi Terkait

Semua konfigurasi lewat environment variable (daftar lengkap di `SPEC-GO.md` §3):

| Variabel | Nilai | Keterangan |
|---|---|---|
| `OTP_LENGTH` | `4` | Panjang kode |
| `OTP_TTL_SECONDS` | `300` | Masa berlaku kode |
| `OTP_RESEND_AFTER_SECONDS` | `60` | Cooldown kirim ulang |
| `OTP_MAX_PER_MINUTE` | `2` | Limiter `otp` per nomor |
| `OTP_DEBUG_ECHO` | `false` | ⚠️ `true` hanya di dev; server menolak start jika aktif di production |
| `API_CLIENT_ID` | `KOSAPI WEB:WEB` | Nilai wajib header `x-api-client` |
| `MIDTRANS_CLIENT_KEY` | — | Dipakai frontend untuk memuat Snap.js (bukan rahasia) |

Limiter `api` (60/menit per IP) dan daftar eksperimen A/B didefinisikan di kode aplikasi.

**Uji cepat:**
```bash
curl -s http://127.0.0.1:8121/api/auth/login \
  -H "x-api-client: KOSAPI WEB:WEB" -H "Content-Type: application/json" \
  -d '{"phone_number":"081200000002","password":"password123"}'
```

---

## 11. Modul Akun `[auth]`

Berlaku untuk semua role.

### 11.1 `POST /auth/logout` — keluar (perangkat ini)

Mencabut token yang dipakai. **Sukses `200`:** message `"Berhasil keluar."`

### 11.2 `POST /auth/password/reset` — atur ulang password (PUBLIK, limiter otp)

Didahului `POST /auth/otp/request` dengan `for: "reset_password"` (nomor tidak terdaftar → `404`).

**Body:** `{ "destination": "081200000002", "code": "1125", "password": "passwordBaru123" }`

| Field | Rule |
|---|---|
| destination | required, regex `^08\d{7,12}$` |
| code | required, string |
| password | required, string, min:8 |

Kode salah/kedaluwarsa → `400` `Kode verifikasi salah atau kedaluwarsa.`
Sukses: password diganti, **semua token dicabut**, nomor ditandai terverifikasi.
**Sukses `200`:** message `"Password berhasil diubah. Silakan login kembali."`

### 11.3 `GET /me` — profil

```json
{
  "status": true, "meta": { "...": "OK" },
  "user": { "id": 2, "name": "Anak Kos Contoh", "role": "tenant",
            "phone_number": "081200000002", "email": null, "is_verified": true }
}
```

### 11.4 `PATCH /me` — ubah profil

`{ "name"?: string max:255, "email"?: email | null, unique }` → message `"Profil diperbarui."`, berisi `user`.
Email dipakai untuk notifikasi email dan tidak diverifikasi.

### 11.5 `PUT /me/password` — ganti password

**Body:** `{ "current_password": "password123", "password": "passwordBaru123" }`

`current_password` salah → `422`, `messages.current_password = ["The current password is incorrect."]`.
Sukses: semua token **lain** dicabut (token ini tetap berlaku). Message `"Password berhasil diubah."`

### 11.6 Ganti nomor HP (limiter otp per nomor baru)

| Endpoint | Body | Hasil |
|---|---|---|
| `POST /me/phone/request` | `{ "phone_number": "081311112222" }` | Kirim OTP ke nomor baru. Respons sama dengan `otp/request` (§2.2) |
| `POST /me/phone/verify` | `{ "phone_number": "081311112222", "code": "1125" }` | Nomor diganti & terverifikasi → message `"Nomor handphone diperbarui."`, berisi `user` |

`phone_number`: required, regex `^08\d{7,12}$`, unique (`users.phone`). Cooldown 60 detik berlaku.
Kode salah → `400` `Kode verifikasi salah atau kedaluwarsa.`

---

## 12. Modul Tagihan (Invoice) `[auth]`

Setiap pembayaran ditagih lewat invoice.

| `type` | Kapan dibuat | Isi |
|---|---|---|
| `initial` | Saat submit booking | Sewa periode pertama (bulanan) atau seluruh durasi (harian/mingguan) + deposit + biaya admin − diskon |
| `renewal` | H-3 sebelum periode bulanan berikutnya | Sewa 1 periode + biaya admin |
| `extension` | Saat tenant memperpanjang | Bulanan: sewa 1 periode + biaya admin; harian/mingguan: sewa seluruh tambahan + biaya admin |

`status`: `unpaid`, `paid`, `void` (dibatalkan sistem). Harga sewa periode berikutnya memakai harga normal
saat booking dibuat (diskon flash sale hanya untuk pembayaran pertama).

### Objek invoice

```json
{
  "id": 7, "number": "INV/2026/12/000007", "booking_id": 2, "type": "renewal", "status": "unpaid",
  "period_start": "2026-12-31", "period_end": "2027-01-31",
  "rent_amount": 2250000, "rent_amount_label": "Rp2.250.000",
  "discount": 0, "discount_label": "Rp0",
  "deposit": 0, "deposit_label": "Rp0",
  "admin_fee": 0, "admin_fee_label": "Rp0",
  "total": 2250000, "total_label": "Rp2.250.000",
  "due_date": "2026-12-31", "paid_at": null,
  "issued_at": "2026-12-28T00:00:00+00:00"
}
```

- `period_end` eksklusif. `due_date` dalam WIB.
- `initial`: `due_date` = tanggal `expires_at` booking (WIB).
- `renewal`: jatuh tempo = awal periode; lewat 3 hari setelah `due_date` → booking `terminated`, invoice `void`.

### 12.1 `GET /invoices?booking_id=&status=&page=&limit=` → `invoices[]` + `current_page` (milik tenant)

### 12.2 `GET /invoices/{invoice}` → `invoice` + `payment` + `refund` (bentuk sama dengan §5.4, masing-masing bisa `null`)

Refund pada invoice selain `initial` hanya terjadi jika pembayaran masuk setelah invoice di-`void` (dikembalikan penuh).

### 12.3 `POST /invoices/{invoice}/checkout` — bayar via Snap

Respons sama dengan §5.5. Untuk invoice `initial`, perilakunya sama persis dengan `POST /booking/{booking}/checkout`
(termasuk kunci kamar). Invoice `renewal`/`extension` tidak mengunci kamar.
Invoice bukan `unpaid` → `400` `Tagihan tidak dapat dibayar.`

### 12.4 `GET /invoices/{invoice}/pdf` — invoice/kuitansi PDF

Bisa diakses tenant pemilik invoice, owner properti terkait, dan admin.
Respons **bukan JSON**: `200` dengan `Content-Type: application/pdf` dan
`Content-Disposition: attachment; filename="INV-2026-12-000007.pdf"`.
Invoice `paid` dicetak sebagai **kuitansi** (cap LUNAS); `unpaid` sebagai tagihan; `void` → `400` `Tagihan sudah dibatalkan.`
Error tetap memakai envelope JSON.

### 12.5 `POST /booking/{booking}/extend` — perpanjang sewa

Hanya role `tenant`, status `approved`, dan sebelum `end_date`.

**Body:** `{ "duration": 3 }` — bulanan: salah satu `monthly_duration_options`; harian 1–30; mingguan 1–12.

- Membuat invoice `extension`. Perpanjangan **berlaku setelah dibayar** (`duration` dan `end_date` bertambah).
- Invoice perpanjangan yang belum dibayar saat `end_date` tercapai otomatis `void`.
- Masih ada invoice `extension` yang `unpaid` → `400` `Masih ada perpanjangan yang belum dibayar.`
- Status/tanggal tidak memenuhi → `400` `Sewa tidak dapat diperpanjang.`

**Sukses `200`:** message `"Tagihan perpanjangan dibuat. Silakan lakukan pembayaran."`, berisi `invoice`.

---

## 13. Ulasan & Favorit

### 13.1 `POST /booking/{booking}/review` — beri ulasan `[auth]`

Hanya tenant pemilik booking. Boleh jika status `approved` dan `start_date` ≤ hari ini (WIB),
atau status `completed`/`terminated`. Maksimal **sekali per booking**.

**Body:** `{ "rating": 5, "comment": "Bersih dan dekat stasiun." }`

| Field | Rule |
|---|---|
| rating | required, integer, between:1,5 |
| comment | opsional, string, max:1000 |

Sudah pernah mengulas → `400` `Booking ini sudah diulas.`; belum memenuhi syarat → `400` `Booking belum bisa diulas.`
**Sukses `200`:** message `"Terima kasih atas ulasannya."`, berisi `review`.

### 13.2 `GET /listings/{property}/reviews?page=&limit=` — daftar ulasan (PUBLIK)

```json
{
  "status": true, "meta": { "...": "OK" },
  "rating": { "average": 4.5, "count": 12 },
  "reviews": [
    { "id": 1, "rating": 5, "comment": "Bersih dan dekat stasiun.",
      "tenant_name": "Anak K.", "room_name": "Kamar Tipe A",
      "created_at": "2026-12-05T10:00:00.000000Z" }
  ],
  "current_page": 1
}
```
`tenant_name` disamarkan: kata pertama utuh + inisial kata kedua. Ulasan yang disembunyikan admin tidak tampil
dan tidak dihitung. Urut terbaru.

### 13.3 Favorit `[auth]` (role `tenant`)

| Endpoint | Keterangan |
|---|---|
| `PUT /favorites/{property}` | Tambah (idempoten) → message `"Ditambahkan ke favorit."` |
| `DELETE /favorites/{property}` | Hapus (idempoten) → message `"Dihapus dari favorit."` |
| `GET /favorites?page=&limit=` | Daftar → `favorites[]` + `current_page` |

Item favorit:
```json
{ "id": 1, "name": "Kost Asti Beji", "gender": "campur", "area": { "name": "Kota Depok" },
  "thumbnail_url": "https://cdn.example.com/properties/1/a1b2_thumb.webp",
  "min_price_monthly": 2250000, "min_price_monthly_label": "Rp2.250.000",
  "rating_average": 4.5, "is_available": true }
```
Properti yang tidak lagi tayang tetap muncul dengan `is_available: false` (tidak bisa dibuka detailnya).
`thumbnail_url` = foto properti pertama, atau `null`.
