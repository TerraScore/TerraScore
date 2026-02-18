"use client";

import { useEffect, useRef } from "react";
import mapboxgl from "mapbox-gl";
import "mapbox-gl/dist/mapbox-gl.css";

interface ParcelMapProps {
  boundary: GeoJSON.Geometry;
}

export function ParcelMap({ boundary }: ParcelMapProps) {
  const mapContainer = useRef<HTMLDivElement>(null);
  const map = useRef<mapboxgl.Map | null>(null);

  useEffect(() => {
    if (!mapContainer.current || map.current) return;

    mapboxgl.accessToken = process.env.NEXT_PUBLIC_MAPBOX_TOKEN || "";

    map.current = new mapboxgl.Map({
      container: mapContainer.current,
      style: "mapbox://styles/mapbox/satellite-streets-v12",
      center: [78.9629, 20.5937],
      zoom: 5,
    });

    map.current.addControl(new mapboxgl.NavigationControl(), "top-right");

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
        const bounds = new mapboxgl.LngLatBounds();
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

  return <div ref={mapContainer} className="w-full h-full rounded-lg" />;
}
