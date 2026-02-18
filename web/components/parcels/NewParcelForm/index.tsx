"use client";

import { useReducer, useCallback } from "react";
import { useRouter } from "next/navigation";
import { Step1Location } from "./Step1Location";
import { Step2Boundary } from "./Step2Boundary";
import { Step3Confirm } from "./Step3Confirm";
import { useCreateParcel } from "@/hooks/useCreateParcel";
import type { CreateParcelRequest } from "@/lib/types";

interface FormState {
  step: 1 | 2 | 3;
  label: string;
  survey_number: string;
  village: string;
  taluk: string;
  district: string;
  state: string;
  state_code: string;
  pin_code: string;
  land_type: string;
  registered_area_sqm: string;
  boundary_geometry: GeoJSON.Geometry | null;
  boundary_string: string | null;
}

type FormAction =
  | { type: "UPDATE_FIELD"; field: string; value: string }
  | { type: "SET_BOUNDARY"; geometry: GeoJSON.Geometry | null; geoJSON: string | null }
  | { type: "SET_STEP"; step: 1 | 2 | 3 };

const initialState: FormState = {
  step: 1,
  label: "",
  survey_number: "",
  village: "",
  taluk: "",
  district: "",
  state: "",
  state_code: "",
  pin_code: "",
  land_type: "",
  registered_area_sqm: "",
  boundary_geometry: null,
  boundary_string: null,
};

function reducer(state: FormState, action: FormAction): FormState {
  switch (action.type) {
    case "UPDATE_FIELD":
      return { ...state, [action.field]: action.value };
    case "SET_BOUNDARY":
      return { ...state, boundary_geometry: action.geometry, boundary_string: action.geoJSON };
    case "SET_STEP":
      return { ...state, step: action.step };
    default:
      return state;
  }
}

export function NewParcelForm() {
  const [state, dispatch] = useReducer(reducer, initialState);
  const router = useRouter();
  const createParcel = useCreateParcel();

  const updateField = useCallback((field: string, value: string) => {
    dispatch({ type: "UPDATE_FIELD", field, value });
  }, []);

  const setBoundary = useCallback((geometry: GeoJSON.Geometry | null, geoJSON: string | null) => {
    dispatch({ type: "SET_BOUNDARY", geometry, geoJSON });
  }, []);

  const canProceedStep1 = state.district.trim() !== "" && state.state.trim() !== "" && state.state_code.trim() !== "";
  const canProceedStep2 = state.boundary_string !== null;

  async function handleSubmit() {
    if (!state.boundary_string) return;

    const req: CreateParcelRequest = {
      label: state.label,
      district: state.district,
      state: state.state,
      state_code: state.state_code,
      boundary: state.boundary_string,
    };
    if (state.survey_number) req.survey_number = state.survey_number;
    if (state.village) req.village = state.village;
    if (state.taluk) req.taluk = state.taluk;
    if (state.pin_code) req.pin_code = state.pin_code;
    if (state.land_type) req.land_type = state.land_type;
    if (state.registered_area_sqm) {
      req.registered_area_sqm = parseFloat(state.registered_area_sqm);
    }

    const result = await createParcel.mutateAsync(req);
    if (result.data?.id) {
      router.push(`/parcels/${result.data.id}`);
    }
  }

  const steps = ["Location", "Boundary", "Confirm"];

  return (
    <div>
      {/* Progress indicator */}
      <div className="flex items-center gap-2 mb-6">
        {steps.map((label, i) => {
          const stepNum = (i + 1) as 1 | 2 | 3;
          const isActive = state.step === stepNum;
          const isComplete = state.step > stepNum;
          return (
            <div key={label} className="flex items-center gap-2">
              {i > 0 && <div className={`w-8 h-px ${isComplete ? "bg-emerald-500" : "bg-gray-300"}`} />}
              <div className="flex items-center gap-1.5">
                <div
                  className={`w-6 h-6 rounded-full flex items-center justify-center text-xs font-medium ${
                    isActive
                      ? "bg-emerald-600 text-white"
                      : isComplete
                        ? "bg-emerald-100 text-emerald-700"
                        : "bg-gray-200 text-gray-500"
                  }`}
                >
                  {isComplete ? "✓" : stepNum}
                </div>
                <span className={`text-sm ${isActive ? "font-medium text-gray-900" : "text-gray-500"}`}>
                  {label}
                </span>
              </div>
            </div>
          );
        })}
      </div>

      {state.step === 1 && (
        <Step1Location
          state={state}
          updateField={updateField}
          onNext={() => dispatch({ type: "SET_STEP", step: 2 })}
          canProceed={canProceedStep1}
        />
      )}

      {state.step === 2 && (
        <Step2Boundary
          geometry={state.boundary_geometry}
          onBoundaryChange={setBoundary}
          onBack={() => dispatch({ type: "SET_STEP", step: 1 })}
          onNext={() => dispatch({ type: "SET_STEP", step: 3 })}
          canProceed={canProceedStep2}
        />
      )}

      {state.step === 3 && (
        <Step3Confirm
          state={state}
          onBack={() => dispatch({ type: "SET_STEP", step: 2 })}
          onSubmit={handleSubmit}
          isSubmitting={createParcel.isPending}
          error={createParcel.error?.message || null}
        />
      )}
    </div>
  );
}
