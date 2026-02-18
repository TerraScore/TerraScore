import axios from 'axios';
import {
  readAsStringAsync,
  getInfoAsync,
  uploadAsync,
  EncodingType,
  FileSystemUploadType,
} from 'expo-file-system/legacy';
import * as Crypto from 'expo-crypto';
import { api } from './api';
import type { ApiResponse, PresignedURLResponse, MediaResponse } from '@/types/api';

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
 * Full upload pipeline for a single media file using native binary upload:
 * 1. Get presigned URL
 * 2. PUT file to S3 via expo-file-system uploadAsync
 * 3. POST metadata to backend
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
  const { jobId, stepId, uri, contentType, sha256, fileSize, lat, lng, accuracy, capturedAt } =
    params;

  // 1. Get presigned URL
  const { data: presignedResp } = await api.get<ApiResponse<PresignedURLResponse>>(
    `/v1/jobs/${jobId}/media/presigned`,
    { params: { content_type: contentType, step_id: stepId } },
  );
  const presigned = presignedResp.data!;

  // 2. Upload file to S3 using native uploader
  const uploadResult = await uploadAsync(presigned.upload_url, uri, {
    httpMethod: 'PUT',
    headers: { 'Content-Type': contentType },
    uploadType: FileSystemUploadType.BINARY_CONTENT,
  });

  if (uploadResult.status < 200 || uploadResult.status >= 300) {
    throw new Error(`S3 upload failed with status ${uploadResult.status}`);
  }

  // 3. Record metadata in backend
  const mediaType = contentType.startsWith('video/') ? 'video' : 'photo';
  const { data: metaResp } = await api.post<ApiResponse<MediaResponse>>(
    `/v1/jobs/${jobId}/media`,
    {
      s3_key: presigned.s3_key,
      step_id: stepId,
      media_type: mediaType,
      lat,
      lng,
      accuracy: accuracy ?? 0,
      sha256,
      file_size: fileSize,
      captured_at: capturedAt,
    },
  );

  return metaResp.data!;
}
