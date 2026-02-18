"use client";

import dynamic from "next/dynamic";
import { Button } from "@/components/ui/Button";

const ParcelMap = dynamic(
  () => import("@/components/map/ParcelMap").then((m) => m.ParcelMap),
  { ssr: false }
);

interface Step3Props {
  state: {
    label: string;
    survey_number: string;
    village: string;
    taluk: string;
    district: string;
    state: string;
    pin_code: string;
    land_type: string;
    registered_area_sqm: string;
    boundary_geometry: GeoJSON.Geometry | null;
  };
  onBack: () => void;
  onSubmit: () => void;
  isSubmitting: boolean;
  error: string | null;
}

export function Step3Confirm({ state, onBack, onSubmit, isSubmitting, error }: Step3Props) {
  const fields = [
    { label: "Label", value: state.label },
    { label: "Survey Number", value: state.survey_number },
    { label: "Village", value: state.village },
    { label: "Taluk", value: state.taluk },
    { label: "District", value: state.district },
    { label: "State", value: state.state },
    { label: "PIN Code", value: state.pin_code },
    { label: "Land Type", value: state.land_type },
    { label: "Registered Area", value: state.registered_area_sqm ? `${state.registered_area_sqm} sq m` : "" },
  ].filter((f) => f.value);

  return (
    <div className="space-y-6">
      <div>
        <h3 className="text-sm font-semibold text-gray-900 mb-3">Location Details</h3>
        <div className="grid grid-cols-2 gap-3">
          {fields.map((f) => (
            <div key={f.label} className="bg-gray-50 rounded-lg px-3 py-2">
              <p className="text-xs text-gray-500">{f.label}</p>
              <p className="text-sm font-medium text-gray-900">{f.value}</p>
            </div>
          ))}
        </div>
      </div>

      {state.boundary_geometry && (
        <div>
          <h3 className="text-sm font-semibold text-gray-900 mb-3">Boundary Preview</h3>
          <div className="h-64 rounded-lg overflow-hidden border border-gray-200">
            <ParcelMap boundary={state.boundary_geometry} />
          </div>
        </div>
      )}

      {error && (
        <div className="bg-red-50 border border-red-200 rounded-lg p-3">
          <p className="text-sm text-red-800">{error}</p>
        </div>
      )}

      <div className="flex justify-between pt-4">
        <Button variant="secondary" onClick={onBack} disabled={isSubmitting}>Back</Button>
        <Button onClick={onSubmit} loading={isSubmitting}>
          Register Parcel
        </Button>
      </div>
    </div>
  );
}
