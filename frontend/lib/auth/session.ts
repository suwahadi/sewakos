import { cookies } from "next/headers";

const SESSION_COOKIE_NAME = "sewakos_session";
const ROLE_COOKIE_NAME = "sewakos_role";
// SPEC-GO.md §6.1: access tokens never expire server-side; this is just how long the
// browser keeps offering the cookie before the user would need to log in again.
const SESSION_COOKIE_MAX_AGE_SECONDS = 60 * 60 * 24 * 30;

export async function getSessionToken(): Promise<string | null> {
  const store = await cookies();
  return store.get(SESSION_COOKIE_NAME)?.value ?? null;
}

export async function getSessionRole(): Promise<string | null> {
  const store = await cookies();
  return store.get(ROLE_COOKIE_NAME)?.value ?? null;
}

export async function setSessionToken(token: string, role?: string): Promise<void> {
  const store = await cookies();
  const options = {
    httpOnly: true,
    secure: process.env.NODE_ENV === "production",
    sameSite: "lax" as const,
    path: "/",
    maxAge: SESSION_COOKIE_MAX_AGE_SECONDS,
  };
  store.set(SESSION_COOKIE_NAME, token, options);
  if (role) store.set(ROLE_COOKIE_NAME, role, options);
}

export async function clearSessionToken(): Promise<void> {
  const store = await cookies();
  store.delete(SESSION_COOKIE_NAME);
  store.delete(ROLE_COOKIE_NAME);
}
