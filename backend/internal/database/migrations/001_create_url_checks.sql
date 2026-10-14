CREATE TABLE IF NOT EXISTS url_checks (
    id BIGSERIAL PRIMARY KEY,
    url TEXT NOT NULL,
    available BOOLEAN NOT NULL,
    status_code INTEGER,
    response_time_ms BIGINT,
    error_message TEXT,
    checked_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_url_checks_checked_at
    ON url_checks (checked_at DESC);