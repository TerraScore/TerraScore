"use client";

import { useAuthStore } from "@/stores/authStore";

export function TopBar() {
  const { profile } = useAuthStore();

  return (
    <header className="h-14 bg-white border-b border-gray-200 flex items-center justify-between px-6">
      <div />
      <div className="flex items-center gap-3">
        <div className="text-right">
          <p className="text-sm font-medium text-gray-900">{profile?.name || "User"}</p>
          <p className="text-xs text-gray-500">{profile?.role || "landowner"}</p>
        </div>
        <div className="w-8 h-8 rounded-full bg-emerald-100 flex items-center justify-center">
          <span className="text-sm font-medium text-emerald-700">
            {(profile?.name || "U").charAt(0).toUpperCase()}
          </span>
        </div>
      </div>
    </header>
  );
}
