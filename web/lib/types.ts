// API envelope — mirrors internal/platform/httputil.go
export interface ApiResponse<T> {
  data?: T;
  error?: {
    code: string;
    message: string;
  };
  meta?: PaginationMeta;
}

export interface PaginationMeta {
  page: number;
  per_page: number;
  total: number;
  total_pages: number;
}

// Auth — mirrors internal/auth/service.go
export interface RegisterRequest {
  phone: string;
  full_name: string;
  email?: string;
  role: string;
}

export interface LoginRequest {
  phone: string;
}

export interface VerifyOTPRequest {
  phone: string;
  otp: string;
}

export interface VerifyOTPResponse {
  access_token: string;
  refresh_token: string;
  expires_in: number;
}

export interface AuthMessageResponse {
  message: string;
  phone: string;
}

// Parcels — mirrors internal/land/service.go
export interface Parcel {
  id: string;
  label: string | null;
  survey_number?: string | null;
  village?: string | null;
  taluk?: string | null;
  district: string;
  state: string;
  state_code: string;
  pin_code?: string | null;
  boundary_geojson?: GeoJSON.Geometry | null;
  area_sqm?: number | null;
  land_type?: string | null;
  registered_area_sqm?: number | null;
  status: string | null;
}

export interface CreateParcelRequest {
  label: string;
  survey_number?: string;
  village?: string;
  taluk?: string;
  district: string;
  state: string;
  state_code: string;
  pin_code?: string;
  boundary: string; // GeoJSON string
  land_type?: string;
  registered_area_sqm?: number;
}

export interface UpdateBoundaryRequest {
  boundary: string; // GeoJSON string
}

// User profile (decoded from JWT)
export interface UserProfile {
  sub: string;
  phone: string;
  name: string;
  role: string;
}
