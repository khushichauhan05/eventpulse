CREATE TABLE IF NOT EXISTS alerts (
    id         SERIAL PRIMARY KEY,
    event_id   TEXT        NOT NULL DEFAULT '',
    user_id    TEXT        NOT NULL,
    risk_score INT         NOT NULL,
    message    TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_alerts_event_id UNIQUE (event_id)
);

CREATE INDEX IF NOT EXISTS idx_alerts_user_id    ON alerts (user_id);
CREATE INDEX IF NOT EXISTS idx_alerts_created_at ON alerts (created_at DESC);
