import { NextRequest, NextResponse } from "next/server";
import { apiFetch } from "@/lib/api";

export async function GET(request: NextRequest) {
  const { searchParams } = new URL(request.url);
  const page = searchParams.get("page") || "1";
  const perPage = searchParams.get("per_page") || "20";

  const { status, body } = await apiFetch(`/v1/alerts?page=${page}&per_page=${perPage}`);
  return NextResponse.json(body, { status });
}
