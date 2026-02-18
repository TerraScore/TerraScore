import type { SurveyStep } from '@/types/api';
import type { StepResponse, GPSPoint } from '@/stores/surveyStore';
import { ChecklistStep } from './ChecklistStep';
import { PhotoCapture } from './PhotoCapture';
import { VideoCapture } from './VideoCapture';
import { GPSTraceStep } from './GPSTraceStep';

interface Props {
  step: SurveyStep;
  response: StepResponse | undefined;
  gpsPoints: GPSPoint[];
  location: { lat: number; lng: number; accuracy: number } | null;
  onChecklistChange: (value: 'yes' | 'no' | 'na') => void;
  onMediaCapture: (data: { uri: string; sha256: string; fileSize: number }) => void;
  onGPSPoint: (point: GPSPoint) => void;
  onGPSComplete: (geoJson: string) => void;
}

export function StepRenderer({
  step,
  response,
  gpsPoints,
  location,
  onChecklistChange,
  onMediaCapture,
  onGPSPoint,
  onGPSComplete,
}: Props) {
  switch (step.type) {
    case 'checklist':
      return (
        <ChecklistStep
          step={step}
          value={response?.value ?? null}
          onChange={onChecklistChange}
        />
      );

    case 'photo':
      return (
        <PhotoCapture
          step={step}
          capturedUri={response?.value?.uri ?? null}
          onCapture={onMediaCapture}
        />
      );

    case 'video':
      return (
        <VideoCapture
          step={step}
          capturedUri={response?.value?.uri ?? null}
          onCapture={onMediaCapture}
        />
      );

    case 'gps_trace':
      return (
        <GPSTraceStep
          step={step}
          points={gpsPoints}
          onPointAdded={onGPSPoint}
          onComplete={onGPSComplete}
          completed={response?.completed ?? false}
        />
      );

    default:
      return null;
  }
}
