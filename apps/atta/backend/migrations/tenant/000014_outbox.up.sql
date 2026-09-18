CREATE TABLE outbox_events (
    id TEXT PRIMARY KEY,
    tenant_slug TEXT NOT NULL,
    source TEXT NOT NULL,
    detail_type TEXT NOT NULL,
    payload JSONB NOT NULL,
    correlation_id TEXT,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'processed', 'failed')),
    attempts INT NOT NULL DEFAULT 0,
    max_attempts INT NOT NULL DEFAULT 10,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMPTZ,
    last_error TEXT
);

CREATE TABLE outbox_jobs (
    id TEXT PRIMARY KEY,
    tenant_slug TEXT NOT NULL,
    job_type TEXT NOT NULL,
    payload JSONB NOT NULL,
    correlation_id TEXT,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'processed', 'failed')),
    attempts INT NOT NULL DEFAULT 0,
    max_attempts INT NOT NULL DEFAULT 10,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMPTZ,
    last_error TEXT
);

CREATE INDEX idx_outbox_events_pending ON outbox_events (created_at) WHERE status = 'pending';
CREATE INDEX idx_outbox_jobs_pending ON outbox_jobs (created_at) WHERE status = 'pending';
