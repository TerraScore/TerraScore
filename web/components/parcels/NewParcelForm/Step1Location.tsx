"use client";

import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { Label } from "@/components/ui/Label";

const INDIAN_STATES: { name: string; code: string }[] = [
  { name: "Andhra Pradesh", code: "AP" }, { name: "Arunachal Pradesh", code: "AR" },
  { name: "Assam", code: "AS" }, { name: "Bihar", code: "BR" },
  { name: "Chhattisgarh", code: "CT" }, { name: "Goa", code: "GA" },
  { name: "Gujarat", code: "GJ" }, { name: "Haryana", code: "HR" },
  { name: "Himachal Pradesh", code: "HP" }, { name: "Jharkhand", code: "JH" },
  { name: "Karnataka", code: "KA" }, { name: "Kerala", code: "KL" },
  { name: "Madhya Pradesh", code: "MP" }, { name: "Maharashtra", code: "MH" },
  { name: "Manipur", code: "MN" }, { name: "Meghalaya", code: "ML" },
  { name: "Mizoram", code: "MZ" }, { name: "Nagaland", code: "NL" },
  { name: "Odisha", code: "OD" }, { name: "Punjab", code: "PB" },
  { name: "Rajasthan", code: "RJ" }, { name: "Sikkim", code: "SK" },
  { name: "Tamil Nadu", code: "TN" }, { name: "Telangana", code: "TG" },
  { name: "Tripura", code: "TR" }, { name: "Uttar Pradesh", code: "UP" },
  { name: "Uttarakhand", code: "UK" }, { name: "West Bengal", code: "WB" },
  { name: "Delhi", code: "DL" }, { name: "Jammu & Kashmir", code: "JK" },
  { name: "Ladakh", code: "LA" }, { name: "Chandigarh", code: "CH" },
  { name: "Puducherry", code: "PY" }, { name: "Andaman & Nicobar", code: "AN" },
  { name: "Dadra & Nagar Haveli and Daman & Diu", code: "DD" }, { name: "Lakshadweep", code: "LD" },
];

const LAND_TYPES = [
  "agricultural", "residential", "commercial", "industrial", "forest", "barren", "wetland", "other",
];

interface Step1Props {
  state: {
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
  };
  updateField: (field: string, value: string) => void;
  onNext: () => void;
  canProceed: boolean;
}

export function Step1Location({ state, updateField, onNext, canProceed }: Step1Props) {
  function handleStateChange(e: React.ChangeEvent<HTMLSelectElement>) {
    const selectedState = INDIAN_STATES.find((s) => s.name === e.target.value);
    updateField("state", e.target.value);
    updateField("state_code", selectedState?.code || "");
  }

  return (
    <div className="space-y-4">
      <div className="grid grid-cols-2 gap-4">
        <div>
          <Label htmlFor="label">Parcel Label</Label>
          <Input
            id="label"
            placeholder="e.g., North Field"
            value={state.label}
            onChange={(e) => updateField("label", e.target.value)}
          />
        </div>
        <div>
          <Label htmlFor="survey_number">Survey Number</Label>
          <Input
            id="survey_number"
            placeholder="e.g., 123/4A"
            value={state.survey_number}
            onChange={(e) => updateField("survey_number", e.target.value)}
          />
        </div>
      </div>

      <div className="grid grid-cols-2 gap-4">
        <div>
          <Label htmlFor="village">Village</Label>
          <Input
            id="village"
            value={state.village}
            onChange={(e) => updateField("village", e.target.value)}
          />
        </div>
        <div>
          <Label htmlFor="taluk">Taluk</Label>
          <Input
            id="taluk"
            value={state.taluk}
            onChange={(e) => updateField("taluk", e.target.value)}
          />
        </div>
      </div>

      <div className="grid grid-cols-2 gap-4">
        <div>
          <Label htmlFor="district" required>District</Label>
          <Input
            id="district"
            value={state.district}
            onChange={(e) => updateField("district", e.target.value)}
            required
          />
        </div>
        <div>
          <Label htmlFor="state" required>State</Label>
          <select
            id="state"
            value={state.state}
            onChange={handleStateChange}
            className="w-full rounded-lg border border-gray-300 bg-white px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-emerald-500 focus:border-transparent"
            required
          >
            <option value="">Select state</option>
            {INDIAN_STATES.map((s) => (
              <option key={s.code} value={s.name}>{s.name}</option>
            ))}
          </select>
        </div>
      </div>

      <div className="grid grid-cols-3 gap-4">
        <div>
          <Label htmlFor="pin_code">PIN Code</Label>
          <Input
            id="pin_code"
            value={state.pin_code}
            onChange={(e) => updateField("pin_code", e.target.value)}
            maxLength={6}
          />
        </div>
        <div>
          <Label htmlFor="land_type">Land Type</Label>
          <select
            id="land_type"
            value={state.land_type}
            onChange={(e) => updateField("land_type", e.target.value)}
            className="w-full rounded-lg border border-gray-300 bg-white px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-emerald-500 focus:border-transparent"
          >
            <option value="">Select type</option>
            {LAND_TYPES.map((t) => (
              <option key={t} value={t}>{t.charAt(0).toUpperCase() + t.slice(1)}</option>
            ))}
          </select>
        </div>
        <div>
          <Label htmlFor="registered_area_sqm">Registered Area (sq m)</Label>
          <Input
            id="registered_area_sqm"
            type="number"
            step="0.01"
            value={state.registered_area_sqm}
            onChange={(e) => updateField("registered_area_sqm", e.target.value)}
          />
        </div>
      </div>

      <div className="flex justify-end pt-4">
        <Button onClick={onNext} disabled={!canProceed}>
          Next: Draw Boundary
        </Button>
      </div>
    </div>
  );
}
