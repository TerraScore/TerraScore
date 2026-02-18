import { create } from 'zustand';
import type { SurveyStep } from '@/types/api';

export interface StepResponse {
  step_id: string;
  type: SurveyStep['type'];
  value: any; // checklist: string, photo/video: { uri, sha256, s3_key, ... }, gps_trace: GeoJSON
  completed: boolean;
}

export interface MediaItem {
  step_id: string;
  uri: string;
  sha256: string;
  s3_key?: string;
  content_type: string;
  file_size: number;
  lat: number;
  lng: number;
  captured_at: string;
  uploaded: boolean;
}

export interface GPSPoint {
  lat: number;
  lng: number;
  accuracy: number;
  timestamp: number;
}

interface SurveyState {
  jobId: string | null;
  templateId: string | null;
  steps: SurveyStep[];
  responses: Record<string, StepResponse>;
  mediaQueue: MediaItem[];
  gpsTrail: GPSPoint[];
  startedAt: string | null;
  currentStepIndex: number;

  init: (jobId: string, templateId: string, steps: SurveyStep[]) => void;
  setResponse: (stepId: string, response: StepResponse) => void;
  addMedia: (item: MediaItem) => void;
  markMediaUploaded: (stepId: string, s3Key: string) => void;
  addGPSPoint: (point: GPSPoint) => void;
  nextStep: () => void;
  prevStep: () => void;
  reset: () => void;
  isAllComplete: () => boolean;
}

export const useSurveyStore = create<SurveyState>((set, get) => ({
  jobId: null,
  templateId: null,
  steps: [],
  responses: {},
  mediaQueue: [],
  gpsTrail: [],
  startedAt: null,
  currentStepIndex: 0,

  init: (jobId, templateId, steps) => {
    set({
      jobId,
      templateId,
      steps,
      responses: {},
      mediaQueue: [],
      gpsTrail: [],
      startedAt: new Date().toISOString(),
      currentStepIndex: 0,
    });
  },

  setResponse: (stepId, response) => {
    set((state) => ({
      responses: { ...state.responses, [stepId]: response },
    }));
  },

  addMedia: (item) => {
    set((state) => ({
      mediaQueue: [...state.mediaQueue, item],
    }));
  },

  markMediaUploaded: (stepId, s3Key) => {
    set((state) => ({
      mediaQueue: state.mediaQueue.map((m) =>
        m.step_id === stepId && !m.uploaded ? { ...m, uploaded: true, s3_key: s3Key } : m,
      ),
    }));
  },

  addGPSPoint: (point) => {
    set((state) => ({
      gpsTrail: [...state.gpsTrail, point],
    }));
  },

  nextStep: () => {
    set((state) => ({
      currentStepIndex: Math.min(state.currentStepIndex + 1, state.steps.length - 1),
    }));
  },

  prevStep: () => {
    set((state) => ({
      currentStepIndex: Math.max(state.currentStepIndex - 1, 0),
    }));
  },

  reset: () => {
    set({
      jobId: null,
      templateId: null,
      steps: [],
      responses: {},
      mediaQueue: [],
      gpsTrail: [],
      startedAt: null,
      currentStepIndex: 0,
    });
  },

  isAllComplete: () => {
    const { steps, responses } = get();
    return steps
      .filter((s) => s.required)
      .every((s) => responses[s.id]?.completed);
  },
}));
