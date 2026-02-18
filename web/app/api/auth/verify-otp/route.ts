import { NextRequest, NextResponse } from "next/server";
import { apiFetch } from "@/lib/api";
import { setAuthCookies } from "@/lib/auth";
import type { ApiResponse, VerifyOTPResponse } from "@/lib/types";

export async function POST(request: NextRequest) {
  const body = await request.json();
  const { status, body: data } = await apiFetch<ApiResponse<VerifyOTPResponse>>(
    "/v1/auth/verify-otp",
    {
      method: "POST",
      body: JSON.stringify(body),
    }
  );

  if (status === 200 && data.data) {
    setAuthCookies(data.data.access_token, data.data.refresh_token, data.data.expires_in);
    return NextResponse.json({ ok: true });
  }

  return NextResponse.json(data, { status });
}
