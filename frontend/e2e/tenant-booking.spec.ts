import { createHash } from "node:crypto";
import { expect, test } from "@playwright/test";

const API_URL = process.env.E2E_API_URL ?? "http://127.0.0.1:8121/api";
const MIDTRANS_SERVER_KEY = process.env.E2E_MIDTRANS_SERVER_KEY ?? "e2e-server-key";
const SNAP_STUB = "window.snap = { pay: function (token, callbacks) { callbacks.onPending(); } };";

function randomPhone(): string {
  return `08${String(Math.floor(Math.random() * 1e10)).padStart(10, "0")}`;
}

function signature(orderId: string, statusCode: string, grossAmount: string): string {
  return createHash("sha512").update(`${orderId}${statusCode}${grossAmount}${MIDTRANS_SERVER_KEY}`).digest("hex");
}

test("tenant registers, rents a room, and sees it paid after the Midtrans webhook", async ({ page, request }) => {
  await page.route(/\/snap\/snap\.js$/, (route) => route.fulfill({ contentType: "application/javascript", body: SNAP_STUB }));

  await page.goto("/daftar");
  await page.getByLabel("Nama", { exact: true }).fill("Penyewa E2E");
  await page.getByLabel("Nomor HP").fill(randomPhone());
  await page.getByLabel("Password", { exact: true }).fill("Rahasia123");
  await page.getByRole("checkbox").check();
  await page.getByRole("button", { name: "Daftar", exact: true }).click();

  const otpRequest = page.waitForResponse("**/api/auth/otp/request");
  await page.getByRole("button", { name: "Kirim kode" }).click();
  const { debug_code: otpCode } = await (await otpRequest).json();
  await page.getByLabel("Kode verifikasi").fill(otpCode);
  await expect(page.getByRole("heading", { name: "Cari kost yang pas" })).toBeVisible();

  await page.getByRole("button", { name: "Kota Depok" }).first().click();
  await expect(page).toHaveURL(/\/cari\?area_id=\d+/);
  await page.locator('a[href^="/kost/"]').first().click();

  await page.getByRole("button", { name: "Ajukan sewa" }).click();
  await page.getByRole("button", { name: "Lanjutkan" }).click();
  await expect(page).toHaveURL(/\/kost\/\d+\/sewa\?room=\d+$/);
  await expect(page.getByText("Total bayar sekarang")).toBeVisible();
  await page.getByRole("button", { name: "Ajukan sewa" }).click();

  await expect(page).toHaveURL(/\/booking\/\d+$/);
  await expect(page.getByText("Menunggu pembayaran").first()).toBeVisible();
  await page.waitForFunction(() => "snap" in window);

  const checkout = page.waitForResponse(/\/api\/booking\/\d+\/checkout$/);
  await page.getByRole("button", { name: "Bayar sekarang" }).click();
  const { payment } = await (await checkout).json();
  await expect(page.getByText("Memeriksa status pembayaran…")).toBeVisible();

  const grossAmount = `${payment.gross_amount}.00`;
  const webhook = await request.post(`${API_URL}/payments/midtrans/notification`, {
    data: {
      order_id: payment.order_id,
      status_code: "200",
      gross_amount: grossAmount,
      transaction_status: "settlement",
      fraud_status: "accept",
      signature_key: signature(payment.order_id, "200", grossAmount),
    },
  });
  expect(webhook.status()).toBe(200);

  await expect(page.getByText("Menunggu konfirmasi pemilik")).toBeVisible({ timeout: 15_000 });
});
