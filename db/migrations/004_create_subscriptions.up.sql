CREATE TABLE subscriptions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id),
    parcel_id       UUID NOT NULL REFERENCES parcels(id),

    plan            VARCHAR(20) NOT NULL,
    status          VARCHAR(20) DEFAULT 'active',
    amount_per_cycle NUMERIC(10,2) NOT NULL,

    razorpay_subscription_id VARCHAR(100),
    current_period_start TIMESTAMPTZ,
    current_period_end   TIMESTAMPTZ,

    visits_used_this_period INTEGER DEFAULT 0,
    on_demand_visits_remaining INTEGER DEFAULT 0,

    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_subs_active ON subscriptions(status) WHERE status = 'active';
CREATE INDEX idx_subs_user ON subscriptions(user_id);
CREATE INDEX idx_subs_parcel ON subscriptions(parcel_id);
