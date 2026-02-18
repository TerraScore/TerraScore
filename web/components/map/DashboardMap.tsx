"use client";

import { useEffect, useRef } from "react";
import mapboxgl from "mapbox-gl";
import "mapbox-gl/dist/mapbox-gl.css";
import type { Parcel } from "@/lib/types";

// Indian state centers for marker placement
// Phase 2: add `?include=boundary` to list endpoint and render actual polygons
const STATE_CENTERS: Record<string, [number, number]> = {
  AN: [92.6586, 11.7401], AP: [79.7400, 15.9129], AR: [94.7278, 28.2180],
  AS: [92.9376, 26.2006], BR: [85.3131, 25.0961], CH: [76.7794, 30.7333],
  CT: [81.8661, 21.2787], DD: [70.9874, 20.4283], DL: [77.1025, 28.7041],
  GA: [74.1240, 15.2993], GJ: [71.1924, 22.2587], HP: [77.1734, 31.1048],
  HR: [76.0856, 29.0588], JH: [85.2799, 23.6102], JK: [74.7973, 33.7782],
  KA: [75.7139, 15.3173], KL: [76.2711, 10.8505], LA: [77.5771, 34.1526],
  MH: [75.7139, 19.7515], ML: [91.3662, 25.4670], MN: [93.9063, 24.6637],
  MP: [78.6569, 22.9734], MZ: [92.9376, 23.1645], NL: [94.5624, 26.1584],
  OD: [85.0985, 20.9517], PB: [75.3412, 31.1471], PY: [79.8083, 11.9416],
  RJ: [74.2179, 27.0238], SK: [88.5122, 27.5330], TG: [79.0193, 18.1124],
  TN: [78.6569, 11.1271], TR: [91.9882, 23.9408], UK: [79.0193, 30.0668],
  UP: [80.9462, 26.8467], WB: [87.8550, 22.9868],
};

interface DashboardMapProps {
  parcels: Parcel[];
  selectedParcel: Parcel | null;
  onMarkerClick: (id: string) => void;
}

export function DashboardMap({ parcels, selectedParcel, onMarkerClick }: DashboardMapProps) {
  const mapContainer = useRef<HTMLDivElement>(null);
  const map = useRef<mapboxgl.Map | null>(null);
  const markers = useRef<mapboxgl.Marker[]>([]);

  useEffect(() => {
    if (!mapContainer.current || map.current) return;

    mapboxgl.accessToken = process.env.NEXT_PUBLIC_MAPBOX_TOKEN || "";

    map.current = new mapboxgl.Map({
      container: mapContainer.current,
      style: "mapbox://styles/mapbox/light-v11",
      center: [78.9629, 20.5937], // India center
      zoom: 4.5,
    });

    map.current.addControl(new mapboxgl.NavigationControl(), "top-right");

    return () => {
      map.current?.remove();
      map.current = null;
    };
  }, []);

  // Update markers when parcels change
  useEffect(() => {
    if (!map.current) return;

    // Clear existing markers
    markers.current.forEach((m) => m.remove());
    markers.current = [];

    parcels.forEach((parcel) => {
      const center = STATE_CENTERS[parcel.state_code];
      if (!center) return;

      const el = document.createElement("div");
      el.className = "cursor-pointer";
      el.innerHTML = `<svg width="24" height="32" viewBox="0 0 24 32" fill="none"><path d="M12 0C5.4 0 0 5.4 0 12c0 9 12 20 12 20s12-11 12-20C24 5.4 18.6 0 12 0z" fill="${
        parcel.id === selectedParcel?.id ? "#059669" : "#10b981"
      }"/><circle cx="12" cy="12" r="5" fill="white"/></svg>`;

      const marker = new mapboxgl.Marker({ element: el })
        .setLngLat(center)
        .addTo(map.current!);

      el.addEventListener("click", () => onMarkerClick(parcel.id));
      markers.current.push(marker);
    });
  }, [parcels, selectedParcel?.id, onMarkerClick]);

  // Show boundary polygon when a parcel with boundary is selected
  useEffect(() => {
    if (!map.current) return;
    const m = map.current;

    // Remove existing boundary layer
    if (m.getSource("selected-boundary")) {
      m.removeLayer("selected-boundary-fill");
      m.removeLayer("selected-boundary-line");
      m.removeSource("selected-boundary");
    }

    if (!selectedParcel?.boundary_geojson) return;

    // Wait for style to load
    const addBoundary = () => {
      m.addSource("selected-boundary", {
        type: "geojson",
        data: {
          type: "Feature",
          properties: {},
          geometry: selectedParcel.boundary_geojson as GeoJSON.Geometry,
        },
      });

      m.addLayer({
        id: "selected-boundary-fill",
        type: "fill",
        source: "selected-boundary",
        paint: {
          "fill-color": "#059669",
          "fill-opacity": 0.2,
        },
      });

      m.addLayer({
        id: "selected-boundary-line",
        type: "line",
        source: "selected-boundary",
        paint: {
          "line-color": "#059669",
          "line-width": 2,
        },
      });

      // Fit to boundary
      const coords = (selectedParcel.boundary_geojson as GeoJSON.Polygon).coordinates[0];
      if (coords) {
        const bounds = new mapboxgl.LngLatBounds();
        coords.forEach((coord) => bounds.extend(coord as [number, number]));
        m.fitBounds(bounds, { padding: 80, maxZoom: 16 });
      }
    };

    if (m.isStyleLoaded()) {
      addBoundary();
    } else {
      m.on("style.load", addBoundary);
    }
  }, [selectedParcel]);

  return <div ref={mapContainer} className="w-full h-full" />;
}
