CREATE TABLE checklist_templates (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(100) NOT NULL,
    survey_type VARCHAR(30) NOT NULL,
    version     INTEGER DEFAULT 1,
    is_active   BOOLEAN DEFAULT TRUE,
    steps       JSONB NOT NULL,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE survey_responses (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id      UUID NOT NULL REFERENCES survey_jobs(id),
    agent_id    UUID NOT NULL REFERENCES agents(id),
    template_id UUID REFERENCES checklist_templates(id),

    responses   JSONB NOT NULL,
    gps_trail   GEOMETRY(LINESTRING, 4326),
    device_info JSONB,

    started_at  TIMESTAMPTZ,
    submitted_at TIMESTAMPTZ DEFAULT NOW(),
    duration_minutes REAL
);

CREATE INDEX idx_responses_job ON survey_responses(job_id);

CREATE TABLE survey_media (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id          UUID NOT NULL REFERENCES survey_jobs(id),
    agent_id        UUID NOT NULL REFERENCES agents(id),
    step_id         VARCHAR(100) NOT NULL,

    media_type      VARCHAR(10) NOT NULL,
    s3_key          TEXT NOT NULL,
    file_size_bytes BIGINT,
    duration_sec    INTEGER,

    location        GEOMETRY(POINT, 4326) NOT NULL,
    captured_at     TIMESTAMPTZ NOT NULL,

    file_hash_sha256 VARCHAR(64) NOT NULL,
    device_id       VARCHAR(100),

    within_boundary BOOLEAN,
    duplicate_hash  VARCHAR(64),

    uploaded_at     TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_media_job ON survey_media(job_id);
