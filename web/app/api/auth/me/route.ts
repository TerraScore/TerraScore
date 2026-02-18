import { NextResponse } from "next/server";
import { getServerToken } from "@/lib/auth";
import type { UserProfile } from "@/lib/types";

export async function GET() {
  const token = getServerToken();
  if (!token) {
    return NextResponse.json({ error: { message: "Not authenticated" } }, { status: 401 });
  }

  try {
    // Decode JWT payload (no verification — Kong already validated it)
    const parts = token.split(".");
    if (parts.length !== 3) {
      return NextResponse.json({ error: { message: "Invalid token" } }, { status: 401 });
    }

    const payload = JSON.parse(Buffer.from(parts[1], "base64url").toString());
    const profile: UserProfile = {
      sub: payload.sub || "",
      phone: payload.preferred_username || payload.phone_number || "",
      name: payload.given_name || payload.name || "",
      role: payload.realm_access?.roles?.includes("landowner")
        ? "landowner"
        : payload.realm_access?.roles?.includes("agent")
          ? "agent"
          : "landowner",
    };

    return NextResponse.json({ profile });
  } catch {
    return NextResponse.json({ error: { message: "Invalid token" } }, { status: 401 });
  }
}
