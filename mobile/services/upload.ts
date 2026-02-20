import {
  readAsStringAsync,
  getInfoAsync,
  uploadAsync,
  EncodingType,
  FileSystemUploadType,
} from 'expo-file-system/legacy';
import * as Crypto from 'expo-crypto';
import { BASE_URL, getStoredToken } from './api';
import type { ApiResponse, MediaResponse } from '@/types/api';

/**
 * Compute SHA-256 hash of a local file (over raw bytes).
 * Reads the file as base64, decodes to binary, then hashes.
 */
export async function hashFile(uri: string): Promise<string> {
  const base64 = await readAsStringAsync(uri, {
    encoding: EncodingType.Base64,
  });
  // Decode base64 to Uint8Array
  const binaryString = atob(base64);
  const bytes = new Uint8Array(binaryString.length);
  for (let i = 0; i < binaryString.length; i++) {
    bytes[i] = binaryString.charCodeAt(i);
  }
  // Hash the raw bytes
  const hashBuffer = await Crypto.digest(Crypto.CryptoDigestAlgorithm.SHA256, bytes);
  // Convert ArrayBuffer to hex string
  const hashArray = new Uint8Array(hashBuffer);
  return Array.from(hashArray)
    .map((b) => b.toString(16).padStart(2, '0'))
    .join('');
}

/**
 * Get file size in bytes.
 */
export async function getFileSize(uri: string): Promise<number> {
  const info = await getInfoAsync(uri);
  if (!info.exists) throw new Error('File not found');
  return (info as any).size ?? 0;
}

/**
 * Upload media via API proxy (multipart form).
 * The Go API handles the S3 upload server-side, avoiding direct
 * mobile-to-MinIO HTTP issues with expo-file-system.
 */
export async function uploadMediaNative(params: {
  jobId: string;
  stepId: string;
  uri: string;
  contentType: string;
  sha256: string;
  fileSize: number;
  lat: number;
  lng: number;
  accuracy?: number;
  capturedAt: string;
}): Promise<MediaResponse> {
  const { jobId, stepId, uri, contentType, lat, lng, capturedAt } = params;

  const mediaType = contentType.startsWith('video/') ? 'video' : 'photo';

  // Get auth token for the upload request
  const token = await getStoredToken();

  // Upload via multipart form to the proxy endpoint
  const uploadResult = await uploadAsync(
    `${BASE_URL}/v1/jobs/${jobId}/media/upload`,
    uri,
    {
      httpMethod: 'POST',
      uploadType: FileSystemUploadType.MULTIPART,
      fieldName: 'file',
      mimeType: contentType,
      headers: {
        ...(token ? { Authorization: `Bearer ${token}` } : {}),
      },
      parameters: {
        step_id: stepId,
        media_type: mediaType,
        lat: String(lat),
        lng: String(lng),
        captured_at: capturedAt,
      },
    },
  );

  if (uploadResult.status < 200 || uploadResult.status >= 300) {
    throw new Error(`Upload failed with status ${uploadResult.status}: ${uploadResult.body}`);
  }

  const resp = JSON.parse(uploadResult.body) as ApiResponse<MediaResponse>;
  return resp.data!;
}
