-- 009: Reports table + media hash index for QA duplicate detection

CREATE TABLE IF NOT EXISTS reports (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    parcel_id       UUID NOT NULL REFERENCES parcels(id) ON DELETE CASCADE,
    job_id          UUID NOT NULL REFERENCES survey_jobs(id) ON DELETE CASCADE,
    s3_key          TEXT NOT NULL,
    report_type     TEXT NOT NULL DEFAULT 'survey',
    format          TEXT NOT NULL DEFAULT 'html',
    generated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_reports_parcel_id ON reports(parcel_id);
CREATE INDEX idx_reports_job_id ON reports(job_id);

-- Index for duplicate media detection (QA check)
CREATE INDEX idx_survey_media_file_hash ON survey_media(file_hash_sha256);
