import { NextRequest, NextResponse } from "next/server";
import { apiFetch } from "@/lib/api";

export async function PUT(
  request: NextRequest,
  { params }: { params: { id: string } }
) {
  const { status, body } = await apiFetch(`/v1/alerts/${params.id}/read`, {
    method: "PUT",
  });
  return NextResponse.json(body, { status });
}
