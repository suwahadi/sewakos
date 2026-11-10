# 🛡️ Dokumentasi API — Admin

Kontrak endpoint admin platform. Semua konvensi umum **sama dengan `API.md` §1**.
Detail implementasi ada di `SPEC-GO.md`.

---

## 1. Konvensi khusus admin

- Semua endpoint di bawah prefix **`/api/admin`** dan wajib `Authorization: Bearer <token>`.
- Hanya user dengan `role = admin`. Selain itu → `403` `Akses ditolak.`
- Akun admin **hanya** dibuat lewat CLI (`cmd/admin`) atau seed dev. Tidak ada endpoint pembuatan admin.
- Admin login lewat `POST /auth/login` yang sama (respons `role: "admin"`).
- Daftar memakai `page` (default 1) dan `limit` (default 20, maksimal 50), dengan `current_page` di respons.

---

## 2. Area

Objek: `{ "id", "name", "slug", "type", "is_top", "latitude", "longitude", "place_id" }`

| Endpoint | Keterangan |
|---|---|
| `GET /admin/areas?type=&page=&limit=` | Daftar → `areas[]` |
| `POST /admin/areas` | Tambah → message `"Area ditambahkan."`, `area` |
| `PATCH /admin/areas/{area}` | Ubah → message `"Area diperbarui."`, `area` |
| `DELETE /admin/areas/{area}` | Hapus → message `"Area dihapus."` |

| Field | Rule (POST; PATCH semua opsional) |
|---|---|
| name | required, string, max:255 |
| slug | required, string, regex `^[a-z0-9]+(-[a-z0-9]+)*$`, unique (`areas.slug`) |
| type | required, in:city,campus,mrt |
| is_top | boolean (default `false`) |
| latitude | required, numeric, between:-90,90 |
| longitude | required, numeric, between:-180,180 |
| place_id | opsional, string, max:255 |

Area yang masih dipakai properti tidak bisa dihapus → `400` `Area masih dipakai properti.`

---

## 3. Pengguna

Objek:
```json
{ "id": 3, "name": "Pemilik Contoh", "role": "owner", "phone_number": "081200000001",
  "email": null, "is_verified": true, "suspended_at": null,
  "created_at": "2026-11-10T10:00:00.000000Z" }
```

| Endpoint | Keterangan |
|---|---|
| `GET /admin/users?role=&q=&page=&limit=` | `q` mencari nama/nomor (min 3 karakter) → `users[]` |
| `GET /admin/users/{user}` | → `user` |
| `POST /admin/users/{user}/suspend` | Nonaktifkan → message `"Pengguna dinonaktifkan."` |
| `POST /admin/users/{user}/unsuspend` | Aktifkan kembali → message `"Pengguna diaktifkan kembali."` |

Efek nonaktif:
- Semua token user dihapus; login (password maupun OTP) → `403` `Akun dinonaktifkan.`
- Properti milik owner yang dinonaktifkan tidak tayang di pencarian maupun detail.
- Booking yang sedang berjalan tetap diproses normal (termasuk batas waktu otomatis).

Admin tidak bisa dinonaktifkan → `400` `Admin tidak dapat dinonaktifkan.`

---

## 4. Moderasi properti

Objek = objek properti owner (`API-OWNER.md` §2) ditambah `owner: { "id", "name", "phone_number" }`.

| Endpoint | Dari status | Ke status | Body | Pesan sukses |
|---|---|---|---|---|
| `GET /admin/properties?moderation_status=&q=&page=&limit=` | — | — | — | `properties[]` |
| `GET /admin/properties/{property}` | — | — | — | `property` |
| `POST /admin/properties/{property}/approve` | `pending`, `suspended` | `approved` | — | `Properti disetujui.` |
| `POST /admin/properties/{property}/reject` | `pending` | `rejected` | `reason` wajib | `Properti ditolak.` |
| `POST /admin/properties/{property}/suspend` | `approved` | `suspended` | `reason` wajib | `Properti dinonaktifkan.` |

`reason`: required, string, max:500; disimpan di `moderation_note` dan dikirim ke owner lewat notifikasi.
Transisi lain → `400` `Status properti tidak sesuai untuk tindakan ini.`

Properti yang di-suspend tetap memproses booking yang sudah `paid` (owner tetap bisa approve/reject).

---

## 5. Booking (hanya baca)

| Endpoint | Keterangan |
|---|---|
| `GET /admin/bookings?status=&page=&limit=` | Semua booking → `bookings[]` |
| `GET /admin/bookings/{booking}` | Detail + `payment`, `refund`, `payout` |

Objek booking = objek booking owner (`API-OWNER.md` §5) ditambah `admin_fee`, `admin_fee_label`,
dan `tenant.phone_number` selalu terisi. Status yang mungkin sama dengan `API.md` §5.3.

---

## 6. Refund (manual)

Objek:
```json
{
  "id": 1, "booking_id": 2, "tenant": { "id": 2, "name": "Anak Kos Contoh" },
  "amount": 4250000, "amount_label": "Rp4.250.000",
  "reason": "owner_rejected", "status": "ready",
  "bank_account": { "bank_name": "BCA", "account_number": "1234567890", "account_name": "Anak Kos Contoh" },
  "transfer_reference": null, "note": null, "completed_at": null,
  "created_at": "2026-11-10T12:00:00.000000Z"
}
```

`reason`: `owner_rejected`, `owner_timeout`, `tenant_cancelled`, `stock_unavailable`.
`status`: `awaiting_account` (rekening belum diisi tenant, `bank_account: null`), `ready`, `completed`.

| Endpoint | Keterangan |
|---|---|
| `GET /admin/refunds?status=&page=&limit=` | → `refunds[]`, urut `created_at ASC` |
| `POST /admin/refunds/{refund}/complete` | Tandai sudah ditransfer |

**Body complete:** `{ "transfer_reference": "TRX-20261110-001", "note": "Transfer BCA" }`
— `transfer_reference` required, string, max:100; `note` opsional, max:500.

Hanya dari `ready` → `completed`; selain itu `400` `Refund belum siap diproses.`
Tenant diberi notifikasi. **Sukses `200`:** message `"Refund ditandai selesai."`, berisi `refund`.

Refund dilakukan **manual** oleh admin (transfer bank atau dashboard Midtrans); API hanya mencatat.

---

## 7. Payout ke owner (manual)

Objek:
```json
{
  "id": 1, "booking_id": 2, "owner": { "id": 1, "name": "Pemilik Contoh" },
  "amount": 4250000, "amount_label": "Rp4.250.000", "status": "pending",
  "bank_account": null, "transfer_reference": null, "completed_at": null,
  "created_at": "2026-11-10T11:00:00.000000Z"
}
```

- Payout dibuat otomatis saat owner menyetujui booking. `amount = price + deposit`.
- `bank_account` menampilkan rekening owner saat ini (status `pending`) atau salinan rekening saat transfer (status `completed`).

| Endpoint | Keterangan |
|---|---|
| `GET /admin/payouts?status=&page=&limit=` | → `payouts[]`, urut `created_at ASC` |
| `POST /admin/payouts/{payout}/complete` | Body sama dengan refund complete |

Owner belum mengisi rekening → `400` `Pemilik belum mengisi rekening bank.`
Bukan `pending` → `400` `Payout sudah diproses.`
Owner diberi notifikasi. **Sukses `200`:** message `"Payout ditandai selesai."`, berisi `payout`.

---

## 8. Pengaturan

| Endpoint | Keterangan |
|---|---|
| `GET /admin/settings` | → `{ "settings": { "admin_fee": 0, "admin_fee_label": "Rp0" } }` |
| `PATCH /admin/settings` | Body `{ "admin_fee": 5000 }` → message `"Pengaturan disimpan."`, `settings` |

`admin_fee`: required, integer, between:0,1000000. Berlaku untuk booking yang **di-submit setelah** perubahan;
booking lama menyimpan nilai saat submit.

---

## 9. Broadcast notifikasi

`POST /admin/notifications/broadcast`

**Body:** `{ "target": "tenant", "title": "Promo Ngebut!", "body": "Diskon s.d. Rp250.000 untuk bulan pertama." }`

| Field | Rule |
|---|---|
| target | required, in:all,tenant,owner |
| title | required, string, max:255 |
| body | required, string, max:1000 |

Kategori selalu `promo`. Hanya ke user yang tidak dinonaktifkan.
**Sukses `200`:** message `"Notifikasi dikirim."`, `{ "sent": 120 }`.

---

## 10. Keuangan platform

### 10.1 `GET /admin/finance/summary?from=&to=`

Rentang sama seperti owner (`YYYY-MM-DD` WIB, default bulan berjalan, maks 366 hari).

```json
{
  "status": true, "meta": { "...": "OK" },
  "summary": {
    "from": "2026-11-01", "to": "2026-11-30",
    "payments_received": 12750000, "payments_received_label": "Rp12.750.000",
    "admin_fee_revenue": 15000, "admin_fee_revenue_label": "Rp15.000",
    "refunds_completed": 4250000, "refunds_completed_label": "Rp4.250.000",
    "refunds_outstanding": 0, "refunds_outstanding_label": "Rp0",
    "payouts_completed": 4250000, "payouts_completed_label": "Rp4.250.000",
    "payouts_outstanding": 4250000, "payouts_outstanding_label": "Rp4.250.000",
    "open_reconciliation_issues": 0
  }
}
```
`*_outstanding` dan `open_reconciliation_issues` dihitung seluruh waktu.

### 10.2 `GET /admin/finance/ledger?from=&to=&type=&owner_id=&page=&limit=` dan `GET /admin/finance/ledger.csv`

Seluruh entri buku besar. Bentuk dan format CSV sama dengan owner (`API-OWNER.md` §8), ditambah kolom `owner_id`.

| `type` | Tanda | Kapan |
|---|---|---|
| `payment_received` | + | Tagihan dibayar (total) |
| `admin_fee_earned` | + | Bagian biaya admin dari tagihan yang dibayar |
| `owner_earning` | + | Hak owner (tagihan pertama saat `approved`; tagihan lain saat dibayar) |
| `payout_completed` | − | Payout ditransfer |
| `refund_due` | + | Refund dibuat |
| `refund_completed` | − | Refund ditransfer |
| `admin_fee_reversed` | − | Biaya admin dikembalikan karena refund penuh |
| `deposit_returned` | catatan | Owner mencatat deposit dikembalikan (uang di luar platform) |

Koreksi tidak dilakukan dengan mengubah entri, tetapi dengan entri pembalik.

### 10.3 Rekonsiliasi Midtrans

Sistem setiap hari mencocokkan status dan nominal pembayaran lokal dengan Midtrans. Selisih dicatat sebagai *issue*.

| Endpoint | Keterangan |
|---|---|
| `GET /admin/finance/reconciliation?status=&page=&limit=` | → `issues[]` |
| `POST /admin/finance/reconciliation/{issue}/resolve` | Body `{ "note": "..." }` (required, max:500) → message `"Selisih ditandai selesai."` |

```json
{ "id": 3, "order_id": "KOS-7-12", "kind": "status_mismatch",
  "local_status": "pending", "midtrans_status": "settlement",
  "local_amount": 2250000, "midtrans_amount": 2250000,
  "status": "open", "note": null, "detected_at": "2026-12-02T19:00:00+00:00" }
```
`kind`: `status_mismatch`, `amount_mismatch`, `not_found_at_midtrans`. `status`: `open`, `resolved`.
Resolve hanya mencatat; perubahan data tetap lewat alur normal (mis. sistem memproses ulang status pembayaran).

---

## 11. Moderasi ulasan

| Endpoint | Keterangan |
|---|---|
| `GET /admin/reviews?is_hidden=&property_id=&page=&limit=` | → `reviews[]` (+ `property_id`, `is_hidden`, `hidden_reason`, nama tenant lengkap) |
| `POST /admin/reviews/{review}/hide` | Body `{ "reason": "..." }` (required, max:500) → message `"Ulasan disembunyikan."` |
| `POST /admin/reviews/{review}/unhide` | → message `"Ulasan ditampilkan kembali."` |

---

## 12. Tagihan & audit log

| Endpoint | Keterangan |
|---|---|
| `GET /admin/invoices?status=&type=&booking_id=&page=&limit=` | Semua invoice (bentuk `API.md` §12) |
| `GET /admin/audit-logs?actor_id=&action=&entity_type=&entity_id=&from=&to=&page=&limit=` | Semua audit log (bentuk `API-OWNER.md` §11), ditambah `ip` dan `user_agent` |

Audit log dan buku besar tidak bisa diubah atau dihapus oleh siapa pun lewat API maupun database aplikasi.

---

## 13. Tidak termasuk

Pengaturan eksperimen A/B tetap di kode aplikasi (mengubah urutan varian memindahkan pengunjung ke varian lain).
Notifikasi push (FCM) dan chat dijadwalkan untuk fase berikutnya.
