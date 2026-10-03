-- 002_create_webhook_events.sql
-- Master log of every webhook event ingested by the API.
-- idempotency_key enforces exactly-once processing.

CREATE TABLE IF NOT EXISTS webhook_events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    endpoint_id     UUID NOT NULL REFERENCES webhook_endpoints(id) ON DELETE CASCADE,
    event_type      VARCHAR(100) NOT NULL,
    payload         JSONB NOT NULL,
    idempotency_key VARCHAR(255) UNIQUE NOT NULL,  -- Client-supplied dedup key
    status          VARCHAR(50) NOT NULL DEFAULT 'PENDING', -- PENDING | DELIVERED | FAILED | DLQ
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_events_endpoint_id ON webhook_events(endpoint_id);
CREATE INDEX IF NOT EXISTS idx_events_status ON webhook_events(status);
CREATE INDEX IF NOT EXISTS idx_events_created_at ON webhook_events(created_at DESC);
