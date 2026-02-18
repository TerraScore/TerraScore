CREATE TABLE agents (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    full_name       VARCHAR(200) NOT NULL,
    phone           VARCHAR(15) NOT NULL UNIQUE,
    email           VARCHAR(255),
    date_of_birth   DATE,

    aadhaar_hash    VARCHAR(64),
    aadhaar_verified BOOLEAN DEFAULT FALSE,

    home_location   GEOMETRY(POINT, 4326),
    last_known_location GEOMETRY(POINT, 4326),
    last_location_at TIMESTAMPTZ,
    preferred_radius_km INTEGER DEFAULT 25,
    state_code      VARCHAR(10),
    district_code   VARCHAR(10),

    status          VARCHAR(25) DEFAULT 'pending_verification',
    tier            VARCHAR(20) DEFAULT 'basic',
    vehicle_type    VARCHAR(20),

    total_jobs_completed INTEGER DEFAULT 0,
    avg_rating      NUMERIC(3,2) DEFAULT 0.00,
    completion_rate NUMERIC(5,4) DEFAULT 1.0000,
    qa_pass_rate    NUMERIC(5,4) DEFAULT 1.0000,
    last_job_completed_at TIMESTAMPTZ,

    bank_account_enc TEXT,
    bank_ifsc       VARCHAR(11),
    upi_id          VARCHAR(100),
    wallet_balance  NUMERIC(10,2) DEFAULT 0.00,

    certifications  TEXT[] DEFAULT ARRAY['basic_survey'],

    fcm_token       TEXT,
    device_id       VARCHAR(100),
    app_version     VARCHAR(20),

    is_online       BOOLEAN DEFAULT FALSE,
    available_days  TEXT[] DEFAULT ARRAY['mon','tue','wed','thu','fri','sat'],
    available_start TIME DEFAULT '08:00',
    available_end   TIME DEFAULT '18:00',

    keycloak_id     VARCHAR(100) UNIQUE,

    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_agents_location ON agents USING GIST(last_known_location);
CREATE INDEX idx_agents_matching ON agents(status, is_online, tier)
    WHERE status = 'active' AND is_online = TRUE;
CREATE INDEX idx_agents_phone ON agents(phone);
CREATE INDEX idx_agents_keycloak ON agents(keycloak_id) WHERE keycloak_id IS NOT NULL;
