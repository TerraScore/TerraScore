import { NextRequest, NextResponse } from "next/server";
import { apiFetch } from "@/lib/api";

export async function GET(
  _request: NextRequest,
  { params }: { params: { id: string } }
) {
  const { status, body: data } = await apiFetch(`/v1/parcels/${params.id}/surveys`, {
    method: "GET",
  });
  return NextResponse.json(data, { status });
}
