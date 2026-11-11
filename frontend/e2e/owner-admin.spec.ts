import { createHash, randomInt } from "node:crypto";
import { expect, test, type APIRequestContext, type Locator, type Page } from "@playwright/test";

const API_URL = process.env.E2E_API_URL ?? "http://127.0.0.1:8121/api";
const API_CLIENT_ID = process.env.API_CLIENT_ID ?? "KOSAPI WEB:WEB";
const MIDTRANS_SERVER_KEY = process.env.E2E_MIDTRANS_SERVER_KEY ?? "e2e-server-key";
const SEED_PASSWORD = "password123";

function visitorIP(): string {
  return `100.${randomInt(64, 127)}.${randomInt(0, 255)}.${randomInt(1, 254)}`;
}

function todayWIB(): string {
  return new Date(Date.now() + 7 * 3_600_000).toISOString().slice(0, 10);
}

async function login(page: Page, phone: string) {
  await page.goto("/masuk");
  await page.getByLabel("Nomor HP").fill(phone);
  await page.getByLabel("Password", { exact: true }).fill(SEED_PASSWORD);
  await page.getByRole("button", { name: "Masuk", exact: true }).click();
  await expect(page).not.toHaveURL(/\/masuk/, { timeout: 15_000 });
}

function tenantApi(request: APIRequestContext) {
  const ip = visitorIP();
  let token = "";
  const call = async <T>(method: "POST" | "PUT", path: string, data: unknown): Promise<T> => {
    const response = await request.fetch(`${API_URL}${path}`, {
      method,
      data,
      headers: { "x-api-client": API_CLIENT_ID, "X-Forwarded-For": ip, ...(token ? { Authorization: `Bearer ${token}` } : {}) },
    });
    expect(response.status(), `${method} ${path}: ${await response.text()}`).toBe(200);
    return response.json();
  };
  return {
    async register() {
      const phone = `08${String(randomInt(0, 1e10)).padStart(10, "0")}`;
      await call("POST", "/auth/register", { name: "Penyewa E2E", phone_number: phone, password: "Rahasia123", is_agree_privacy_policy: true });
      const { debug_code } = await call<{ debug_code: string }>("POST", "/auth/otp/request", { for: "tenant_verification", via: "0", destination: phone });
      ({ token } = await call<{ token: string }>("POST", "/auth/otp/check", { for: "tenant_verification", destination: phone, code: debug_code }));
    },
    async rentAndPay(roomId: number): Promise<number> {
      const { booking } = await call<{ booking: { id: number } }>("POST", "/booking/submit", { room_id: roomId, rent_type: "monthly", start_date: todayWIB() });
      const { payment } = await call<{ payment: { order_id: string; gross_amount: number } }>("POST", `/booking/${booking.id}/checkout`, {});
      const grossAmount = `${payment.gross_amount}.00`;
      const webhook = await request.post(`${API_URL}/payments/midtrans/notification`, {
        data: {
          order_id: payment.order_id, status_code: "200", gross_amount: grossAmount, transaction_status: "settlement", fraud_status: "accept",
          signature_key: createHash("sha512").update(`${payment.order_id}200${grossAmount}${MIDTRANS_SERVER_KEY}`).digest("hex"),
        },
      });
      expect(webhook.status()).toBe(200);
      return booking.id;
    },
    async setRefundAccount(bookingId: number) {
      await call("PUT", `/booking/${bookingId}/refund-account`, { bank_name: "BCA", account_number: "5550001111", account_name: "Penyewa E2E" });
    },
  };
}

async function findRow(page: Page, bookingId: number): Promise<Locator> {
  const row = page
    .locator("tr, li")
    .filter({ has: page.getByRole("link", { name: `#${bookingId}`, exact: true }) })
    .filter({ visible: true });
  const loadMore = page.getByRole("button", { name: "Muat lebih banyak" });
  while (!(await row.count()) && (await loadMore.isVisible())) {
    await loadMore.click();
    await expect(loadMore).toBeEnabled();
  }
  return row;
}

async function completeTransfer(page: Page, path: string, bookingId: number, reference: string) {
  await page.goto(path);
  const row = await findRow(page, bookingId);
  await row.getByRole("button", { name: "Tandai selesai" }).click();
  const dialog = page.getByRole("dialog");
  await dialog.getByLabel("Nomor referensi transfer").fill(reference);
  await dialog.getByRole("button", { name: "Tandai selesai" }).click();
  await expect(dialog).toBeHidden();
  await expect(row).toHaveCount(0);
}

test("owner lists a property, admin approves it, owner answers bookings, admin pays out and refunds", async ({ page, browser, request }, testInfo) => {
  test.setTimeout(120_000);
  const propertyName = `Kost E2E ${Date.now()} ${testInfo.project.name}`;
  await page.setExtraHTTPHeaders({ "X-Forwarded-For": visitorIP() });

  await login(page, "081200000001");
  await expect(page).toHaveURL(/\/owner$/);
  await page.goto("/owner/properti/baru");
  await page.getByLabel("Nama kost").fill(propertyName);
  await page.getByLabel("Kota").selectOption({ label: "Kota Depok" });
  await page.getByLabel("Alamat").fill("Jl. Margonda Raya No. 1");
  await page.getByLabel("Deskripsi").fill("Kost dekat stasiun, dibuat oleh test E2E.");
  await page.getByLabel("Latitude").fill("-6.3725");
  await page.getByLabel("Longitude").fill("106.8327");
  await page.getByRole("button", { name: "Simpan properti" }).click();
  await expect(page).toHaveURL(/\/owner\/properti\/\d+$/);
  const propertyId = Number(page.url().split("/").pop());

  await page.getByRole("button", { name: "Tambah kamar" }).click();
  await page.getByLabel("Nama kamar").fill("Kamar E2E");
  await page.getByLabel("Harga per bulan").fill("1500000");
  await page.getByLabel("Kamar kosong").fill("3");
  await page.getByRole("button", { name: "Simpan kamar" }).click();
  await expect(page.getByText("Kamar E2E")).toBeVisible();

  const admin = await browser.newPage({
    baseURL: testInfo.project.use.baseURL,
    viewport: testInfo.project.use.viewport,
    extraHTTPHeaders: { "X-Forwarded-For": visitorIP() },
  });
  await login(admin, "081200000000");
  await expect(admin).toHaveURL(/\/admin\/properti$/);
  await admin.getByLabel("Cari nama kost").fill(propertyName);
  await admin.getByRole("button", { name: "Cari", exact: true }).click();
  await admin.getByRole("link", { name: propertyName }).filter({ visible: true }).click();
  await admin.getByRole("button", { name: "Setujui properti" }).click();
  await expect(admin.getByRole("button", { name: "Nonaktifkan properti" })).toBeVisible();

  const listing = await request.get(`${API_URL}/listings/${propertyId}`, { headers: { "x-api-client": API_CLIENT_ID, "X-Forwarded-For": visitorIP() } });
  expect(listing.status()).toBe(200);
  const roomId: number = (await listing.json()).rooms[0].id;

  const tenant = tenantApi(request);
  await tenant.register();
  const approvedId = await tenant.rentAndPay(roomId);
  const rejectedId = await tenant.rentAndPay(roomId);

  await page.goto(`/owner/booking/${approvedId}`);
  await page.getByRole("button", { name: "Setujui booking" }).click();
  await expect(page.getByText("Aktif", { exact: true }).first()).toBeVisible();

  await page.goto(`/owner/booking/${rejectedId}`);
  await page.getByRole("button", { name: "Tolak booking" }).click();
  await page.getByRole("dialog").getByRole("button", { name: "Tolak booking" }).click();
  await expect(page.getByText("Ditolak", { exact: true }).first()).toBeVisible();

  await tenant.setRefundAccount(rejectedId);
  await completeTransfer(admin, "/admin/payout", approvedId, `TRX-PAYOUT-${approvedId}`);
  await completeTransfer(admin, "/admin/refund", rejectedId, `TRX-REFUND-${rejectedId}`);
  await admin.close();
});
