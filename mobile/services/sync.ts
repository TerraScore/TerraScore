import { api } from './api';
import { uploadMediaNative } from './upload';
import {
  getPendingUploads,
  markUploadStatus,
  getBufferedLocations,
  clearLocationBuffer,
  getSurveyDraft,
  deleteSurveyDraft,
  clearCompletedUploads,
  type QueuedUpload,
} from './db';

let syncing = false;

/**
 * Run the full sync cycle:
 * 1. Flush buffered locations
 * 2. Retry pending uploads
 * 3. Auto-submit completed survey drafts
 */
export async function runSync(): Promise<void> {
  if (syncing) return;
  syncing = true;

  try {
    await flushLocationBuffer();
    await retryPendingUploads();
  } catch {
    // sync errors are non-fatal
  } finally {
    syncing = false;
  }
}

/**
 * Flush buffered location points to the server.
 */
async function flushLocationBuffer(): Promise<void> {
  const locations = await getBufferedLocations();
  if (locations.length === 0) return;

  try {
    // Send locations in batch via the agent location endpoint
    // The backend expects individual location updates, so we send the latest
    const latest = locations[locations.length - 1];
    await api.post('/v1/agents/me/location', {
      lat: latest.lat,
      lng: latest.lng,
      accuracy: latest.accuracy,
    });
    await clearLocationBuffer();
  } catch {
    // Will retry on next sync
  }
}

/**
 * Retry all pending/failed uploads (max 3 retries per item).
 */
async function retryPendingUploads(): Promise<void> {
  const pending = await getPendingUploads();
  if (pending.length === 0) return;

  // Group by job_id to check for survey auto-submit
  const jobIds = new Set<string>();

  for (const item of pending) {
    jobIds.add(item.job_id);
    await markUploadStatus(item.id, 'uploading');

    try {
      const result = await uploadMediaNative({
        jobId: item.job_id,
        stepId: item.step_id,
        uri: item.file_path,
        contentType: item.content_type,
        sha256: item.sha256,
        fileSize: item.file_size,
        lat: item.lat,
        lng: item.lng,
        capturedAt: item.captured_at,
      });
      await markUploadStatus(item.id, 'done', result.s3_key);
    } catch {
      await markUploadStatus(item.id, 'failed');
    }
  }

  // For each job, check if all uploads are done and auto-submit draft
  for (const jobId of jobIds) {
    await tryAutoSubmitSurvey(jobId);
  }
}

/**
 * If all media for a job is uploaded and a draft exists, auto-submit the survey.
 */
async function tryAutoSubmitSurvey(jobId: string): Promise<void> {
  const remaining = await getPendingUploads();
  const jobRemaining = remaining.filter((u) => u.job_id === jobId);
  if (jobRemaining.length > 0) return; // still has pending uploads

  const draft = await getSurveyDraft(jobId);
  if (!draft) return;

  try {
    const responses = JSON.parse(draft.responses_json);
    const gpsTrail = JSON.parse(draft.gps_trail_json);

    const durationMinutes = draft.started_at
      ? (Date.now() - new Date(draft.started_at).getTime()) / 60000
      : undefined;

    await api.post(`/v1/jobs/${jobId}/survey`, {
      responses,
      gps_trail_geojson: JSON.stringify({
        type: 'LineString',
        coordinates: gpsTrail.map((p: any) => [p.lng ?? p[0], p.lat ?? p[1]]),
      }),
      started_at: draft.started_at,
      duration_minutes: durationMinutes,
      template_id: draft.template_id,
    });

    // Clean up
    await deleteSurveyDraft(jobId);
    await clearCompletedUploads(jobId);
  } catch {
    // Will retry on next sync
  }
}
