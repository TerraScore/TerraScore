// API response envelope
export interface ApiResponse<T> {
  data?: T;
  error?: ApiError;
  meta?: PaginationMeta;
}

export interface ApiError {
  code: string;
  message: string;
}

export interface PaginationMeta {
  page: number;
  per_page: number;
  total: number;
  total_pages: number;
}

// Auth
export interface VerifyOTPResponse {
  access_token: string;
  refresh_token: string;
  expires_in: number;
}

// Agent
export interface AgentProfile {
  id: string;
  full_name: string;
  phone: string;
  email?: string | null;
  vehicle_type?: string | null;
  preferred_radius_km?: number | null;
  status?: string | null;
  tier?: string | null;
  is_online?: boolean | null;
  available_days?: string[] | null;
  total_jobs_completed?: number | null;
}

// Parcel (embedded in job details)
export interface Parcel {
  id: string;
  label?: string | null;
  survey_number?: string | null;
  village?: string | null;
  taluk?: string | null;
  district: string;
  state: string;
  state_code: string;
  pin_code?: string | null;
  boundary_geojson?: any;
  area_sqm?: number | null;
  land_type?: string | null;
  registered_area_sqm?: number | null;
  status?: string | null;
}

// Job
export interface Job {
  id: string;
  parcel_id: string;
  user_id: string;
  survey_type: string;
  priority?: string | null;
  deadline: string;
  status?: string | null;
  assigned_agent_id?: string | null;
  assigned_at?: string | null;
  created_at: string;
  // Enriched fields (may come from joins)
  parcel?: Parcel;
  payout_amount?: number;
}

// Offer
export interface Offer {
  id: string;
  job_id: string;
  agent_id: string;
  cascade_round: number;
  offer_rank: number;
  distance_km?: number | null;
  status?: string | null;
  expires_at: string;
  sent_at?: string | null;
  // Enriched
  job?: Job;
}

// WebSocket offer message
export interface WSOfferMessage {
  offer_id: string;
  job_id: string;
  expires_at: string;
}

// Arrive
export interface ArriveRequest {
  lat: number;
  lng: number;
  accuracy?: number;
}

// Media
export interface PresignedURLResponse {
  upload_url: string;
  s3_key: string;
  expires_in: number;
}

export interface MediaRequest {
  s3_key: string;
  step_id: string;
  media_type: string;
  lat: number;
  lng: number;
  accuracy?: number;
  sha256: string;
  file_size?: number;
  captured_at: string;
}

export interface MediaResponse {
  id: string;
  s3_key: string;
  step_id: string;
  media_type: string;
  uploaded_at: string;
}

// Survey
export interface SurveySubmitRequest {
  responses: any;
  gps_trail_geojson: string;
  device_info?: any;
  started_at?: string;
  duration_minutes?: number;
  template_id?: string;
}

export interface SurveyTemplate {
  id: string;
  name: string;
  survey_type: string;
  version?: number;
  steps: SurveyStep[];
}

export interface SurveyStep {
  id: string;
  type: 'photo' | 'video' | 'checklist' | 'gps_trace';
  title: string;
  description?: string;
  required: boolean;
  options?: string[]; // for checklist items
}

// Location
export interface LocationUpdate {
  lat: number;
  lng: number;
  accuracy: number;
}
