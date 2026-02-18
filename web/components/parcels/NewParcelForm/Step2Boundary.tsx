"use client";

import { useMemo } from "react";
import dynamic from "next/dynamic";
import { Button } from "@/components/ui/Button";

const DrawMap = dynamic(
  () => import("@/components/map/DrawMap").then((m) => m.DrawMap),
  { ssr: false }
);

interface Step2Props {
  geometry: GeoJSON.Geometry | null;
  onBoundaryChange: (geometry: GeoJSON.Geometry | null, geoJSON: string | null) => void;
  onBack: () => void;
  onNext: () => void;
  canProceed: boolean;
}

function computeAreaSqm(geometry: GeoJSON.Geometry): number | null {
  try {
    // Simplified area calculation using the Shoelace formula for projected coordinates
    // @mapbox/geojson-area uses proper geodesic calculation
    // Since the package might not have proper TS types, we use dynamic import
    if (geometry.type !== "Polygon") return null;
    const coords = geometry.coordinates[0];
    if (!coords || coords.length < 4) return null;

    // Use approximate conversion: 1 degree ≈ 111,320 meters at equator
    // This is display-only; PostGIS is authoritative for the actual area
    let area = 0;
    for (let i = 0; i < coords.length - 1; i++) {
      const [x1, y1] = coords[i];
      const [x2, y2] = coords[i + 1];
      area += x1 * y2 - x2 * y1;
    }
    area = Math.abs(area) / 2;
    // Convert from degree² to m² (rough approximation)
    const avgLat = coords.reduce((sum, c) => sum + c[1], 0) / coords.length;
    const metersPerDegreeLng = 111320 * Math.cos((avgLat * Math.PI) / 180);
    const metersPerDegreeLat = 110540;
    return area * metersPerDegreeLng * metersPerDegreeLat;
  } catch {
    return null;
  }
}

export function Step2Boundary({ geometry, onBoundaryChange, onBack, onNext, canProceed }: Step2Props) {
  const areaSqm = useMemo(() => {
    if (!geometry) return null;
    return computeAreaSqm(geometry);
  }, [geometry]);

  return (
    <div className="space-y-4">
      <p className="text-sm text-gray-600">
        Draw the boundary of your parcel on the satellite map. Use the polygon tool in the top-left corner.
      </p>

      <div className="h-[500px] rounded-lg overflow-hidden border border-gray-200">
        <DrawMap onBoundaryChange={onBoundaryChange} initialGeometry={geometry} />
      </div>

      {areaSqm !== null && (
        <div className="bg-emerald-50 border border-emerald-200 rounded-lg p-3">
          <p className="text-sm text-emerald-800">
            Estimated area: <span className="font-semibold">{areaSqm.toFixed(0)} sq m</span>
            {" "}({(areaSqm / 4046.86).toFixed(2)} acres)
          </p>
          <p className="text-xs text-emerald-600 mt-0.5">
            Display estimate only — exact area calculated server-side by PostGIS.
          </p>
        </div>
      )}

      {!canProceed && (
        <p className="text-sm text-amber-600">Please draw a polygon boundary to continue.</p>
      )}

      <div className="flex justify-between pt-4">
        <Button variant="secondary" onClick={onBack}>Back</Button>
        <Button onClick={onNext} disabled={!canProceed}>
          Next: Review & Submit
        </Button>
      </div>
    </div>
  );
}
