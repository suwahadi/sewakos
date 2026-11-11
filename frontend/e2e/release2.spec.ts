import { execFileSync } from "node:child_process";
import { createHash, randomInt } from "node:crypto";
import { expect, test, type APIRequestContext, type Browser, type Page, type TestInfo } from "@playwright/test";

const API_URL = process.env.E2E_API_URL ?? "http://127.0.0.1:8121/api";
const API_CLIENT_ID = process.env.API_CLIENT_ID ?? "KOSAPI WEB:WEB";
const MIDTRANS_SERVER_KEY = process.env.E2E_MIDTRANS_SERVER_KEY ?? "e2e-server-key";
const DATABASE_URL = process.env.E2E_DATABASE_URL ?? "postgres://kos:kos@localhost:5432/kos?sslmode=disable";
const SEED_PASSWORD = "password123";
const TENANT_PASSWORD = "Rahasia123";
const SEEDED_PROPERTY_ID = 4;
const SNAP_STUB = "window.snap = { pay: function (token, callbacks) { callbacks.onPending(); } };";

function visitorIP(): string {
  return `100.${randomInt(64, 127)}.${randomInt(0, 255)}.${randomInt(1, 254)}`;
}

function todayWIB(): string {
  return new Date(Date.now() + 7 * 3_600_000).toISOString().slice(0, 10);
}

function sql(statement: string): string {
  return execFileSync("psql", [DATABASE_URL, "-v", "ON_ERROR_STOP=1", "-Atc", statement], { encoding: "utf8" }).trim();
}

async function settle(request: APIRequestContext, payment: { order_id: string; gross_amount: number }) {
  const grossAmount = `${payment.gross_amount}.00`;
  const webhook = await request.post(`${API_URL}/payments/midtrans/notification`, {
    data: {
      order_id: payment.order_id,
      status_code: "200",
      gross_amount: grossAmount,
      transaction_status: "settlement",
      fraud_status: "accept",
      signature_key: createHash("sha512").update(`${payment.order_id}200${grossAmount}${MIDTRANS_SERVER_KEY}`).digest("hex"),
    },
  });
  expect(webhook.status()).toBe(200);
}

async function registerTenantWithPaidBooking(request: APIRequestContext): Promise<{ phone: string; bookingId: number }> {
  const ip = visitorIP();
  let token = "";
  const call = async <T>(method: "GET" | "POST", path: string, data?: unknown): Promise<T> => {
    const response = await request.fetch(`${API_URL}${path}`, {
      method,
      data,
      headers: { "x-api-client": API_CLIENT_ID, "X-Forwarded-For": ip, ...(token ? { Authorization: `Bearer ${token}` } : {}) },
    });
    expect(response.status(), `${method} ${path}: ${await response.text()}`).toBe(200);
    return response.json();
  };

  const phone = `08${String(randomInt(0, 1e10)).padStart(10, "0")}`;
  await call("POST", "/auth/register", { name: "Penyewa Rilis Dua", phone_number: phone, password: TENANT_PASSWORD, is_agree_privacy_policy: true });
  const { debug_code } = await call<{ debug_code: string }>("POST", "/auth/otp/request", { for: "tenant_verification", via: "0", destination: phone });
  ({ token } = await call<{ token: string }>("POST", "/auth/otp/check", { for: "tenant_verification", destination: phone, code: debug_code }));

  const { rooms } = await call<{ rooms: { id: number }[] }>("GET", `/listings/${SEEDED_PROPERTY_ID}`);
  const roomId = rooms[0]?.id;
  expect(roomId, "seeded property has a room").toBeDefined();
  const { booking } = await call<{ booking: { id: number } }>("POST", "/booking/submit", { room_id: roomId, rent_type: "monthly", start_date: todayWIB() });
  const { payment } = await call<{ payment: { order_id: string; gross_amount: number } }>("POST", `/booking/${booking.id}/checkout`, {});
  await settle(request, payment);
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

async function expectDownload(page: Page, path: string, contentType: string, filename: RegExp): Promise<Buffer> {
  const response = await page.request.get(path);
  expect(response.status(), `GET ${path}`).toBe(200);
  expect(response.headers()["content-type"]).toContain(contentType);
  expect(response.headers()["content-disposition"]).toMatch(filename);
  return response.body();
}

test("tenant extends and pays, owner records the deposit, admin reviews finance, reconciliation and history", async ({ browser, request }, testInfo) => {
  test.setTimeout(180_000);
  const { phone, bookingId } = await registerTenantWithPaidBooking(request);

  const owner = await newVisitor(browser, testInfo);
  await login(owner, "081200000001", SEED_PASSWORD);
  await owner.goto(`/owner/booking/${bookingId}`);
  await owner.getByRole("button", { name: "Setujui booking" }).click();
  await expect(owner.getByText("Aktif", { exact: true }).first()).toBeVisible();

  const tenant = await newVisitor(browser, testInfo);
  await tenant.route(/\/snap\/snap\.js$/, (route) => route.fulfill({ contentType: "application/javascript", body: SNAP_STUB }));
  await login(tenant, phone, TENANT_PASSWORD);
  await tenant.goto(`/booking/${bookingId}`);
  await expect(tenant.getByRole("definition").filter({ hasText: /^1 bulan$/ })).toBeVisible();

  await tenant.getByRole("button", { name: "Perpanjang sewa" }).click();
  await tenant.getByRole("dialog").getByRole("button", { name: "Lanjutkan" }).click();
  const extensionCard = tenant.locator("article").filter({ hasText: "Perpanjangan" });
  await expect(extensionCard.getByText("Belum dibayar")).toBeVisible();
  await tenant.waitForFunction(() => "snap" in window);

  const checkout = tenant.waitForResponse(/\/api\/invoices\/\d+\/checkout$/);
  await extensionCard.getByRole("button", { name: "Bayar" }).click();
  const { payment } = await (await checkout).json();
  await expect(tenant.getByText("Memeriksa status pembayaran…")).toBeVisible();
  await settle(request, payment);
  await expect(extensionCard.getByText("Lunas")).toBeVisible({ timeout: 15_000 });
  await expect(tenant.getByRole("definition").filter({ hasText: /^2 bulan$/ })).toBeVisible();

  const pdfHref = await extensionCard.getByRole("link", { name: "Unduh PDF" }).getAttribute("href");
  const pdf = await expectDownload(tenant, pdfHref ?? "", "application/pdf", /filename="INV-\d{4}-\d{2}-\d{6}\.pdf"/);
  expect(pdf.subarray(0, 4).toString()).toBe("%PDF");
  await tenant.close();

  await owner.goto("/owner");
  await expect(owner.getByText("Saldo belum ditransfer")).toBeVisible();
  await expect(owner.getByText("Okupansi")).toBeVisible();

  sql(`UPDATE rooms SET available = available + 1 WHERE id = (SELECT room_id FROM bookings WHERE id = ${bookingId} AND stock_held);
       UPDATE bookings SET status = 'completed', ended_at = now(), stock_held = false WHERE id = ${bookingId}`);
  await owner.goto(`/owner/booking/${bookingId}`);
  await owner.getByRole("button", { name: "Catat deposit dikembalikan" }).click();
  await owner.getByRole("dialog").getByLabel("Catatan (opsional)").fill("Transfer BCA, test E2E");
  await owner.getByRole("dialog").getByRole("button", { name: "Catat deposit dikembalikan" }).click();
  await expect(owner.getByText("Deposit dikembalikan", { exact: true })).toBeVisible();

  await owner.goto("/owner/keuangan");
  await expect(owner.getByRole("heading", { name: "Buku besar" })).toBeVisible();
  await owner.getByRole("tab", { name: "Deposit" }).click();
  await expect(owner.getByText("Catatan", { exact: true }).first()).toBeVisible();
  const ownerCsv = await expectDownload(owner, "/api/owner/finance/ledger.csv", "text/csv", /filename="ledger-\d{4}-\d{2}-\d{2}-\d{4}-\d{2}-\d{2}\.csv"/);
  expect(ownerCsv.toString()).toContain("id,occurred_at_wib,type,amount,booking_id,invoice_number,description");
  expect(ownerCsv.toString()).toContain("deposit_returned");

  await owner.getByRole("link", { name: "Lihat riwayat aktivitas" }).click();
  await expect(owner).toHaveURL(/\/owner\/riwayat$/);
  await expect(owner.getByText("booking.deposit_returned").filter({ visible: true }).first()).toBeVisible();
  await owner.close();

  const issueOrderId = sql(`
    INSERT INTO reconciliation_issues (payment_id, kind, local_status, midtrans_status, local_amount, midtrans_amount)
    SELECT id, 'amount_mismatch', status, 'settlement', gross_amount, gross_amount + 1 FROM payments WHERE order_id = '${payment.order_id}'
    RETURNING (SELECT order_id FROM payments WHERE id = payment_id)`).split("\n")[0];
  expect(issueOrderId).toBe(payment.order_id);

  const admin = await newVisitor(browser, testInfo);
  await login(admin, "081200000000", SEED_PASSWORD);
  await admin.goto("/admin/keuangan");
  await expect(admin.getByRole("heading", { name: "Keuangan platform" })).toBeVisible();
  await expect(admin.getByText("Pendapatan biaya admin")).toBeVisible();
  await expect(admin.getByRole("heading", { name: "Buku besar" })).toBeVisible();
  const adminCsv = await expectDownload(admin, "/api/admin/finance/ledger.csv", "text/csv", /filename="ledger-.*\.csv"/);
  expect(adminCsv.toString()).toContain("id,occurred_at_wib,type,amount,owner_id,booking_id,invoice_number,description");

  await admin.goto("/admin/rekonsiliasi");
  await expect(admin.getByRole("heading", { name: "Rekonsiliasi Midtrans" })).toBeVisible();
  const issueRow = admin.locator("tr, li").filter({ hasText: payment.order_id }).filter({ visible: true });
  const loadMore = admin.getByRole("button", { name: "Muat lebih banyak" });
  while (!(await issueRow.count()) && (await loadMore.isVisible())) {
    await loadMore.click();
    await expect(loadMore).toBeEnabled();
  }
  await issueRow.getByRole("button", { name: "Tandai selesai" }).click();
  const dialog = admin.getByRole("dialog");
  await dialog.getByLabel("Catatan").fill("Selisih uji E2E, sudah dicek.");
  await dialog.getByRole("button", { name: "Tandai selesai" }).click();
  await expect(dialog).toBeHidden();
  await expect(issueRow).toHaveCount(0);

  await admin.goto("/admin/riwayat");
  await expect(admin.getByText("reconciliation_issue.resolve").filter({ visible: true }).first()).toBeVisible();
  await expect(admin.getByText("User agent").filter({ visible: true }).first()).toBeVisible();
  await admin.close();
});
