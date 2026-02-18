import { cookies } from "next/headers";

const ACCESS_COOKIE = "li_access";
const REFRESH_COOKIE = "li_refresh";

const COOKIE_OPTIONS = {
  httpOnly: true,
  secure: process.env.NODE_ENV === "production",
  sameSite: "lax" as const,
  path: "/",
};

export function setAuthCookies(accessToken: string, refreshToken: string, expiresIn: number) {
  const cookieStore = cookies();
  cookieStore.set(ACCESS_COOKIE, accessToken, {
    ...COOKIE_OPTIONS,
    maxAge: expiresIn,
  });
  cookieStore.set(REFRESH_COOKIE, refreshToken, {
    ...COOKIE_OPTIONS,
    maxAge: 30 * 24 * 60 * 60, // 30 days
  });
}

export function clearAuthCookies() {
  const cookieStore = cookies();
  cookieStore.delete(ACCESS_COOKIE);
  cookieStore.delete(REFRESH_COOKIE);
}

export function getServerToken(): string | undefined {
  const cookieStore = cookies();
  return cookieStore.get(ACCESS_COOKIE)?.value;
}

export function getServerRefreshToken(): string | undefined {
  const cookieStore = cookies();
  return cookieStore.get(REFRESH_COOKIE)?.value;
}
