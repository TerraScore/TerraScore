import { NextRequest, NextResponse } from "next/server";
import { apiFetch } from "@/lib/api";

export async function POST(
  _request: NextRequest,
  { params }: { params: { id: string } }
) {
  const { status, body: data } = await apiFetch(`/v1/parcels/${params.id}/request-survey`, {
    method: "POST",
  });
  return NextResponse.json(data, { status });
}
