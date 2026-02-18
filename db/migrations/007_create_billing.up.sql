CREATE TABLE transactions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id),
    subscription_id UUID,
    type            VARCHAR(20) NOT NULL,
    amount          NUMERIC(10,2) NOT NULL,
    status          VARCHAR(20) DEFAULT 'pending',
    razorpay_payment_id VARCHAR(100),
    razorpay_order_id   VARCHAR(100),
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_transactions_user ON transactions(user_id);

CREATE TABLE agent_payouts (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id        UUID NOT NULL REFERENCES agents(id),
    period_start    DATE NOT NULL,
    period_end      DATE NOT NULL,
    total_jobs      INTEGER,
    gross_amount    NUMERIC(10,2),
    platform_commission NUMERIC(10,2),
    tds_deducted    NUMERIC(10,2),
    net_amount      NUMERIC(10,2),
    status          VARCHAR(20) DEFAULT 'pending',
    razorpay_payout_id VARCHAR(100),
    failure_reason  TEXT,
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_payouts_agent ON agent_payouts(agent_id);
