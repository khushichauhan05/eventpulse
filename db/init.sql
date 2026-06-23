CREATE TABLE IF NOT EXISTS alerts (
    id          SERIAL PRIMARY KEY,
    event_id    TEXT           NOT NULL DEFAULT '',
    user_id     TEXT           NOT NULL,
    risk_score  INT            NOT NULL,
    confidence  NUMERIC(5,4)   NOT NULL DEFAULT 0,
    message     TEXT           NOT NULL,
    ml_scored   BOOLEAN        NOT NULL DEFAULT FALSE,
    explanation JSONB          NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_alerts_event_id UNIQUE (event_id)
);

CREATE INDEX IF NOT EXISTS idx_alerts_user_id    ON alerts (user_id);
CREATE INDEX IF NOT EXISTS idx_alerts_created_at ON alerts (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_alerts_risk_score ON alerts (risk_score DESC);
CREATE INDEX IF NOT EXISTS idx_alerts_explanation ON alerts USING GIN (explanation);
