import { NextRequest, NextResponse } from "next/server";
import { apiFetch } from "@/lib/api";

export async function GET(
  _request: NextRequest,
  { params }: { params: { id: string } }
) {
  const { status, body } = await apiFetch(`/v1/parcels/${params.id}`);
  return NextResponse.json(body, { status });
}

export async function DELETE(
  _request: NextRequest,
  { params }: { params: { id: string } }
) {
  const { status, body } = await apiFetch(`/v1/parcels/${params.id}`, {
    method: "DELETE",
  });
  return NextResponse.json(body, { status });
}
