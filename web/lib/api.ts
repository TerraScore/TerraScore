import { getServerToken } from "./auth";

const API_URL = process.env.API_URL || "http://localhost:8000";

/**
 * Server-side fetch that injects the Bearer token from httpOnly cookie.
 * Used in Route Handlers to proxy requests to Kong → Go backend.
 */
export async function apiFetch<T>(
  path: string,
  opts: RequestInit = {}
): Promise<{ status: number; body: T }> {
  const token = getServerToken();
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...(opts.headers as Record<string, string>),
  };

  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }

  const res = await fetch(`${API_URL}${path}`, {
    ...opts,
    headers,
    cache: "no-store",
  });

  // For 204 No Content
  if (res.status === 204) {
    return { status: res.status, body: {} as T };
  }

  const body = await res.json();
  return { status: res.status, body };
}
