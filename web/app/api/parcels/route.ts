import { NextRequest, NextResponse } from "next/server";
import { apiFetch } from "@/lib/api";

export async function GET(request: NextRequest) {
  const { searchParams } = new URL(request.url);
  const page = searchParams.get("page") || "1";
  const perPage = searchParams.get("per_page") || "20";

  const { status, body } = await apiFetch(`/v1/parcels?page=${page}&per_page=${perPage}`);
  return NextResponse.json(body, { status });
}

export async function POST(request: NextRequest) {
  const body = await request.json();
  const { status, body: data } = await apiFetch("/v1/parcels", {
    method: "POST",
    body: JSON.stringify(body),
  });
  return NextResponse.json(data, { status });
}
