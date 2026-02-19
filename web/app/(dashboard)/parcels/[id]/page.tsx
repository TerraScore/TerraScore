"use client";

import { useState, useCallback } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import dynamic from "next/dynamic";
import { useParcel } from "@/hooks/useParcel";
import { useUpdateBoundary } from "@/hooks/useUpdateBoundary";
import { useDeleteParcel } from "@/hooks/useDeleteParcel";
import { ParcelStatusBadge } from "@/components/parcels/ParcelStatusBadge";
import { Button } from "@/components/ui/Button";
import { Spinner } from "@/components/ui/Spinner";
import { ErrorMessage } from "@/components/ui/ErrorMessage";

const ParcelMap = dynamic(
  () => import("@/components/map/ParcelMap").then((m) => m.ParcelMap),
  { ssr: false }
);
const DrawMap = dynamic(
  () => import("@/components/map/DrawMap").then((m) => m.DrawMap),
  { ssr: false }
);

export default function ParcelDetailPage({ params }: { params: { id: string } }) {
  const router = useRouter();
  const { data, isLoading, error, refetch } = useParcel(params.id);
  const updateBoundary = useUpdateBoundary(params.id);
  const deleteParcel = useDeleteParcel();

  const [isEditing, setIsEditing] = useState(false);
  const [newBoundary, setNewBoundary] = useState<string | null>(null);
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);

  const parcel = data?.data;

  const handleBoundaryChange = useCallback(
    (_geometry: GeoJSON.Geometry | null, geoJSON: string | null) => {
      setNewBoundary(geoJSON);
    },
    []
  );

  async function handleSaveBoundary() {
    if (!newBoundary) return;
    await updateBoundary.mutateAsync({ boundary: newBoundary });
    setIsEditing(false);
    setNewBoundary(null);
    refetch();
  }

  async function handleDelete() {
    await deleteParcel.mutateAsync(params.id);
    router.push("/");
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <Spinner className="h-8 w-8" />
      </div>
    );
  }

  if (error || !parcel) {
    return (
      <div className="max-w-3xl mx-auto p-6">
        <ErrorMessage
          message={error?.message || "Parcel not found"}
          onRetry={() => refetch()}
        />
      </div>
    );
  }

  const infoFields = [
    { label: "Survey Number", value: parcel.survey_number },
    { label: "Village", value: parcel.village },
    { label: "Taluk", value: parcel.taluk },
    { label: "District", value: parcel.district },
    { label: "State", value: parcel.state },
    { label: "PIN Code", value: parcel.pin_code },
    { label: "Land Type", value: parcel.land_type },
    { label: "Registered Area", value: parcel.registered_area_sqm ? `${parcel.registered_area_sqm} sq m` : null },
    { label: "Computed Area", value: parcel.area_sqm ? `${parcel.area_sqm.toFixed(0)} sq m (${(parcel.area_sqm / 4046.86).toFixed(2)} acres)` : null },
  ].filter((f) => f.value);

  return (
    <div className="p-6">
      <div className="mb-4">
        <Link href="/" className="text-sm text-gray-500 hover:text-gray-700">&larr; Dashboard</Link>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Left: Info */}
        <div className="bg-white rounded-xl border border-gray-200 p-6">
          <div className="flex items-start justify-between mb-4">
            <div>
              <h1 className="text-lg font-bold text-gray-900">
                {parcel.label || "Untitled Parcel"}
              </h1>
              <p className="text-sm text-gray-500">{parcel.district}, {parcel.state}</p>
            </div>
            <ParcelStatusBadge status={parcel.status} />
          </div>

          <div className="space-y-3 mb-6">
            {infoFields.map((f) => (
              <div key={f.label} className="flex justify-between text-sm">
                <span className="text-gray-500">{f.label}</span>
                <span className="font-medium text-gray-900">{f.value}</span>
              </div>
            ))}
          </div>

          {/* Sub-navigation */}
          <div className="border-t border-gray-200 pt-4 mb-4">
            <h3 className="text-xs font-semibold text-gray-400 uppercase tracking-wider mb-2">Parcel Actions</h3>
            <div className="space-y-1">
              <Link
                href={`/parcels/${params.id}/tracking`}
                className="block px-3 py-2 text-sm text-gray-600 hover:bg-gray-50 rounded-lg"
              >
                Live Tracking
              </Link>
              <Link
                href={`/parcels/${params.id}/surveys`}
                className="block px-3 py-2 text-sm text-gray-600 hover:bg-gray-50 rounded-lg"
              >
                Surveys
              </Link>
              <Link
                href={`/parcels/${params.id}/reports`}
                className="block px-3 py-2 text-sm text-gray-600 hover:bg-gray-50 rounded-lg"
              >
                Reports
              </Link>
            </div>
          </div>

          {/* Boundary edit toggle */}
          <div className="border-t border-gray-200 pt-4 space-y-2">
            {isEditing ? (
              <div className="flex gap-2">
                <Button
                  size="sm"
                  onClick={handleSaveBoundary}
                  loading={updateBoundary.isPending}
                  disabled={!newBoundary}
                >
                  Save Boundary
                </Button>
                <Button
                  size="sm"
                  variant="secondary"
                  onClick={() => { setIsEditing(false); setNewBoundary(null); }}
                >
                  Cancel
                </Button>
              </div>
            ) : (
              <Button size="sm" variant="secondary" onClick={() => setIsEditing(true)}>
                Edit Boundary
              </Button>
            )}

            {updateBoundary.error && (
              <p className="text-sm text-red-600">{updateBoundary.error.message}</p>
            )}
          </div>

          {/* Delete */}
          <div className="border-t border-gray-200 pt-4 mt-4">
            {showDeleteConfirm ? (
              <div className="bg-red-50 border border-red-200 rounded-lg p-3">
                <p className="text-sm text-red-800 mb-2">Are you sure? This cannot be undone.</p>
                <div className="flex gap-2">
                  <Button
                    size="sm"
                    variant="danger"
                    onClick={handleDelete}
                    loading={deleteParcel.isPending}
                  >
                    Delete
                  </Button>
                  <Button
                    size="sm"
                    variant="secondary"
                    onClick={() => setShowDeleteConfirm(false)}
                  >
                    Cancel
                  </Button>
                </div>
              </div>
            ) : (
              <Button size="sm" variant="ghost" onClick={() => setShowDeleteConfirm(true)}>
                <span className="text-red-600">Delete Parcel</span>
              </Button>
            )}
          </div>
        </div>

        {/* Right: Map */}
        <div className="bg-white rounded-xl border border-gray-200 overflow-hidden h-[600px]">
          {isEditing ? (
            <DrawMap
              initialGeometry={typeof parcel.boundary_geojson === "string" ? JSON.parse(parcel.boundary_geojson) : parcel.boundary_geojson}
              onBoundaryChange={handleBoundaryChange}
            />
          ) : parcel.boundary_geojson ? (
            <ParcelMap boundary={typeof parcel.boundary_geojson === "string" ? JSON.parse(parcel.boundary_geojson) : parcel.boundary_geojson} />
          ) : (
            <div className="flex items-center justify-center h-full text-gray-400">
              <p>No boundary defined</p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
