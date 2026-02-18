import { openDatabaseAsync, type SQLiteDatabase } from 'expo-sqlite';

let db: SQLiteDatabase | null = null;

export async function getDB(): Promise<SQLiteDatabase> {
  if (db) return db;
  db = await openDatabaseAsync('terrascore.db');
  await initTables(db);
  return db;
}

async function initTables(database: SQLiteDatabase): Promise<void> {
  await database.execAsync(`
    CREATE TABLE IF NOT EXISTS upload_queue (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      job_id TEXT NOT NULL,
      file_path TEXT NOT NULL,
      s3_key TEXT,
      content_type TEXT NOT NULL,
      step_id TEXT NOT NULL,
      sha256 TEXT NOT NULL,
      file_size INTEGER NOT NULL,
      lat REAL NOT NULL,
      lng REAL NOT NULL,
      captured_at TEXT NOT NULL,
      status TEXT NOT NULL DEFAULT 'pending',
      retries INTEGER NOT NULL DEFAULT 0,
      created_at TEXT NOT NULL DEFAULT (datetime('now'))
    );

    CREATE TABLE IF NOT EXISTS survey_draft (
      job_id TEXT PRIMARY KEY,
      template_id TEXT NOT NULL,
      responses_json TEXT NOT NULL DEFAULT '{}',
      gps_trail_json TEXT NOT NULL DEFAULT '[]',
      started_at TEXT,
      updated_at TEXT NOT NULL DEFAULT (datetime('now'))
    );

    CREATE TABLE IF NOT EXISTS location_buffer (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      lat REAL NOT NULL,
      lng REAL NOT NULL,
      accuracy REAL NOT NULL,
      timestamp INTEGER NOT NULL
    );
  `);
}

// --- Upload Queue ---

export interface QueuedUpload {
  id: number;
  job_id: string;
  file_path: string;
  s3_key: string | null;
  content_type: string;
  step_id: string;
  sha256: string;
  file_size: number;
  lat: number;
  lng: number;
  captured_at: string;
  status: string;
  retries: number;
}

export async function enqueueUpload(params: {
  jobId: string;
  filePath: string;
  contentType: string;
  stepId: string;
  sha256: string;
  fileSize: number;
  lat: number;
  lng: number;
  capturedAt: string;
}): Promise<void> {
  const database = await getDB();
  await database.runAsync(
    `INSERT INTO upload_queue (job_id, file_path, content_type, step_id, sha256, file_size, lat, lng, captured_at)
     VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
    params.jobId,
    params.filePath,
    params.contentType,
    params.stepId,
    params.sha256,
    params.fileSize,
    params.lat,
    params.lng,
    params.capturedAt,
  );
}

export async function getPendingUploads(): Promise<QueuedUpload[]> {
  const database = await getDB();
  return database.getAllAsync<QueuedUpload>(
    `SELECT * FROM upload_queue WHERE status IN ('pending', 'failed') AND retries < 3 ORDER BY created_at ASC`,
  );
}

export async function markUploadStatus(id: number, status: 'uploading' | 'done' | 'failed', s3Key?: string): Promise<void> {
  const database = await getDB();
  if (status === 'done' && s3Key) {
    await database.runAsync(
      `UPDATE upload_queue SET status = ?, s3_key = ? WHERE id = ?`,
      status,
      s3Key,
      id,
    );
  } else {
    await database.runAsync(
      `UPDATE upload_queue SET status = ?, retries = retries + 1 WHERE id = ?`,
      status,
      id,
    );
  }
}

export async function clearCompletedUploads(jobId: string): Promise<void> {
  const database = await getDB();
  await database.runAsync(`DELETE FROM upload_queue WHERE job_id = ? AND status = 'done'`, jobId);
}

// --- Survey Draft ---

export async function saveSurveyDraft(params: {
  jobId: string;
  templateId: string;
  responsesJson: string;
  gpsTrailJson: string;
  startedAt: string | null;
}): Promise<void> {
  const database = await getDB();
  await database.runAsync(
    `INSERT OR REPLACE INTO survey_draft (job_id, template_id, responses_json, gps_trail_json, started_at, updated_at)
     VALUES (?, ?, ?, ?, ?, datetime('now'))`,
    params.jobId,
    params.templateId,
    params.responsesJson,
    params.gpsTrailJson,
    params.startedAt,
  );
}

export interface SurveyDraft {
  job_id: string;
  template_id: string;
  responses_json: string;
  gps_trail_json: string;
  started_at: string | null;
  updated_at: string;
}

export async function getSurveyDraft(jobId: string): Promise<SurveyDraft | null> {
  const database = await getDB();
  return database.getFirstAsync<SurveyDraft>(
    `SELECT * FROM survey_draft WHERE job_id = ?`,
    jobId,
  );
}

export async function deleteSurveyDraft(jobId: string): Promise<void> {
  const database = await getDB();
  await database.runAsync(`DELETE FROM survey_draft WHERE job_id = ?`, jobId);
}

// --- Location Buffer ---

export async function bufferLocation(lat: number, lng: number, accuracy: number, timestamp: number): Promise<void> {
  const database = await getDB();
  await database.runAsync(
    `INSERT INTO location_buffer (lat, lng, accuracy, timestamp) VALUES (?, ?, ?, ?)`,
    lat,
    lng,
    accuracy,
    timestamp,
  );
}

export interface BufferedLocation {
  id: number;
  lat: number;
  lng: number;
  accuracy: number;
  timestamp: number;
}

export async function getBufferedLocations(): Promise<BufferedLocation[]> {
  const database = await getDB();
  return database.getAllAsync<BufferedLocation>(
    `SELECT * FROM location_buffer ORDER BY timestamp ASC`,
  );
}

export async function clearLocationBuffer(): Promise<void> {
  const database = await getDB();
  await database.runAsync(`DELETE FROM location_buffer`);
}
