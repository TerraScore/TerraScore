import { NextRequest, NextResponse } from "next/server";
import { apiFetch } from "@/lib/api";

export async function PUT(
  request: NextRequest,
  { params }: { params: { id: string } }
) {
  const body = await request.json();
  const { status, body: data } = await apiFetch(`/v1/parcels/${params.id}/boundary`, {
    method: "PUT",
    body: JSON.stringify(body),
  });
  return NextResponse.json(data, { status });
}
