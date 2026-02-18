"use client";

import type { Parcel } from "@/lib/types";
import { ParcelStatusBadge } from "./ParcelStatusBadge";

interface ParcelCardProps {
  parcel: Parcel;
  isSelected: boolean;
  onClick: () => void;
}

export function ParcelCard({ parcel, isSelected, onClick }: ParcelCardProps) {
  return (
    <button
      onClick={onClick}
      className={`w-full text-left p-3 rounded-lg border transition-colors ${
        isSelected
          ? "border-emerald-500 bg-emerald-50"
          : "border-gray-200 bg-white hover:border-gray-300"
      }`}
    >
      <div className="flex items-start justify-between gap-2">
        <h3 className="text-sm font-medium text-gray-900 truncate">
          {parcel.label || "Untitled Parcel"}
        </h3>
        <ParcelStatusBadge status={parcel.status} />
      </div>
      <p className="text-xs text-gray-500 mt-1">
        {parcel.district}, {parcel.state}
      </p>
      {parcel.area_sqm && (
        <p className="text-xs text-gray-400 mt-0.5">
          {(parcel.area_sqm / 4046.86).toFixed(2)} acres
        </p>
      )}
    </button>
  );
}
