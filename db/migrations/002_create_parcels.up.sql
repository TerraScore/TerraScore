CREATE TABLE parcels (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    label           VARCHAR(200),
    survey_number   VARCHAR(50),
    village         VARCHAR(100),
    taluk           VARCHAR(100),
    district        VARCHAR(100) NOT NULL,
    state           VARCHAR(100) NOT NULL,
    state_code      VARCHAR(10) NOT NULL,
    pin_code        VARCHAR(6),

    boundary        GEOMETRY(POLYGON, 4326) NOT NULL,
    centroid        GEOMETRY(POINT, 4326) GENERATED ALWAYS AS (ST_Centroid(boundary)) STORED,
    area_sqm        REAL GENERATED ALWAYS AS (ST_Area(boundary::geography)) STORED,

    land_type       VARCHAR(30),
    registered_area_sqm REAL,
    title_deed_s3_key TEXT,

    status          VARCHAR(20) DEFAULT 'active',
    monitoring_since TIMESTAMPTZ DEFAULT NOW(),
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_parcels_user ON parcels(user_id);
CREATE INDEX idx_parcels_boundary ON parcels USING GIST(boundary);
CREATE INDEX idx_parcels_centroid ON parcels USING GIST(centroid);
