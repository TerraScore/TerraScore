"use client";

const statusConfig: Record<string, { bg: string; text: string; label: string }> = {
  registered: { bg: "bg-blue-50", text: "text-blue-700", label: "Registered" },
  verified: { bg: "bg-emerald-50", text: "text-emerald-700", label: "Verified" },
  pending_verification: { bg: "bg-yellow-50", text: "text-yellow-700", label: "Pending" },
  disputed: { bg: "bg-red-50", text: "text-red-700", label: "Disputed" },
};

export function ParcelStatusBadge({ status }: { status: string | null }) {
  const config = statusConfig[status || "registered"] || statusConfig.registered;
  return (
    <span className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium ${config.bg} ${config.text}`}>
      {config.label}
    </span>
  );
}
