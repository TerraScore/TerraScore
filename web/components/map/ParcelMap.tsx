"use client";

import { useEffect, useRef } from "react";
import maplibregl from "maplibre-gl";
import "maplibre-gl/dist/maplibre-gl.css";

interface ParcelMapProps {
  boundary: GeoJSON.Geometry;
}

export function ParcelMap({ boundary }: ParcelMapProps) {
  const mapContainer = useRef<HTMLDivElement>(null);
  const map = useRef<maplibregl.Map | null>(null);
  const token = process.env.NEXT_PUBLIC_MAPTILER_KEY;

  useEffect(() => {
    if (!mapContainer.current || map.current || !token) return;

    map.current = new maplibregl.Map({
      container: mapContainer.current,
      style: `https://api.maptiler.com/maps/hybrid/style.json?key=${token}`,
      center: [78.9629, 20.5937],
      zoom: 5,
    });

    map.current.addControl(new maplibregl.NavigationControl(), "top-right");

    map.current.on("load", () => {
      if (!map.current) return;

      map.current.addSource("parcel-boundary", {
        type: "geojson",
        data: {
          type: "Feature",
          properties: {},
          geometry: boundary,
        },
      });

      map.current.addLayer({
        id: "parcel-fill",
        type: "fill",
        source: "parcel-boundary",
        paint: {
          "fill-color": "#059669",
          "fill-opacity": 0.3,
        },
      });

      map.current.addLayer({
        id: "parcel-line",
        type: "line",
        source: "parcel-boundary",
        paint: {
          "line-color": "#059669",
          "line-width": 2,
        },
      });

      // Fit to boundary
      if (boundary.type === "Polygon") {
        const bounds = new maplibregl.LngLatBounds();
        boundary.coordinates[0].forEach((coord) =>
          bounds.extend(coord as [number, number])
        );
        map.current.fitBounds(bounds, { padding: 60, maxZoom: 16 });
      }
    });

    return () => {
      map.current?.remove();
      map.current = null;
    };
  }, [boundary]);

  if (!token) {
    return (
      <div className="w-full h-full rounded-lg flex items-center justify-center bg-gray-100 text-gray-500 text-sm">
        Map unavailable — NEXT_PUBLIC_MAPTILER_KEY not configured
      </div>
    );
  }

  return <div ref={mapContainer} className="w-full h-full rounded-lg" />;
}
