"use client";

import { useEffect, useRef, useCallback } from "react";
import maplibregl from "maplibre-gl";
import MapboxDraw from "maplibre-gl-draw";
import "maplibre-gl/dist/maplibre-gl.css";
import "maplibre-gl-draw/dist/mapbox-gl-draw.css";

interface DrawMapProps {
  initialGeometry?: GeoJSON.Geometry | null;
  onBoundaryChange: (geometry: GeoJSON.Geometry | null, geoJSONString: string | null) => void;
  center?: [number, number];
}

export function DrawMap({ initialGeometry, onBoundaryChange, center }: DrawMapProps) {
  const mapContainer = useRef<HTMLDivElement>(null);
  const map = useRef<maplibregl.Map | null>(null);
  const draw = useRef<MapboxDraw | null>(null);
  const token = process.env.NEXT_PUBLIC_MAPTILER_KEY;

  const handleDrawChange = useCallback(() => {
    if (!draw.current) return;
    const data = draw.current.getAll();
    if (data.features.length > 0) {
      const feature = data.features[0];
      const geometry = feature.geometry as GeoJSON.Geometry;
      onBoundaryChange(geometry, JSON.stringify(geometry));
    } else {
      onBoundaryChange(null, null);
    }
  }, [onBoundaryChange]);

  useEffect(() => {
    if (!mapContainer.current || map.current || !token) return;

    map.current = new maplibregl.Map({
      container: mapContainer.current,
      style: `https://api.maptiler.com/maps/hybrid/style.json?key=${token}`,
      center: center || [78.9629, 20.5937],
      zoom: center ? 14 : 5,
    });

    map.current.addControl(new maplibregl.NavigationControl(), "top-right");

    draw.current = new MapboxDraw({
      displayControlsDefault: false,
      controls: {
        polygon: true,
        trash: true,
      },
    });

    map.current.addControl(draw.current as unknown as maplibregl.IControl, "top-left");

    map.current.on("draw.create", handleDrawChange);
    map.current.on("draw.update", handleDrawChange);
    map.current.on("draw.delete", handleDrawChange);

    // Load initial geometry if provided
    if (initialGeometry) {
      map.current.on("load", () => {
        if (draw.current && initialGeometry) {
          draw.current.add({
            type: "Feature",
            properties: {},
            geometry: initialGeometry,
          } as GeoJSON.Feature);

          // Fit to existing boundary
          if (initialGeometry.type === "Polygon") {
            const bounds = new maplibregl.LngLatBounds();
            initialGeometry.coordinates[0].forEach((coord) =>
              bounds.extend(coord as [number, number])
            );
            map.current?.fitBounds(bounds, { padding: 80, maxZoom: 16 });
          }
        }
      });
    }

    return () => {
      map.current?.remove();
      map.current = null;
      draw.current = null;
    };
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  if (!token) {
    return (
      <div className="w-full h-full rounded-lg flex items-center justify-center bg-gray-100 text-gray-500 text-sm">
        Map unavailable — NEXT_PUBLIC_MAPTILER_KEY not configured
      </div>
    );
  }

  return <div ref={mapContainer} className="w-full h-full rounded-lg" />;
}
