CREATE TABLE survey_jobs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    parcel_id       UUID NOT NULL REFERENCES parcels(id),
    subscription_id UUID REFERENCES subscriptions(id),
    user_id         UUID NOT NULL,

    survey_type     VARCHAR(30) NOT NULL,
    priority        VARCHAR(10) DEFAULT 'normal',
    deadline        TIMESTAMPTZ NOT NULL,
    trigger         VARCHAR(20) DEFAULT 'scheduled',

    status          VARCHAR(25) DEFAULT 'pending_assignment',

    assigned_agent_id UUID REFERENCES agents(id),
    assigned_at     TIMESTAMPTZ,
    cascade_round   INTEGER DEFAULT 0,
    total_offers_sent INTEGER DEFAULT 0,

    agent_arrived_at TIMESTAMPTZ,
    survey_started_at TIMESTAMPTZ,
    survey_submitted_at TIMESTAMPTZ,
    completed_at    TIMESTAMPTZ,

    arrival_location GEOMETRY(POINT, 4326),
    arrival_distance_m REAL,

    base_payout     NUMERIC(8,2),
    distance_bonus  NUMERIC(8,2) DEFAULT 0,
    urgency_bonus   NUMERIC(8,2) DEFAULT 0,
    total_payout    NUMERIC(8,2),
    payout_status   VARCHAR(20) DEFAULT 'pending',

    landowner_rating NUMERIC(2,1),
    qa_score        NUMERIC(5,4),
    qa_status       VARCHAR(20) DEFAULT 'pending',
    qa_notes        TEXT,

    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_jobs_status ON survey_jobs(status);
CREATE INDEX idx_jobs_parcel ON survey_jobs(parcel_id, created_at DESC);
CREATE INDEX idx_jobs_agent ON survey_jobs(assigned_agent_id, status);
CREATE INDEX idx_jobs_pending ON survey_jobs(deadline) WHERE status IN ('pending_assignment', 'offered');

CREATE TABLE job_offers (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id          UUID NOT NULL REFERENCES survey_jobs(id),
    agent_id        UUID NOT NULL REFERENCES agents(id),
    cascade_round   INTEGER NOT NULL,
    offer_rank      INTEGER NOT NULL,

    distance_km     REAL,
    match_score     NUMERIC(5,4),

    status          VARCHAR(20) DEFAULT 'sent',
    sent_at         TIMESTAMPTZ DEFAULT NOW(),
    responded_at    TIMESTAMPTZ,
    expires_at      TIMESTAMPTZ NOT NULL,
    decline_reason  VARCHAR(100)
);

CREATE INDEX idx_offers_job ON job_offers(job_id);
CREATE INDEX idx_offers_agent ON job_offers(agent_id, status);
