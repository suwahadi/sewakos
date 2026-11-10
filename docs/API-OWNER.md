# 🏠 Dokumentasi API — Pemilik Kost (Owner)

Kontrak endpoint untuk pemilik kost. Semua konvensi umum (base URL, header, envelope, kode status,
rate limit, bahasa pesan, format Rupiah dan tanggal) **sama dengan `API.md` §1**.
Detail implementasi ada di `SPEC-GO.md`.

---

## 1. Konvensi khusus owner

- Semua endpoint di bawah prefix **`/api/owner`** dan wajib `Authorization: Bearer <token>`.
- Hanya user dengan `role = owner` yang tidak dinonaktifkan. Selain itu → `403` `Akses ditolak.`
- Owner hanya bisa mengakses data miliknya. Data owner lain → `404` (tidak dibedakan dari "tidak ada").
- Akun owner dibuat lewat `POST /auth/register` dengan `role: "owner"`, lalu verifikasi OTP seperti tenant.
- Daftar memakai `page` (default 1) dan `limit` (default 20, maksimal 50), dengan `current_page` di respons.

---

## 2. Properti

### Status moderasi

| `moderation_status` | Arti |
|---|---|
| `pending` | Menunggu tinjauan admin (status awal) |
| `approved` | Disetujui; tayang bila `is_active = true` |
| `rejected` | Ditolak admin; lihat `moderation_note`, perbaiki lalu ajukan ulang |
| `suspended` | Dinonaktifkan admin; hanya admin yang bisa memulihkan |

Perubahan data setelah `approved` **tidak** ditinjau ulang.

### Objek properti (owner)

```json
{
  "id": 1, "name": "Kost Asti Beji", "slug": "kost-asti-beji-1",
  "gender": "campur", "description": "Kost dekat stasiun UI Depok",
  "address": "Jl. Contoh No. 42", "latitude": -6.38, "longitude": 106.81,
  "area": { "id": 1, "name": "Kota Depok" },
  "is_active": true,
  "moderation_status": "approved", "moderation_note": null,
  "is_flash_sale": true, "flash_sale_discount": 250000, "flash_sale_discount_label": "Rp250.000",
  "views": 12, "rooms_count": 1,
  "photos": [ { "id": 1, "url": "https://cdn.example.com/properties/1/a1b2.webp",
               "thumbnail_url": "https://cdn.example.com/properties/1/a1b2_thumb.webp",
               "caption": "Tampak depan", "room_id": null, "sort_order": 0 } ],
  "rating": { "average": 4.5, "count": 12 },
  "created_at": "2026-11-10T10:00:00.000000Z"
}
```

### 2.1 `GET /owner/properties?moderation_status=&page=&limit=`

**Sukses `200`:** `{ "status": true, "meta": {...}, "properties": [ <objek properti> ], "current_page": 1 }`

### 2.2 `POST /owner/properties` — tambah properti

**Body:**
```json
{
  "area_id": 1, "name": "Kost Asti Beji", "gender": "campur",
  "description": "Kost dekat stasiun UI Depok", "address": "Jl. Contoh No. 42",
  "latitude": -6.38, "longitude": 106.81
}
```

| Field | Rule |
|---|---|
| area_id | required, integer, exists (`areas.id`) |
| name | required, string, max:255 |
| gender | required, in:campur,putra,putri |
| description | required, string, max:5000 |
| address | required, string, max:500 |
| latitude | required, numeric, between:-90,90 |
| longitude | required, numeric, between:-180,180 |

Slug dibuat otomatis: `slugify(name)-{id}`. Status awal `moderation_status = pending`, `is_active = true`.

**Sukses `200`:** message `"Properti dibuat dan menunggu persetujuan admin."`, berisi `property`.

### 2.3 `GET /owner/properties/{property}` → `property`

### 2.4 `PATCH /owner/properties/{property}` — ubah properti

Semua field §2.2 opsional, ditambah:

| Field | Rule |
|---|---|
| is_active | boolean |
| is_flash_sale | boolean |
| flash_sale_discount | integer, min:0 |

- Properti `suspended` tidak bisa diubah → `400` `Properti sedang dinonaktifkan admin.`
- Mengubah `name` **tidak** mengubah slug.

**Sukses `200`:** message `"Properti diperbarui."`, berisi `property`.

### 2.5 `POST /owner/properties/{property}/submit` — ajukan ulang

Hanya dari `rejected` → `pending` (`moderation_note` dikosongkan).
Selain itu → `400` `Properti tidak dapat diajukan ulang.`

**Sukses `200`:** message `"Properti diajukan ulang ke admin."`, berisi `property`.

---

## 3. Kamar

### Objek kamar (owner)

```json
{
  "id": 1, "property_id": 1, "name": "Kamar Tipe A",
  "price_monthly": 2250000, "price_weekly": null, "price_daily": null,
  "available": 2, "held": 1, "occupied": 0, "capacity": 1,
  "monthly_duration_options": [1, 3, 6, 12],
  "facilities": ["WiFi", "AC", "K. Mandi Dalam", "Kasur", "Akses 24 Jam"],
  "photos": []
}
```

- `available` = jumlah kamar kosong yang **belum dikunci** siapa pun (diisi owner).
- `held` = jumlah kamar yang sedang dikunci sistem (checkout berjalan atau sudah dibayar menunggu respons).
- `occupied` = jumlah kamar yang sedang disewa (booking `approved`).
- `held` dan `occupied` hanya informasi, tidak bisa diubah. Kamar kembali ke `available` saat sewa berakhir.

### 3.1 `GET /owner/properties/{property}/rooms` → `{ "rooms": [ <objek kamar> ] }`

### 3.2 `POST /owner/properties/{property}/rooms` — tambah kamar

| Field | Rule |
|---|---|
| name | required, string, max:255 |
| price_monthly | required, integer, min:1 |
| price_weekly | opsional, integer, min:1 |
| price_daily | opsional, integer, min:1 |
| available | required, integer, between:0,255 |
| capacity | required, integer, between:1,20 |
| facilities | required, array, maks 30 item, tiap item string max:50 |
| monthly_duration_options | opsional, array 1–6 item, tiap item integer between:1,12, tidak boleh duplikat (default `[1]`, disimpan terurut) |

Properti `suspended` → `400` `Properti sedang dinonaktifkan admin.`

**Sukses `200`:** message `"Kamar ditambahkan."`, berisi `room`.

### 3.3 `PATCH /owner/rooms/{room}` — ubah kamar

Semua field §3.2 opsional. `price_weekly`/`price_daily` boleh `null` untuk menghapus harga.
`available` di-set langsung (nilai absolut); sistem tetap mengurangi/menambah saat mengunci/melepas.

**Sukses `200`:** message `"Kamar diperbarui."`, berisi `room`.

### 3.4 `DELETE /owner/rooms/{room}`

Kamar yang pernah punya booking tidak bisa dihapus → `400` `Kamar sudah memiliki booking dan tidak dapat dihapus.`

**Sukses `200`:** message `"Kamar dihapus."` Foto kamar ikut terhapus beserta filenya.

---

## 4. Foto (presigned URL)

Alur: minta URL upload → klien `PUT` file langsung ke storage → daftarkan foto ke API.

### 4.1 `POST /owner/uploads/presign`

**Body:** `{ "content_type": "image/jpeg", "size": 345678 }`

| Field | Rule |
|---|---|
| content_type | required, in:image/jpeg,image/png,image/webp |
| size | required, integer, between:1,10485760 (maks 10 MB) |

**Sukses `200`:**
```json
{
  "status": true, "meta": { "...": "OK" },
  "upload": {
    "method": "PUT",
    "url": "https://storage.example.com/kos-bucket/uploads/3/0b1c....jpg?X-Amz-Signature=...",
    "headers": { "Content-Type": "image/jpeg" },
    "object_key": "uploads/3/0b1c2d3e-....jpg",
    "expires_at": "2026-11-10T10:10:00+00:00"
  }
}
```
URL berlaku 10 menit. Klien wajib mengirim header persis seperti `headers`.

### 4.2 `POST /owner/properties/{property}/photos` — daftarkan foto

**Body:** `{ "object_key": "uploads/3/0b1c2d3e-....jpg", "room_id": null, "caption": "Tampak depan" }`

| Field | Rule |
|---|---|
| object_key | required, string; harus diawali `uploads/{owner_id}/` |
| room_id | opsional, integer; harus kamar milik properti ini |
| caption | opsional, string, max:255 |

Server memeriksa objek, lalu **mengolahnya sebelum merespons**:
- Rotasi sesuai EXIF, hapus metadata (termasuk lokasi GPS).
- Konversi ke **WebP**: versi besar maks 1600px (sisi terpanjang) dan thumbnail maks 400px. Gambar kecil tidak diperbesar.
- File asli di `uploads/` dihapus. Maksimal 30 foto per properti.

**Gagal `400`:** objek tidak ditemukan/tidak valid → `"File foto tidak ditemukan atau tidak valid."`;
file rusak/bukan gambar/terlalu besar resolusinya (> 40 megapiksel) → `"File foto tidak dapat diproses."`;
lewat batas → `"Jumlah foto sudah mencapai batas."`

**Sukses `200`:** message `"Foto ditambahkan."`,
`photo: { "id", "url", "thumbnail_url", "caption", "room_id", "sort_order" }` (`sort_order` = urutan terakhir + 1).

### 4.3 `PATCH /owner/photos/{photo}` — `{ "caption"?, "sort_order"? (integer, min:0) }` → message `"Foto diperbarui."`

### 4.4 `DELETE /owner/photos/{photo}` → message `"Foto dihapus."`

Kedua file WebP dihapus otomatis dari storage (dicoba ulang bila gagal). Untuk mengganti foto: hapus lalu tambah.

---

## 5. Booking masuk

### Objek booking (owner)

```json
{
  "id": 2, "status": "paid", "rent_type": "monthly", "start_date": "2026-12-01",
  "duration": 3, "end_date": "2027-03-01",
  "tenant": { "id": 2, "name": "Anak Kos Contoh", "phone_number": null },
  "property": { "id": 1, "name": "Kost Asti Beji" },
  "room": { "id": 1, "name": "Kamar Tipe A" },
  "price": 2000000, "price_label": "Rp2.000.000",
  "deposit": 2250000, "deposit_label": "Rp2.250.000",
  "payout_amount": 4250000, "payout_amount_label": "Rp4.250.000",
  "paid_at": "2026-11-10T10:05:00+00:00",
  "owner_deadline_at": "2026-11-11T10:05:00+00:00",
  "reject_reason": null,
  "deposit_returned_at": null,
  "created_at": "2026-11-10T10:00:00.000000Z"
}
```

- Status yang mungkin: `paid`, `approved`, `completed`, `terminated`, `rejected`, `cancelled` (lihat `API.md` §5.3).
- `payout_amount` di objek ini = payout tagihan pertama. Payout tagihan berikutnya lihat §6.3.
- Owner hanya melihat booking berstatus `paid`, `approved`, `rejected`, dan `cancelled` yang **pernah dibayar**.
  Booking `pending` (belum dibayar) tidak ditampilkan.
- `tenant.phone_number` hanya terisi setelah `approved`; sebelumnya `null`.
- `payout_amount` = `price + deposit` (biaya admin milik platform).

### 5.1 `GET /owner/bookings?status=&property_id=&page=&limit=`

Urut: `paid` dengan `owner_deadline_at` terdekat dulu, lalu `created_at DESC`.
**Sukses `200`:** `{ "bookings": [ <objek booking> ], "current_page": 1 }`

### 5.2 `GET /owner/bookings/{booking}` → `booking`

### 5.3 `POST /owner/bookings/{booking}/approve`

Hanya status `paid` dan belum lewat `owner_deadline_at`. Hasil: `approved`, kamar tetap terpakai,
dana masuk antrean payout, tenant diberi notifikasi.

**Sukses `200`:** message `"Booking disetujui."`, berisi `booking`.

### 5.4 `POST /owner/bookings/{booking}/reject`

**Body:** `{ "reason": "Kamar sedang renovasi." }` — `reason` opsional, string, max:500.

Hanya status `paid` dan belum lewat batas. Hasil: `rejected`, kunci kamar dilepas,
refund **penuh** (termasuk biaya admin) ke tenant, tenant diberi notifikasi.

**Sukses `200`:** message `"Booking ditolak. Dana akan dikembalikan ke penyewa."`, berisi `booking`.

**Gagal approve/reject `400`:** `"Booking tidak dapat diproses."` (status bukan `paid` atau batas sudah lewat).

### 5.5 `POST /owner/bookings/{booking}/deposit-returned` — catat deposit sudah dikembalikan

**Body:** `{ "note": "Transfer BCA 1 Mar 2027" }` — `note` opsional, string, max:500.

Hanya status `completed` atau `terminated` dan belum pernah dicatat; deposit dikembalikan **penuh** oleh owner
di luar sistem. Selain itu → `400` `Pengembalian deposit tidak dapat dicatat.`
**Sukses `200`:** message `"Pengembalian deposit dicatat."`, berisi `booking`.

### 5.6 `GET /owner/invoices?booking_id=&status=&page=&limit=`

Invoice booking milik owner (bentuk sama dengan `API.md` §12). PDF lewat `GET /invoices/{invoice}/pdf`.

---

## 6. Rekening & payout

### 6.1 `GET /owner/bank-account`

**Sukses `200`:** `{ "bank_account": { "bank_name": "BCA", "account_number": "1234567890", "account_name": "Pemilik Contoh" } }`
atau `"bank_account": null` jika belum diisi.

### 6.2 `PUT /owner/bank-account`

Rule sama dengan rekening refund tenant (`API.md` §5.7). Wajib diisi sebelum payout pertama bisa diproses.

**Sukses `200`:** message `"Rekening disimpan."`, berisi `bank_account`.

### 6.3 `GET /owner/payouts?status=&page=&limit=`

```json
{
  "status": true, "meta": { "...": "OK" },
  "payouts": [
    { "id": 1, "booking_id": 2, "amount": 4250000, "amount_label": "Rp4.250.000",
      "status": "pending", "transfer_reference": null, "completed_at": null,
      "created_at": "2026-11-10T11:00:00.000000Z" }
  ],
  "current_page": 1
}
```
`status`: `pending` (menunggu transfer admin) atau `completed`.

---

## 7. Notifikasi untuk owner

Owner memakai endpoint notifikasi yang sama (`API.md` §6). Kejadian yang mengirim notifikasi:

| Kejadian | Kategori |
|---|---|
| Booking baru dibayar tenant | `booking` |
| Pengingat 2 jam sebelum batas respons | `booking` |
| Tenant membatalkan booking yang sudah dibayar | `booking` |
| Properti disetujui / ditolak / dinonaktifkan admin | `booking` |
| Payout sudah ditransfer | `payment` |
| Tagihan bulanan/perpanjangan dibayar tenant | `payment` |
| Sewa berakhir (`completed`) — pengingat catat deposit | `booking` |
| Sewa dihentikan karena telat bayar (`terminated`) | `booking` |
| Ulasan baru | `booking` |

Teks lengkap ada di `SPEC-GO.md`.

---

## 8. Keuangan

Saldo dihitung dari **buku besar** (*ledger*) yang hanya bisa ditambah, tidak bisa diubah atau dihapus.

### 8.1 `GET /owner/finance/summary?from=&to=`

`from`/`to` format `YYYY-MM-DD` (WIB, inklusif); default bulan berjalan; rentang maks 366 hari.

```json
{
  "status": true, "meta": { "...": "OK" },
  "summary": {
    "from": "2026-11-01", "to": "2026-11-30",
    "earnings": 4250000, "earnings_label": "Rp4.250.000",
    "payouts_completed": 0, "payouts_completed_label": "Rp0",
    "pending_balance": 4250000, "pending_balance_label": "Rp4.250.000",
    "deposits_held": 2250000, "deposits_held_label": "Rp2.250.000",
    "paid_invoices": 1
  }
}
```

| Field | Arti |
|---|---|
| `earnings` | Hak owner dari tagihan yang dibayar dalam rentang (sewa + deposit, tanpa biaya admin) |
| `payouts_completed` | Payout yang ditransfer dalam rentang |
| `pending_balance` | Saldo belum ditransfer (**seluruh waktu**, tidak terpengaruh rentang) |
| `deposits_held` | Deposit di tangan owner yang belum dicatat kembali (**seluruh waktu**) |

### 8.2 `GET /owner/finance/ledger?from=&to=&type=&page=&limit=`

```json
{
  "status": true, "meta": { "...": "OK" },
  "entries": [
    { "id": 15, "occurred_at": "2026-11-10T11:00:00+00:00", "type": "owner_earning",
      "amount": 4250000, "amount_label": "Rp4.250.000",
      "booking_id": 2, "invoice_number": "INV/2026/11/000002",
      "description": "Sewa + deposit Kamar Tipe A (Kost Asti Beji)" }
  ],
  "current_page": 1
}
```

`type` untuk owner: `owner_earning` (+), `payout_completed` (−), `deposit_returned` (catatan, `amount` = deposit).
Urut `occurred_at DESC`.

### 8.3 `GET /owner/finance/ledger.csv?from=&to=&type=`

Respons **bukan JSON**: `text/csv; charset=utf-8`, `Content-Disposition: attachment; filename="ledger-2026-11-01-2026-11-30.csv"`.
Kolom: `id,occurred_at_wib,type,amount,booking_id,invoice_number,description`. Pemisah koma, angka tanpa format.
Rentang maks 366 hari. Error tetap envelope JSON.

---

## 9. Statistik

### 9.1 `GET /owner/stats?from=&to=&property_id=`

```json
{
  "status": true, "meta": { "...": "OK" },
  "stats": {
    "from": "2026-11-01", "to": "2026-11-30",
    "views": 340, "bookings_paid": 3, "bookings_approved": 2, "bookings_rejected": 1,
    "earnings": 8500000, "earnings_label": "Rp8.500.000",
    "occupancy": { "occupied": 2, "total": 5, "rate": 0.4 },
    "rating": { "average": 4.5, "count": 12 }
  }
}
```
- `views` dan angka booking mengikuti rentang tanggal.
- `occupancy` adalah kondisi **saat ini**: `total = available + held + occupied`, `rate` dibulatkan 2 desimal (`0` jika total 0).

---

## 10. Ulasan

`GET /owner/reviews?property_id=&page=&limit=` → ulasan properti milik owner (bentuk sama dengan `API.md` §13.2,
ditambah `property_id` dan `is_hidden`). Owner **tidak** bisa membalas atau menghapus ulasan.

---

## 11. Audit log

`GET /owner/audit-logs?entity_type=&from=&to=&page=&limit=` → aksi yang dilakukan owner sendiri dan aksi admin
terhadap properti/kamar/booking milik owner.

```json
{ "id": 120, "created_at": "2026-11-10T11:00:00+00:00",
  "actor": { "id": 1, "name": "Pemilik Contoh", "role": "owner" },
  "action": "booking.approve", "entity_type": "booking", "entity_id": 2,
  "changes": { "status": ["paid", "approved"] } }
```
`changes` hanya memuat field yang berubah: `{ field: [sebelum, sesudah] }`. Data sensitif (password, nomor rekening)
disamarkan.
