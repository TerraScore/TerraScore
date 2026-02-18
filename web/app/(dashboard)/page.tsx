"use client";

import { useState, useCallback } from "react";
import dynamic from "next/dynamic";
import Link from "next/link";
import { useParcels } from "@/hooks/useParcels";
import { useParcel } from "@/hooks/useParcel";
import { ParcelList } from "@/components/parcels/ParcelList";
import { Button } from "@/components/ui/Button";
import { ErrorMessage } from "@/components/ui/ErrorMessage";
import type { Parcel } from "@/lib/types";

const DashboardMap = dynamic(
  () => import("@/components/map/DashboardMap").then((m) => m.DashboardMap),
  { ssr: false }
);

export default function DashboardPage() {
  const [page] = useState(1);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const { data, isLoading, error, refetch } = useParcels(page);

  const parcels: Parcel[] = data?.data || [];
  const { data: parcelDetail } = useParcel(selectedId);
  const selectedParcel = parcelDetail?.data || null;

  const handleMarkerClick = useCallback((id: string) => {
    setSelectedId(id);
  }, []);

  return (
    <div className="flex h-full">
      {/* Sidebar parcel list */}
      <div className="w-72 border-r border-gray-200 bg-white flex flex-col">
        <div className="p-4 border-b border-gray-200 flex items-center justify-between">
          <h2 className="font-semibold text-gray-900">My Parcels</h2>
          <Link href="/parcels/new">
            <Button size="sm">+ New Parcel</Button>
          </Link>
        </div>

        {error ? (
          <div className="p-4">
            <ErrorMessage
              message="Failed to load parcels"
              onRetry={() => refetch()}
            />
          </div>
        ) : (
          <div className="flex-1 overflow-y-auto">
            <ParcelList
              parcels={parcels}
              selectedId={selectedId}
              onSelect={setSelectedId}
              isLoading={isLoading}
            />
          </div>
        )}

        {data?.meta && data.meta.total_pages > 1 && (
          <div className="p-3 border-t border-gray-200 text-center text-xs text-gray-500">
            Page {data.meta.page} of {data.meta.total_pages} ({data.meta.total} parcels)
          </div>
        )}
      </div>

      {/* Map area */}
      <div className="flex-1 relative">
        <DashboardMap
          parcels={parcels}
          selectedParcel={selectedParcel}
          onMarkerClick={handleMarkerClick}
        />

        {/* Selected parcel info bar */}
        {selectedParcel && (
          <div className="absolute bottom-4 left-4 right-4 bg-white rounded-lg shadow-lg border border-gray-200 p-4 flex items-center justify-between">
            <div>
              <h3 className="font-medium text-gray-900">
                {selectedParcel.label || "Untitled Parcel"}
              </h3>
              <p className="text-sm text-gray-500">
                {selectedParcel.district}, {selectedParcel.state}
              </p>
            </div>
            <Link href={`/parcels/${selectedParcel.id}`}>
              <Button size="sm" variant="secondary">View Details</Button>
            </Link>
          </div>
        )}
      </div>
    </div>
  );
}
