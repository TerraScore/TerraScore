-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS postgis;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE TABLE users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    phone           VARCHAR(15) NOT NULL UNIQUE,
    email           VARCHAR(255),
    full_name       VARCHAR(200) NOT NULL,
    role            VARCHAR(20) NOT NULL DEFAULT 'landowner',
    avatar_url      TEXT,

    state_code      VARCHAR(10),
    district_code   VARCHAR(10),
    city            VARCHAR(100),

    status          VARCHAR(20) DEFAULT 'active',
    phone_verified  BOOLEAN DEFAULT TRUE,
    language        VARCHAR(5) DEFAULT 'en',

    notification_prefs JSONB DEFAULT '{"email": true, "sms": true, "push": true}',

    keycloak_id     VARCHAR(100) UNIQUE,

    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_users_phone ON users(phone);
CREATE INDEX idx_users_keycloak ON users(keycloak_id) WHERE keycloak_id IS NOT NULL;
