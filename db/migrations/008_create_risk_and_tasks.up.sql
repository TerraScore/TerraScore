CREATE TABLE risk_scores (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    parcel_id       UUID NOT NULL REFERENCES parcels(id),
    job_id          UUID,

    overall_score   NUMERIC(5,2) NOT NULL,
    risk_level      VARCHAR(10) NOT NULL,

    encroachment_score  NUMERIC(5,2),
    boundary_score      NUMERIC(5,2),
    environmental_score NUMERIC(5,2),
    neighborhood_score  NUMERIC(5,2),

    contributing_factors JSONB,
    computed_at     TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_risk_parcel ON risk_scores(parcel_id, computed_at DESC);

CREATE TABLE task_queue (
    id              BIGSERIAL PRIMARY KEY,
    task_type       VARCHAR(50) NOT NULL,
    payload         JSONB NOT NULL,
    status          VARCHAR(20) DEFAULT 'pending',
    priority        INTEGER DEFAULT 0,
    attempts        INTEGER DEFAULT 0,
    max_attempts    INTEGER DEFAULT 3,
    last_error      TEXT,
    error_message   TEXT,
    scheduled_at    TIMESTAMPTZ DEFAULT NOW(),
    started_at      TIMESTAMPTZ,
    completed_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_task_queue_pending ON task_queue(status, priority DESC, scheduled_at)
    WHERE status = 'pending';

CREATE TABLE alerts (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id),
    type        VARCHAR(50) NOT NULL,
    title       VARCHAR(200) NOT NULL,
    body        TEXT,
    data        JSONB,
    is_read     BOOLEAN DEFAULT FALSE,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_alerts_user ON alerts(user_id, is_read, created_at DESC);

CREATE TABLE analytics_events (
    id          BIGSERIAL PRIMARY KEY,
    user_id     UUID,
    agent_id    UUID,
    event_type  VARCHAR(100) NOT NULL,
    properties  JSONB,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_analytics_type ON analytics_events(event_type, created_at DESC);
