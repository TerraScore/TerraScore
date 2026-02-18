import { NextResponse } from "next/server";
import { getServerRefreshToken, setAuthCookies, clearAuthCookies } from "@/lib/auth";
import { apiFetch } from "@/lib/api";
import type { ApiResponse, VerifyOTPResponse } from "@/lib/types";

export async function POST() {
  const refreshToken = getServerRefreshToken();
  if (!refreshToken) {
    return NextResponse.json({ error: { message: "No refresh token" } }, { status: 401 });
  }

  const { status, body: data } = await apiFetch<ApiResponse<VerifyOTPResponse>>(
    "/v1/auth/refresh",
    {
      method: "POST",
      body: JSON.stringify({ refresh_token: refreshToken }),
    }
  );

  if (status === 200 && data.data) {
    setAuthCookies(data.data.access_token, data.data.refresh_token, data.data.expires_in);
    return NextResponse.json({ ok: true });
  }

  clearAuthCookies();
  return NextResponse.json(data, { status: 401 });
}
