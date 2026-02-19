"use client";

import { useEffect, useRef, useCallback } from "react";
import maplibregl from "maplibre-gl";
import MapboxDraw from "@mapbox/mapbox-gl-draw";
import "maplibre-gl/dist/maplibre-gl.css";
import "@mapbox/mapbox-gl-draw/dist/mapbox-gl-draw.css";

// Patched draw styles — fixes line-dasharray incompatibility with MapLibre
// MapLibre requires ["literal", [...]] for array values in expressions
const drawStyles = [
  // Polygon fill (active)
  { id: "gl-draw-polygon-fill-active", type: "fill", filter: ["all", ["==", "$type", "Polygon"], ["==", "active", "true"]], paint: { "fill-color": "#fbb03b", "fill-outline-color": "#fbb03b", "fill-opacity": 0.1 } },
  // Polygon fill (inactive)
  { id: "gl-draw-polygon-fill-inactive", type: "fill", filter: ["all", ["==", "$type", "Polygon"], ["==", "active", "false"]], paint: { "fill-color": "#3bb2d0", "fill-outline-color": "#3bb2d0", "fill-opacity": 0.1 } },
  // Polygon outline (active)
  { id: "gl-draw-polygon-stroke-active", type: "line", filter: ["all", ["==", "$type", "Polygon"], ["==", "active", "true"]], layout: { "line-cap": "round", "line-join": "round" }, paint: { "line-color": "#fbb03b", "line-dasharray": ["literal", [0.2, 2]], "line-width": 2 } },
  // Polygon outline (inactive)
  { id: "gl-draw-polygon-stroke-inactive", type: "line", filter: ["all", ["==", "$type", "Polygon"], ["==", "active", "false"]], layout: { "line-cap": "round", "line-join": "round" }, paint: { "line-color": "#3bb2d0", "line-dasharray": ["literal", [0.2, 2]], "line-width": 2 } },
  // Polygon midpoints
  { id: "gl-draw-polygon-midpoint", type: "circle", filter: ["all", ["==", "$type", "Point"], ["==", "meta", "midpoint"]], paint: { "circle-radius": 3, "circle-color": "#fbb03b" } },
  // Vertex points (active)
  { id: "gl-draw-point-active", type: "circle", filter: ["all", ["==", "$type", "Point"], ["==", "meta", "vertex"], ["==", "active", "true"]], paint: { "circle-radius": 5, "circle-color": "#fff", "circle-stroke-color": "#fbb03b", "circle-stroke-width": 2 } },
  // Vertex points (inactive)
  { id: "gl-draw-point-inactive", type: "circle", filter: ["all", ["==", "$type", "Point"], ["==", "meta", "vertex"], ["==", "active", "false"]], paint: { "circle-radius": 3, "circle-color": "#fff", "circle-stroke-color": "#3bb2d0", "circle-stroke-width": 2 } },
  // Line (active)
  { id: "gl-draw-line-active", type: "line", filter: ["all", ["==", "$type", "LineString"], ["==", "active", "true"]], layout: { "line-cap": "round", "line-join": "round" }, paint: { "line-color": "#fbb03b", "line-dasharray": ["literal", [0.2, 2]], "line-width": 2 } },
  // Line (inactive)
  { id: "gl-draw-line-inactive", type: "line", filter: ["all", ["==", "$type", "LineString"], ["==", "active", "false"]], layout: { "line-cap": "round", "line-join": "round" }, paint: { "line-color": "#3bb2d0", "line-dasharray": ["literal", [0.2, 2]], "line-width": 2 } },
];

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
    map.current.addControl(
      new maplibregl.GeolocateControl({
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
      styles: drawStyles,
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
