import { randomInt } from "node:crypto";
import { expect, test, type APIRequestContext, type Page } from "@playwright/test";

const API_URL = process.env.E2E_API_URL ?? "http://127.0.0.1:8121/api";
const API_CLIENT_ID = process.env.API_CLIENT_ID ?? "KOSAPI WEB:WEB";
const PASSWORD = "Rahasia123";

function visitorIP(): string {
  return `100.${randomInt(64, 127)}.${randomInt(0, 255)}.${randomInt(1, 254)}`;
}

function randomPhone(): string {
  return `08${String(randomInt(0, 1e10)).padStart(10, "0")}`;
}

const MAX_TEST_COOLDOWN_SECONDS = 10;

async function createVerifiedUser(request: APIRequestContext, ip: string): Promise<{ phone: string; cooldownSeconds: number }> {
  const phone = randomPhone();
  const headers = { "x-api-client": API_CLIENT_ID, "X-Forwarded-For": ip };
  const register = await request.post(`${API_URL}/auth/register`, {
    headers,
    data: { name: "Penyewa OTP", phone_number: phone, password: PASSWORD, is_agree_privacy_policy: true },
  });
  expect(register.status(), await register.text()).toBe(200);
  const otp = await request.post(`${API_URL}/auth/otp/request`, {
    headers,
    data: { for: "tenant_verification", destination: phone },
  });
  expect(otp.status(), await otp.text()).toBe(200);
  const { debug_code, verification_code } = await otp.json();
  const check = await request.post(`${API_URL}/auth/otp/check`, {
    headers,
    data: { for: "tenant_verification", destination: phone, code: debug_code },
  });
  expect(check.status(), await check.text()).toBe(200);
  return { phone, cooldownSeconds: verification_code.can_resend_after_seconds };
}

async function waitOutCooldown(page: Page, cooldownSeconds: number) {
  test.skip(
    cooldownSeconds > MAX_TEST_COOLDOWN_SECONDS,
    `Needs the backend under test to run with OTP_RESEND_AFTER_SECONDS <= ${MAX_TEST_COOLDOWN_SECONDS}`,
  );
  await page.waitForTimeout(cooldownSeconds * 1000 + 500);
}

function trackOtpRequests(page: Page): string[] {
  const bodies: string[] = [];
  page.on("request", (request) => {
    if (request.method() === "POST" && request.url().endsWith("/api/auth/otp/request")) bodies.push(request.postData() ?? "");
  });
  return bodies;
}

async function expectNoProviderOrSmsMention(page: Page) {
  await expect(page.locator("body")).not.toContainText(/fazpass/i);
  await expect(page.getByRole("radio", { name: /SMS|WhatsApp/ })).toHaveCount(0);
  await expect(page.getByText("SMS", { exact: true })).toHaveCount(0);
}

test.describe("OTP over WhatsApp", () => {
  let ip: string;

  test.beforeEach(async ({ page }) => {
    ip = visitorIP();
    await page.setExtraHTTPHeaders({ "X-Forwarded-For": ip });
  });

  test("register offers WhatsApp only, sends once on a double click, and verifies the code", async ({ page }) => {
    const phone = randomPhone();
    await page.goto("/daftar");
    await page.getByLabel("Nama", { exact: true }).fill("Penyewa OTP E2E");
    await page.getByLabel("Nomor HP").fill(phone);
    await page.getByLabel("Password", { exact: true }).fill(PASSWORD);
    await page.getByRole("checkbox").check();
    await page.getByRole("button", { name: "Daftar", exact: true }).click();

    await expect(page.getByText(`Kode verifikasi akan dikirim lewat WhatsApp ke ${phone}.`)).toBeVisible();
    await expectNoProviderOrSmsMention(page);

    const sent = trackOtpRequests(page);
    const otpResponse = page.waitForResponse("**/api/auth/otp/request");
    await page.getByRole("button", { name: "Kirim kode lewat WhatsApp" }).dblclick();
    const { debug_code: code } = await (await otpResponse).json();

    await expect(page.getByText(`Kode dikirim lewat WhatsApp ke ${phone}.`)).toBeVisible();
    await expectNoProviderOrSmsMention(page);
    expect(sent).toHaveLength(1);
    expect(JSON.parse(sent[0] ?? "{}").via).toBe("1");

    await page.getByLabel("Kode verifikasi").fill(code);
    await expect(page).not.toHaveURL(/\/daftar/, { timeout: 15_000 });
  });

  test("login by OTP caps resends at three and still logs in with the latest code", async ({ page, request }) => {
    const { phone, cooldownSeconds } = await createVerifiedUser(request, ip);
    await waitOutCooldown(page, cooldownSeconds);
    await page.goto("/masuk");
    await page.getByRole("radio", { name: "Kode OTP" }).click();
    await page.getByLabel("Nomor HP").fill(phone);
    await page.getByRole("button", { name: "Lanjut" }).click();
    await expectNoProviderOrSmsMention(page);

    const sent = trackOtpRequests(page);
    let response = page.waitForResponse("**/api/auth/otp/request");
    await page.getByRole("button", { name: "Kirim kode lewat WhatsApp" }).click();
    let body = await (await response).json();

    for (const left of [2, 1, 0]) {
      const resend = page.getByRole("button", { name: "Kirim ulang kode" });
      await expect(resend).toBeVisible({ timeout: 15_000 });
      response = page.waitForResponse("**/api/auth/otp/request");
      await resend.click();
      body = await (await response).json();
      if (left > 0) {
        await expect(page.getByText(`Sisa ${left} kali kirim ulang`)).toBeVisible({ timeout: 15_000 });
      }
    }

    await expect(page.getByText("Batas kirim ulang kode tercapai. Silakan coba lagi nanti.")).toBeVisible({ timeout: 15_000 });
    await expect(page.getByRole("button", { name: "Kirim ulang kode" })).toHaveCount(0);
    expect(sent).toHaveLength(4);

    await page.getByLabel("Kode verifikasi").fill(body.debug_code);
    await expect(page).not.toHaveURL(/\/masuk/, { timeout: 15_000 });
  });

  test("reset password with a WhatsApp code, then log in with the new password", async ({ page, request }) => {
    const { phone, cooldownSeconds } = await createVerifiedUser(request, ip);
    await waitOutCooldown(page, cooldownSeconds);
    await page.goto("/lupa-password");
    await page.getByLabel("Nomor HP").fill(phone);
    await page.getByRole("button", { name: "Lanjut" }).click();
    await expect(page.getByText(`Kode verifikasi akan dikirim lewat WhatsApp ke ${phone}.`)).toBeVisible();
    await expectNoProviderOrSmsMention(page);

    const response = page.waitForResponse("**/api/auth/otp/request");
    await page.getByRole("button", { name: "Kirim kode lewat WhatsApp" }).click();
    const { debug_code: code } = await (await response).json();

    await page.getByLabel("Kode verifikasi").fill(code);
    await page.getByLabel("Password baru").fill("RahasiaBaru123");
    await page.getByRole("button", { name: "Simpan password baru" }).click();
    await expect(page).toHaveURL(/\/masuk/, { timeout: 15_000 });

    await page.getByLabel("Nomor HP").fill(phone);
    await page.getByLabel("Password", { exact: true }).fill("RahasiaBaru123");
    await page.getByRole("button", { name: "Masuk", exact: true }).click();
    await expect(page).not.toHaveURL(/\/masuk/, { timeout: 15_000 });
  });

  test("a 429 from the server stops further sends on the page", async ({ page }) => {
    const limitMessage = "Batas pengiriman kode untuk nomor ini sudah tercapai. Silakan coba lagi nanti.";
    let calls = 0;
    await page.route("**/api/auth/otp/request", async (route) => {
      calls += 1;
      await route.fulfill({
        status: 429,
        contentType: "application/json",
        body: JSON.stringify({
          status: false,
          meta: { response_code: 429, code: 429, severity: "WARNING", message: limitMessage },
          messages: null,
        }),
      });
    });

    await page.goto("/lupa-password");
    await page.getByLabel("Nomor HP").fill(randomPhone());
    await page.getByRole("button", { name: "Lanjut" }).click();
    const send = page.getByRole("button", { name: "Kirim kode lewat WhatsApp" });
    await send.click();
    await expect(page.getByText(limitMessage).first()).toBeVisible();
    await send.click();
    await send.click();

    expect(calls).toBe(1);
  });
});
