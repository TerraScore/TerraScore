"use client";

import { useEffect, useRef, useCallback } from "react";
import mapboxgl from "mapbox-gl";
import MapboxDraw from "@mapbox/mapbox-gl-draw";
import "mapbox-gl/dist/mapbox-gl.css";
import "@mapbox/mapbox-gl-draw/dist/mapbox-gl-draw.css";

interface DrawMapProps {
  initialGeometry?: GeoJSON.Geometry | null;
  onBoundaryChange: (geometry: GeoJSON.Geometry | null, geoJSONString: string | null) => void;
  center?: [number, number];
}

export function DrawMap({ initialGeometry, onBoundaryChange, center }: DrawMapProps) {
  const mapContainer = useRef<HTMLDivElement>(null);
  const map = useRef<mapboxgl.Map | null>(null);
  const draw = useRef<MapboxDraw | null>(null);
  const token = process.env.NEXT_PUBLIC_MAPBOX_TOKEN;

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

    mapboxgl.accessToken = token;

    map.current = new mapboxgl.Map({
      container: mapContainer.current,
      style: "mapbox://styles/mapbox/satellite-streets-v12",
      center: center || [78.9629, 20.5937],
      zoom: center ? 14 : 5,
    });

    map.current.addControl(new mapboxgl.NavigationControl(), "top-right");
    map.current.addControl(
      new mapboxgl.GeolocateControl({
        positionOptions: { enableHighAccuracy: true },
        trackUserLocation: false,
      }),
      "top-right"
    );

    draw.current = new MapboxDraw({
      displayControlsDefault: false,
      controls: {
        polygon: true,
        trash: true,
      },
    });

    map.current.addControl(draw.current as unknown as mapboxgl.IControl, "top-left");

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
            const bounds = new mapboxgl.LngLatBounds();
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
        Map unavailable — NEXT_PUBLIC_MAPBOX_TOKEN not configured
      </div>
    );
  }

  return <div ref={mapContainer} className="w-full h-full rounded-lg" />;
}
