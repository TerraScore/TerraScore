"use client";

import type { Parcel } from "@/lib/types";
import { ParcelCard } from "./ParcelCard";

interface ParcelListProps {
  parcels: Parcel[];
  selectedId: string | null;
  onSelect: (id: string) => void;
  isLoading: boolean;
}

export function ParcelList({ parcels, selectedId, onSelect, isLoading }: ParcelListProps) {
  if (isLoading) {
    return (
      <div className="space-y-3 p-4">
        {Array.from({ length: 4 }).map((_, i) => (
          <div key={i} className="animate-pulse">
            <div className="h-20 bg-gray-200 rounded-lg" />
          </div>
        ))}
      </div>
    );
  }

  if (parcels.length === 0) {
    return (
      <div className="p-6 text-center">
        <svg className="w-12 h-12 text-gray-300 mx-auto mb-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1} d="M9 20l-5.447-2.724A1 1 0 013 16.382V5.618a1 1 0 011.447-.894L9 7m0 13l6-3m-6 3V7m6 10l4.553 2.276A1 1 0 0021 18.382V7.618a1 1 0 00-.553-.894L15 4m0 13V4m0 0L9 7" />
        </svg>
        <p className="text-sm font-medium text-gray-900">No parcels yet</p>
        <p className="text-xs text-gray-500 mt-1">Register your first parcel to get started</p>
      </div>
    );
  }

  return (
    <div className="space-y-2 p-3">
      {parcels.map((parcel) => (
        <ParcelCard
          key={parcel.id}
          parcel={parcel}
          isSelected={selectedId === parcel.id}
          onClick={() => onSelect(parcel.id)}
        />
      ))}
    </div>
  );
}
