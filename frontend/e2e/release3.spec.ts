import { createHash, randomInt } from "node:crypto";
import { expect, test, type APIRequestContext, type Browser, type Page, type TestInfo } from "@playwright/test";

const API_URL = process.env.E2E_API_URL ?? "http://127.0.0.1:8121/api";
const API_CLIENT_ID = process.env.API_CLIENT_ID ?? "KOSAPI WEB:WEB";
const MIDTRANS_SERVER_KEY = process.env.E2E_MIDTRANS_SERVER_KEY ?? "e2e-server-key";
const SEED_PASSWORD = "password123";
const TENANT_PASSWORD = "Rahasia123";

function visitorIP(): string {
  return `100.${randomInt(64, 127)}.${randomInt(0, 255)}.${randomInt(1, 254)}`;
}

function todayWIB(): string {
  return new Date(Date.now() + 7 * 3_600_000).toISOString().slice(0, 10);
}

function apiClient(request: APIRequestContext) {
  const ip = visitorIP();
  let token = "";
  return {
    async call<T>(method: "GET" | "POST", path: string, data?: unknown): Promise<T> {
      const response = await request.fetch(`${API_URL}${path}`, {
        method,
        data,
        headers: { "x-api-client": API_CLIENT_ID, "X-Forwarded-For": ip, ...(token ? { Authorization: `Bearer ${token}` } : {}) },
      });
      expect(response.status(), `${method} ${path}: ${await response.text()}`).toBe(200);
      return response.json();
    },
    useToken(value: string) {
      token = value;
    },
  };
}

async function listedProperty(request: APIRequestContext, name: string): Promise<number> {
  const owner = apiClient(request);
  owner.useToken((await owner.call<{ token: string }>("POST", "/auth/login", { phone_number: "081200000001", password: SEED_PASSWORD })).token);
  const { areas } = await owner.call<{ areas: { id: number }[] }>("GET", "/area/city");
  const { property } = await owner.call<{ property: { id: number } }>("POST", "/owner/properties", {
    area_id: areas[0]!.id, name, gender: "campur", description: "Kost untuk uji ulasan dan favorit.",
    address: "Jl. Uji Rilis Tiga No. 3", latitude: -6.37, longitude: 106.83,
  });
  await owner.call("POST", `/owner/properties/${property.id}/rooms`, {
    name: "Kamar Uji", price_monthly: 1_500_000, available: 5, capacity: 1, facilities: ["WiFi"],
  });
  const admin = apiClient(request);
  admin.useToken((await admin.call<{ token: string }>("POST", "/auth/login", { phone_number: "081200000000", password: SEED_PASSWORD })).token);
  await admin.call("POST", `/admin/properties/${property.id}/approve`, {});
  return property.id;
}

async function approvedBookingForNewTenant(request: APIRequestContext, propertyId: number): Promise<{ phone: string; bookingId: number }> {
  const tenant = apiClient(request);
  const phone = `08${String(randomInt(0, 1e10)).padStart(10, "0")}`;
  await tenant.call("POST", "/auth/register", { name: "Penyewa Rilis Tiga", phone_number: phone, password: TENANT_PASSWORD, is_agree_privacy_policy: true });
  const { debug_code } = await tenant.call<{ debug_code: string }>("POST", "/auth/otp/request", { for: "tenant_verification", via: "0", destination: phone });
  tenant.useToken((await tenant.call<{ token: string }>("POST", "/auth/otp/check", { for: "tenant_verification", destination: phone, code: debug_code })).token);

  const { rooms } = await tenant.call<{ rooms: { id: number; available: number }[] }>("GET", `/listings/${propertyId}`);
  const room = rooms.find((candidate) => candidate.available > 0);
  expect(room, "test property has a free room").toBeDefined();
  const { booking } = await tenant.call<{ booking: { id: number } }>("POST", "/booking/submit", { room_id: room!.id, rent_type: "monthly", start_date: todayWIB() });
  const { payment } = await tenant.call<{ payment: { order_id: string; gross_amount: number } }>("POST", `/booking/${booking.id}/checkout`, {});
  const grossAmount = `${payment.gross_amount}.00`;
  const webhook = await request.post(`${API_URL}/payments/midtrans/notification`, {
    data: {
      order_id: payment.order_id, status_code: "200", gross_amount: grossAmount, transaction_status: "settlement", fraud_status: "accept",
      signature_key: createHash("sha512").update(`${payment.order_id}200${grossAmount}${MIDTRANS_SERVER_KEY}`).digest("hex"),
    },
  });
  expect(webhook.status()).toBe(200);

  const owner = apiClient(request);
  owner.useToken((await owner.call<{ token: string }>("POST", "/auth/login", { phone_number: "081200000001", password: SEED_PASSWORD })).token);
  await owner.call("POST", `/owner/bookings/${booking.id}/approve`, {});
  return { phone, bookingId: booking.id };
}

async function login(page: Page, phone: string, password: string) {
  await page.goto("/masuk");
  await page.getByLabel("Nomor HP").fill(phone);
  await page.getByLabel("Password", { exact: true }).fill(password);
  await page.getByRole("button", { name: "Masuk", exact: true }).click();
  await expect(page).not.toHaveURL(/\/masuk/, { timeout: 15_000 });
}

async function newVisitor(browser: Browser, testInfo: TestInfo): Promise<Page> {
  return browser.newPage({
    baseURL: testInfo.project.use.baseURL,
    viewport: testInfo.project.use.viewport,
    extraHTTPHeaders: { "X-Forwarded-For": visitorIP() },
  });
}

test("tenant reviews and favorites a kost, admin hides the review, owner sees it hidden", async ({ browser, request }, testInfo) => {
  test.setTimeout(150_000);
  const propertyName = `Kost Ulasan ${testInfo.project.name} ${Date.now()}`;
  const propertyId = await listedProperty(request, propertyName);
  const { phone, bookingId } = await approvedBookingForNewTenant(request, propertyId);
  const comment = `Kamar bersih dan pemiliknya ramah (${testInfo.project.name} ${Date.now()}).`;

  const tenant = await newVisitor(browser, testInfo);
  await login(tenant, phone, TENANT_PASSWORD);
  await tenant.goto(`/booking/${bookingId}`);
  await tenant.getByRole("button", { name: "Tulis ulasan" }).click();
  const sheet = tenant.getByRole("dialog");
  await sheet.getByRole("radio", { name: "5 bintang" }).click();
  await sheet.getByLabel("Ulasan (opsional)").fill(comment);
  await sheet.getByRole("button", { name: "Kirim ulasan" }).click();
  await expect(tenant.getByText("Terima kasih atas ulasannya.")).toBeVisible();
  await expect(tenant.getByRole("button", { name: "Tulis ulasan" })).toHaveCount(0);

  await tenant.goto(`/kost/${propertyId}`);
  await expect(tenant.getByText(comment)).toBeVisible();
  const saved = tenant.waitForResponse((response) => response.url().endsWith(`/api/favorites/${propertyId}`) && response.request().method() === "PUT");
  await tenant.getByRole("button", { name: "Simpan ke favorit" }).click();
  await expect(tenant.getByRole("button", { name: "Hapus dari favorit" })).toHaveAttribute("aria-pressed", "true");
  expect((await saved).status()).toBe(200);
  await tenant.goto("/favorit");
  const favorite = tenant.getByRole("listitem").filter({ hasText: propertyName });
  await expect(favorite).toBeVisible();
  await favorite.getByRole("button", { name: "Hapus dari favorit" }).click();
  await expect(favorite).toHaveCount(0);

  const admin = await newVisitor(browser, testInfo);
  await login(admin, "081200000000", SEED_PASSWORD);
  await admin.goto("/admin/ulasan");
  const row = admin.locator("tr, li").filter({ hasText: comment }).filter({ visible: true });
  await row.getByRole("button", { name: "Sembunyikan" }).click();
  const dialog = admin.getByRole("dialog");
  await dialog.getByLabel("Alasan").fill("Uji moderasi E2E");
  await dialog.getByRole("button", { name: "Sembunyikan" }).click();
  await expect(admin.getByText("Ulasan disembunyikan.")).toBeVisible();
  await expect(row).toHaveCount(0);
  await admin.close();

  await tenant.goto(`/kost/${propertyId}/ulasan`);
  await expect(tenant.getByText(comment)).toHaveCount(0);
  await tenant.close();

  const owner = await newVisitor(browser, testInfo);
  await login(owner, "081200000001", SEED_PASSWORD);
  await owner.goto("/owner/ulasan");
  const ownerReview = owner.locator("article").filter({ hasText: comment });
  await expect(ownerReview.getByText("Disembunyikan admin")).toBeVisible();
  await owner.close();
});
