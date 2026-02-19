declare module "maplibre-gl-draw" {
  import type { IControl, Map } from "maplibre-gl";

  interface DrawOptions {
    displayControlsDefault?: boolean;
    controls?: {
      point?: boolean;
      line_string?: boolean;
      polygon?: boolean;
      trash?: boolean;
      combine_features?: boolean;
      uncombine_features?: boolean;
    };
    defaultMode?: string;
    styles?: Record<string, unknown>[];
  }

  export default class MapboxDraw implements IControl {
    constructor(options?: DrawOptions);
    onAdd(map: Map): HTMLElement;
    onRemove(map: Map): void;
    add(geojson: GeoJSON.Feature | GeoJSON.FeatureCollection): string[];
    getAll(): GeoJSON.FeatureCollection;
    delete(ids: string | string[]): this;
    deleteAll(): this;
    set(featureCollection: GeoJSON.FeatureCollection): string[];
    getMode(): string;
    changeMode(mode: string, options?: Record<string, unknown>): this;
    trash(): this;
  }
}

declare module "@mapbox/geojson-area" {
  export function geometry(geojson: GeoJSON.Geometry): number;
  export function ring(coordinates: number[][]): number;
}
