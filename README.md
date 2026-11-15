# Sewa Kos

Sewa Kos is a boarding-house (kos) rental marketplace. Tenants search listings, book a room and pay online. Owners manage
properties, rooms, bookings and payouts. Admins moderate listings and users, and handle finance.

- **Backend:** Go API and background worker, with PostgreSQL, Redis and S3-compatible storage (MinIO).
- **Frontend:** Next.js app. It renders every page and proxies browser calls to the API.
- **Deploy:** Docker Compose on a single Debian server, behind Caddy with automatic HTTPS.

## Contents

1. [Features](#features)
2. [Architecture](#architecture)
3. [Repository layout](#repository-layout)
4. [Deploying to a server](#deploying-to-a-server)
5. [Redeploying (updating)](#redeploying-updating)
6. [Configuration reference](#configuration-reference)
7. [Public API basics](#public-api-basics)
8. [Troubleshooting](#troubleshooting)
9. [Local development](#local-development)

---

## Features

### Tenants

- **Discovery:**
  - Home page with popular areas (cities, campuses, transit stations) and the latest listings.
  - Search with autocomplete. It matches areas, property names and room names.
  - Filters: gender (mixed, male, female), price range, rent type (daily, weekly, monthly), property type, flash sale and availability.
  - Sorting by price or availability, with infinite scroll.
- **Listing detail:**
  - Photo gallery, per-room photos and facilities, and a "see all facilities" sheet.
  - Owner summary card, map location, rating and reviews.
  - Price breakdown (deposit, admin fee, flash-sale discount), share and favorite buttons.
- **Accounts:**
  - Register as tenant or owner, verify the phone number with a 4-digit OTP code, and log in with a password or OTP.
  - Reset the password with OTP, change the phone number, and edit the profile and password.
- **Bookings:**
  - Submit a booking request; the owner approves or rejects it.
  - Pay online through a hosted payment page (QRIS, virtual accounts, e-wallets and more), or cancel.
  - Enter a refund bank account when a refund applies.
- **Tenancy:**
  - Monthly invoices with online payment and PDF invoices/receipts.
  - Extend the stay, write a review after the stay, and keep a favorites list.
  - In-app notifications and email notifications.

### Owners

- **Properties:**
  - Create and edit properties (address, map pin, description, gender, photos with drag-to-reorder and a main photo).
  - Rooms with daily, weekly and monthly prices, deposit, stock, facilities and room photos.
  - Listings go live only after admin approval.
- **Bookings:**
  - Approve or reject requests within a response window. Owners get reminders before the window ends, and the request expires automatically after it.
  - Booking detail with tenant contact and payment status.
- **Finance:** earnings ledger, payout history and payout bank account.

### Admins

- **Dashboard:** key metrics.
- **Moderation:**
  - Approve or reject properties.
  - Hide reviews.
  - Activate or deactivate users, and set a user's password. This signs the user out of every device.
- **Records:** bookings and invoices, each showing the owner.
- **Finance:**
  - Payments, refunds, owner payouts and the ledger.
  - Filter by owner and export to CSV.
- **Settings:** admin fee and other platform settings.
- **Audit log:** filter by actor and action, with expandable change details.

### Background jobs (worker)

Jobs run every minute:

| Job | What it does |
|---|---|
| Expire payments | Expires unpaid payment sessions (default 15 minutes) and releases held rooms |
| Expire bookings / owner timeout | Expires bookings the owner did not answer in time, and voids their open invoices |
| Owner reminders | Reminds owners shortly before a booking request times out |
| Renewals | Issues the next invoice a few days before the period ends and sends due reminders |
| Terminate overdue | Ends tenancies whose renewal invoice is unpaid after the grace period |
| Void extensions | Voids unpaid extension invoices |
| Complete tenancies | Marks finished stays as completed so tenants can review |
| Purge storage | Deletes orphaned uploaded files |
| Send emails | Delivers queued emails through SMTP |
| Reconcile payments (daily) | Re-checks payment status with the payment gateway to catch missed notifications |

---

## Architecture

```
Browser ──HTTPS──> Caddy ──> web (Next.js, :3000) ──> api (Go, :8121) ──> PostgreSQL / Redis / MinIO
                     │                                     ▲
                     ├── /api/payments/midtrans/notification ┘   (payment gateway webhook)
                     └── :9000 ──> MinIO (public photo URLs)
```

| Service | Role |
|---|---|
| `caddy` | TLS termination (Let's Encrypt), reverse proxy, `www` redirect |
| `web` | Next.js frontend and backend-for-frontend (BFF); holds user sessions in cookies |
| `api` | Go REST API (internal only, except the payment notification path) |
| `worker` | Go background jobs (see above) |
| `postgres` | PostgreSQL 17 |
| `redis` | Rate limits, OTP codes, caches |
| `minio` / `minio-init` | S3-compatible storage for photos and PDFs; `minio-init` creates the bucket |
| `migrate` | One-shot database migration runner |

The API is not exposed to the internet. Browsers only talk to the Next.js app, which adds the API client secret and
the user's token. The only public API path is the payment notification webhook.

---

## Repository layout

```
backend/            Go module: cmd/ (api, worker, admin, seed, demoseed), internal/, db/migrations, db/queries
frontend/           Next.js app (App Router), unit tests (Vitest) and E2E tests (Playwright)
deploy/             Production stack: docker-compose.yml, Caddyfile, install.sh, .env.example
docker-compose.yml  Local development services (PostgreSQL, Redis, MinIO, Mailpit)
```

---

## Deploying to a server

### Requirements

- A fresh **Debian** server. 2 vCPU / 2 GB RAM is enough; the script adds a 2 GB swap file.
- Open ports 22, 80, 443 and 9000. The script configures `ufw` to allow them.
- A domain whose A record points to the server. An IP address also works; it then gets a short-lived IP certificate.

### First install

```bash
curl -fsSL "https://raw.githubusercontent.com/suwahadi/sewakos/main/deploy/install.sh" -o install.sh
sudo PUBLIC_HOST=example.com bash install.sh
```

The script:

1. Installs Docker, Git and `ufw`, and adds swap.
2. Clones the repository into `/opt/sewakos`.
3. Generates `/opt/sewakos/deploy/.env` with random secrets. This happens only if the file does not exist yet.
4. Builds the images, starts storage, runs migrations and starts all services.

Without `PUBLIC_HOST`, the server's public IP is used. Optional variables: `REPO_URL`, `BRANCH` (default `main`),
`APP_DIR` (default `/opt/sewakos`), `SWAP_SIZE` (default `2G`).

### After the first install

```bash
cd /opt/sewakos/deploy

# Create the first admin account
sudo docker compose run --rm api admin create --name "Admin" --phone 08xxxxxxxxxx --email admin@example.com --password '<password>'

# Optional: load demo listings with photos
sudo docker compose run --rm api demoseed --count 100
```

Then fill in the payment, OTP and SMTP keys in `deploy/.env` (see [Configuration reference](#configuration-reference))
and redeploy.

---

## Redeploying (updating)

Run this on the server to pull the latest `main` and rebuild:

```bash
curl -fsSL "https://raw.githubusercontent.com/suwahadi/sewakos/main/deploy/install.sh" -o install.sh
sudo bash install.sh
```

- The existing `deploy/.env` is kept, and new migrations are applied automatically.
- A redeploy takes about **10–15 minutes** on a 2 vCPU / 2 GB server, mostly the image builds. The site keeps running
  while images build; downtime is only the few seconds it takes to swap containers.
- Old images and build cache over 5 GB are pruned at the end.

**Changed only `deploy/.env`?** Either run the script again, or apply the change without a rebuild:

```bash
cd /opt/sewakos/deploy && sudo docker compose up -d
```

Use `up -d`, not `restart`: `restart` does not reload environment variables. Changing `MIDTRANS_CLIENT_KEY` or
`MIDTRANS_IS_PRODUCTION` needs the full script, because the frontend image embeds them at build time.

---

## Configuration reference

All operator settings live in `/opt/sewakos/deploy/.env`. The file is owned by root, so edit it with
`sudo nano /opt/sewakos/deploy/.env`. Never commit it or share its values.

### Core (generated by `install.sh`)

| Key | Default | Purpose |
|---|---|---|
| `PUBLIC_HOST` | server IP | Domain or IP the site is served on; storage is served on the same host at port 9000 |
| `ACME_PROFILE` | `classic` / `shortlived` | Certificate profile: `classic` for a domain, `shortlived` for an IP |
| `WWW_HOST` | `www.<PUBLIC_HOST>` | Alias redirected to `PUBLIC_HOST` (`localhost` for an IP) |
| `POSTGRES_PASSWORD` | random | Database owner password, used for migrations |
| `APP_DB_PASSWORD` | random | Password of the restricted role the API and worker use |
| `STORAGE_ACCESS_KEY` | `sewakos` | Object storage access key |
| `STORAGE_SECRET_KEY` | random | Object storage secret key |
| `API_CLIENT_ID` | random | Shared secret between the frontend and the API (`x-api-client` header) |

### Payment gateway

| Key | Default | Purpose |
|---|---|---|
| `MIDTRANS_SERVER_KEY` | empty | Server key from the payment dashboard |
| `MIDTRANS_CLIENT_KEY` | empty | Client key; embedded in the frontend at build time |
| `MIDTRANS_SIGNATURE_KEY` | server key | Key used only to verify payment notifications |
| `MIDTRANS_IS_PRODUCTION` | `false` | `false` = sandbox API, `true` = production API |

In the payment dashboard, set the **payment notification URL** to
`https://<PUBLIC_HOST>/api/payments/midtrans/notification`.

### OTP over WhatsApp

| Key | Default | Purpose |
|---|---|---|
| `OTP_WHATSAPP_PROVIDER` | `log` | `log` writes codes to the API log (testing); `fazpass` sends real WhatsApp messages |
| `FAZPASS_MERCHANT_KEY` | empty | Merchant key (bearer token) of the WhatsApp OTP provider |
| `FAZPASS_GATEWAY_KEY` | empty | Gateway key of the WhatsApp OTP provider |
| `FAZPASS_BASE_URL` | `https://api.fazpass.com` | Provider API base URL |

### OTP over SMS (optional)

SMS is sent through an Android phone running an SMS gateway app (cloud mode). The SMS option appears in the app only
when this channel is on.

| Key | Default | Purpose |
|---|---|---|
| `OTP_SMS_PROVIDER` | `off` | `off` hides SMS; `log` writes codes to the log; `smsgate` sends real SMS |
| `SMSGATE_USERNAME` | empty | Gateway username shown in the phone app |
| `SMSGATE_PASSWORD` | empty | Gateway password shown in the phone app |
| `SMSGATE_DEVICE_ID` | empty | Pin one device when several are registered |
| `SMSGATE_SIM_NUMBER` | `0` | SIM slot to send from (`0` = phone default, `1`, `2`, …) |
| `SMSGATE_BASE_URL` | `https://api.sms-gate.app/3rdparty/v1` | Gateway API base URL |

### OTP limits

| Key | Default | Purpose |
|---|---|---|
| `OTP_MAX_PER_DAY` | `5` | Codes per phone number per 24 hours |
| `OTP_MAX_PER_IP_PER_HOUR` | `10` | Codes per client IP per hour |
| `OTP_SMS_MAX_PER_DAY` | `50` | SMS codes per day for all users combined (WIB day) |

### Email

| Key | Default | Purpose |
|---|---|---|
| `SMTP_HOST` | empty | SMTP relay host; empty keeps emails in the API log instead of sending |
| `SMTP_PORT` | `587` | SMTP port (try `2525` if your VPS blocks 25/465/587) |
| `SMTP_USERNAME` | empty | SMTP login |
| `SMTP_PASSWORD` | empty | SMTP password or key |
| `EMAIL_FROM` | `Sewa Kos <no-reply@<PUBLIC_HOST>>` | Sender; must be verified at the SMTP provider |

**Safety checks:** the API refuses to start if a real OTP provider is selected without its keys. Check
`docker compose ps` and the API log after changing these values.

### Application rules (built-in defaults)

These defaults are not set in `deploy/.env`:

- **Payment sessions** expire after 15 minutes.
- **Owner response** window is 24 hours, with a reminder 2 hours before it ends.
- **Renewal invoices** are issued 3 days before a period ends, with a 3-day grace period.
- **Uploads** are limited to 10 MB.
- **OTP codes** are 4 digits, valid for 5 minutes, and can be resent after 60 seconds.

---

## Public API basics

The API is internal. The frontend calls it at `http://api:8121/api` inside Docker; for local development it is
`http://127.0.0.1:8121/api`. The examples below use the local URL.

### Headers

| Header | Value | Notes |
|---|---|---|
| `x-api-client` | `API_CLIENT_ID` value (`KOSAPI WEB:WEB` in local dev) | Required on every request; wrong or missing → `401` |
| `Content-Type` | `application/json` | For requests with a body |
| `Authorization` | `Bearer {id}\|{token}` | Only for authenticated endpoints |

### Response envelope

Success:

```json
{
  "status": true,
  "meta": { "response_code": 200, "code": 200, "severity": "OK", "message": "OK" },
  "...": "endpoint-specific data"
}
```

Failure (`messages` holds per-field errors, or `null`):

```json
{
  "status": false,
  "meta": { "response_code": 422, "code": 422, "severity": "WARNING", "message": "Validasi gagal." },
  "messages": { "phone_number": ["The phone number field is required."] }
}
```

`meta.message` is in Indonesian. Per-field messages are in English.

### Status codes

| Code | Meaning |
|---|---|
| `200` | Success |
| `400` | Rejected by a business rule (wrong or expired OTP, room unavailable) |
| `401` | Unknown client, invalid token or wrong credentials |
| `403` | Wrong role, deactivated account, or another user's data |
| `404` | Not found (including unpublished listings) |
| `405` | Method not allowed |
| `422` | Validation failed; see `messages` |
| `429` | Rate limit reached |
| `500` | Internal error |
| `502` | Payment gateway error |
| `503` | OTP delivery failed |

### Rate limits

| Limiter | Limit | Key |
|---|---|---|
| General | 60 requests/minute | Client IP |
| OTP request + check | 2 requests/minute (shared) | Destination phone number |
| OTP sends | 5/day per number, 10/hour per IP, 50 SMS/day overall | — |

### Public endpoints

| Method & path | Purpose |
|---|---|
| `POST /auth/register` | Create a tenant or owner account (phone not yet verified) |
| `GET /auth/otp/channels` | Available OTP channels |
| `POST /auth/otp/request` | Send an OTP code (`for`: `tenant_verification`, `login`, `reset_password`) |
| `POST /auth/otp/check` | Verify a code; returns a token for verification and login |
| `POST /auth/login` | Log in with phone and password |
| `POST /auth/password/reset` | Set a new password with an OTP code |
| `GET /area/{city\|campus\|mrt}?top=true` | Popular areas |
| `GET /suggestion?keyword=` | Autocomplete (min. 3 characters) |
| `POST /listings/search` | Search rooms with filters, sorting and paging |
| `GET /listings/latest?limit=` | Latest listings |
| `GET /listings/{property}` | Property detail with rooms |
| `POST /listings/{property}/view` | Count a page view |
| `GET /listings/{property}/gallery` | Photos grouped per room |
| `GET /listings/{property}/reviews?page=&limit=` | Reviews and rating |
| `GET /booking/draft-price/{room}?is_flash_sale=&duration=` | Price breakdown before booking |
| `GET /config/ab-test/{experiment}?visitor_id=` | A/B test variant |
| `POST /payments/midtrans/notification` | Payment gateway webhook (signature-verified) |

All other endpoints (bookings, invoices, profile, favorites, notifications, owner and admin areas) need a bearer token.

### Examples

**Send an OTP code**

```bash
curl -s http://127.0.0.1:8121/api/auth/otp/request \
  -H 'x-api-client: KOSAPI WEB:WEB' -H 'Content-Type: application/json' \
  -d '{"for":"login","via":"1","destination":"081200000002"}'
```

```json
{
  "status": true,
  "meta": { "response_code": 200, "code": 200, "severity": "OK", "message": "Kode verifikasi telah dikirim." },
  "verification_code": { "can_resend_at": 1794245548, "can_resend_after_seconds": 60 },
  "data": null
}
```

`via`: `"1"` = WhatsApp (default), `"0"` = SMS. If a channel is unavailable, the response is `429` or `503` and
`messages.via` explains it. No cooldown applies in that case, so the client can retry right away on the other channel.

**Verify the code**

```bash
curl -s http://127.0.0.1:8121/api/auth/otp/check \
  -H 'x-api-client: KOSAPI WEB:WEB' -H 'Content-Type: application/json' \
  -d '{"for":"login","destination":"081200000002","code":"1125"}'
```

```json
{
  "status": true,
  "meta": { "response_code": 200, "code": 200, "severity": "OK", "message": "Nomor handphone terverifikasi." },
  "verified": true,
  "token": "1|j5knAw0v2mvolbQHdruOsjwXVFa0...",
  "user": { "id": 2, "name": "Anak Kos Contoh", "role": "tenant" }
}
```

**Log in with a password**

```bash
curl -s http://127.0.0.1:8121/api/auth/login \
  -H 'x-api-client: KOSAPI WEB:WEB' -H 'Content-Type: application/json' \
  -d '{"phone_number":"081200000002","password":"password123"}'
```

```json
{
  "status": true,
  "meta": { "response_code": 200, "code": 200, "severity": "OK", "message": "Login berhasil." },
  "token": "2|VjGS6DZX8K096pOaQG3XKq7BkVLtZV8eqj1pNXpn5c3c9053",
  "user": { "id": 2, "name": "Anak Kos Contoh", "role": "tenant", "is_verified": true }
}
```

**Search listings**

```bash
curl -s http://127.0.0.1:8121/api/listings/search \
  -H 'x-api-client: KOSAPI WEB:WEB' -H 'Content-Type: application/json' \
  -d '{"filters":{"keyword":"depok","rent_type":3},"sorting":{"field":"price","direction":"+"},"limit":20,"offset":0}'
```

```json
{
  "status": true,
  "meta": { "response_code": 200, "code": 200, "severity": "OK", "message": "OK" },
  "property_total": 4,
  "total": 4,
  "limit": 20,
  "has-more": false,
  "next-offset": null,
  "rooms": [
    {
      "id": 1, "property_id": 1, "name": "Kamar Tipe A", "property_name": "Kost Asti Beji",
      "area_name": "Kota Depok", "thumbnail_url": "https://example.com:9000/kos-bucket/properties/1/a1b2_thumb.webp",
      "gender": "campur", "price": 2250000, "price_label": "Rp2.250.000",
      "available": 2, "is_flash_sale": true, "rating_average": 4.5, "rating_count": 12
    }
  ]
}
```

Filters (all optional):

- `gender`: `[0, 1, 2]`, where 0 = mixed, 1 = male and 2 = female.
- `price_range`: `[min, max]`.
- `rent_type`: 1 = daily, 2 = weekly, 3 = monthly.
- `property_type`: `kost` or `apartemen`.
- `flash_sale` and `is_available`: booleans.
- `area_id`: an area ID.
- `keyword`: at least 3 characters.

**Autocomplete**

```bash
curl -s 'http://127.0.0.1:8121/api/suggestion?keyword=depok' -H 'x-api-client: KOSAPI WEB:WEB'
```

```json
{
  "status": true,
  "meta": { "response_code": 200, "code": 200, "severity": "OK", "message": "OK" },
  "suggestion": {
    "areas": [ { "title": "Kota Depok", "area": "Kota Depok", "latitude": -6.3878486, "longitude": 106.8177948, "geocode_id": 1, "type": "area" } ],
    "landings": [ { "id": 1, "title": "Kost Asti Beji", "slug": "kost-asti-beji-123", "gender": "campur" } ],
    "room_types": [ { "id": 1, "property_id": 1, "title": "Kamar Tipe A", "slug": "kost-asti-beji-123" } ]
  }
}
```

**Listing detail**

```bash
curl -s http://127.0.0.1:8121/api/listings/1 -H 'x-api-client: KOSAPI WEB:WEB'
```

```json
{
  "status": true,
  "meta": { "response_code": 200, "code": 200, "severity": "OK", "message": "OK" },
  "property": {
    "id": 1, "name": "Kost Asti Beji", "slug": "kost-asti-beji-123", "gender": "campur",
    "description": "Kost dekat stasiun UI Depok", "address": "Jl. Contoh No. 42",
    "latitude": -6.38, "longitude": 106.81, "is_active": true, "is_flash_sale": true, "is_favorite": false,
    "owner": { "name": "Pemilik Contoh", "active_since": "2025-06-14T01:00:00.000000Z" },
    "area": { "name": "Kota Depok" },
    "photos": [ { "id": 1, "url": "https://…/a1b2.webp", "thumbnail_url": "https://…/a1b2_thumb.webp", "caption": "Tampak depan" } ],
    "rating": { "average": 4.5, "count": 12 }
  },
  "rooms": [
    {
      "id": 1, "name": "Kamar Tipe A", "price_monthly": 2250000, "price_label": "Rp2.250.000", "available": 2,
      "facilities": ["WiFi", "AC", "K. Mandi Dalam"], "monthly_duration_options": [1, 3, 6, 12], "photos": []
    }
  ]
}
```

**Price breakdown**

```bash
curl -s 'http://127.0.0.1:8121/api/booking/draft-price/1?is_flash_sale=true' -H 'x-api-client: KOSAPI WEB:WEB'
```

```json
{
  "status": true,
  "meta": { "response_code": 200, "code": 200, "severity": "OK", "message": "OK" },
  "data": [
    {
      "sort": 1, "label": "Per Bulan", "type": "monthly",
      "price": 2250000, "deposit": 2250000, "admin_fee": 0, "flash_sale_discount": 250000,
      "total_off_first_payment": 4250000, "remaining_payment": 4500000
    }
  ]
}
```

Each amount also has a matching `*_label` field, for example `"price_label": "Rp2.250.000"`.

**Typical errors**

```json
{ "status": false, "meta": { "response_code": 401, "code": 401, "severity": "WARNING", "message": "Nomor handphone atau password salah." }, "messages": null }
```

```json
{ "status": false, "meta": { "response_code": 429, "code": 429, "severity": "WARNING", "message": "Terlalu banyak permintaan. Silakan coba lagi nanti." }, "messages": null }
```

---

## Troubleshooting

Run every command from the deploy folder:

```bash
cd /opt/sewakos/deploy
```

### Service status

```bash
sudo docker compose ps                  # all services should be "running" / "healthy"
sudo docker compose ps -a               # include one-shot jobs (migrate, minio-init) and their exit codes
```

A service that keeps restarting usually has a configuration error. Its log says which key is missing or invalid.

### Logs

```bash
sudo docker compose logs -f api                              # follow the API
sudo docker compose logs -f api worker                       # API and background jobs together
sudo docker compose logs --tail=200 web                      # frontend
sudo docker compose logs --tail=200 caddy                    # HTTPS / certificates
sudo docker compose logs --since=30m api                     # last 30 minutes only

sudo docker compose logs -f api | grep -iE "otp|error"       # OTP delivery and errors
sudo docker compose logs -f api | grep -iE "midtrans|payment" # payment calls and notifications
sudo docker compose logs -f worker | grep -iE "expire|error" # background job activity
```

While an OTP provider is set to `log`, codes appear in the API log. This is handy for testing before real keys are set.

### Common problems

| Symptom | Likely cause and fix |
|---|---|
| `Error reading .env: Permission denied` | The file is root-only. Use `sudo` (`sudo nano .env`, `sudo docker compose …`). |
| A change in `.env` has no effect | Run `sudo docker compose up -d` (not `restart`), or rerun `install.sh`. Client-side payment keys need the full script. |
| `api` restarts right after a change | A real OTP provider is selected without its keys, or a value is invalid. Check `sudo docker compose logs --tail=50 api`. |
| OTP log shows `http 401` / `token invalid` | Wrong WhatsApp provider merchant key or gateway key. Copy them again from the provider dashboard. |
| OTP log shows `http 503` / `Vendor is down` | The WhatsApp provider is down. If SMS is on, users are offered SMS automatically; otherwise wait and retry. |
| SMS codes never arrive | The gateway phone is offline, out of credit, or the app is stopped. Open the gateway app and check its message log; verify `SMSGATE_SIM_NUMBER`. |
| "SMS is unavailable" message | The daily SMS quota (`OTP_SMS_MAX_PER_DAY`) is used up or the gateway failed. Users can switch to WhatsApp. |
| Payments stay pending | The payment notification URL is not set in the gateway dashboard, or the server/production keys do not match `MIDTRANS_IS_PRODUCTION`. The worker also expires stale sessions and reconciles daily. |
| Photos do not load | Port 9000 is blocked, or `PUBLIC_HOST` changed. Check `sudo ufw status` and `sudo docker compose logs minio caddy`. |
| Certificate errors | DNS is not pointing at the server yet, or ports 80/443 are closed. Check `sudo docker compose logs caddy`. |
| Emails not sent | `SMTP_HOST` is empty (emails are only logged), the SMTP port is blocked by the VPS (try 2525), or the sender is not verified. |
| Deploy is slow (10–15 min) | Normal on 2 vCPU / 2 GB: Go and Next.js builds are CPU-bound. The site stays up during the build. |
| `no space left on device` | Check `sudo docker system df`, then `sudo docker builder prune -f` and `sudo docker image prune -f`. |

### Database access

```bash
sudo docker compose exec postgres psql -U kos -d kos
```

Use read-only queries unless you know exactly what you are changing. Back up first:

```bash
sudo docker compose exec -T postgres pg_dump -U kos kos > backup-$(date +%F).sql
```

### Admin tasks from the command line

```bash
sudo docker compose run --rm api admin create --name "Admin" --phone 08xxxxxxxxxx --email admin@example.com --password '<password>'
sudo docker compose run --rm api demoseed --count 100
```

---

## Local development

### Prerequisites

- Go (version in `backend/go.mod`) with CGO and libvips for image processing.
- Node.js 22+ and npm.
- Docker.

### Services

```bash
docker compose up -d     # PostgreSQL :5432, Redis :6379, MinIO :9000 (console :9001), Mailpit :1025 (UI :8025)
```

### Backend

```bash
cd backend
cp .env.example .env
set -a && . ./.env && set +a
go run github.com/pressly/goose/v3/cmd/goose@v3.26.0 -dir db/migrations postgres "$DATABASE_URL" up
go run ./cmd/seed          # sample areas, listings and accounts
go run ./cmd/api           # API on :8121
go run ./cmd/worker        # background jobs
```

Seeded accounts use password `password123`:

- **Admin:** `081200000000`
- **Owner:** `081200000001`
- **Tenant:** `081200000002`

In development, OTP codes are written to the API log.

After editing `db/queries/*.sql`, regenerate the query code:

```bash
go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.29.0 generate
```

Checks:

```bash
test -z "$(gofmt -l .)" && go vet ./... && go tool staticcheck ./... && go test -race ./...
```

Contract tests need `TEST_DATABASE_URL` to point at a disposable database. The full suite takes several minutes, so
add `-timeout 25m`.

### Frontend

```bash
cd frontend
cp .env.example .env.local
npm install
npm run dev                # http://localhost:3000
```

Checks:

```bash
npm run typecheck && npm run lint && npm run test && npm run build
npm run e2e                # Playwright; needs the backend running with seed data
```
