-- 003_create_webhook_deliveries.sql
-- Per-attempt delivery records. One event can have many delivery attempts.

CREATE TABLE IF NOT EXISTS webhook_deliveries (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id         UUID NOT NULL REFERENCES webhook_events(id) ON DELETE CASCADE,
    attempt_number   INT NOT NULL,
    http_status      INT,
    response_body    TEXT,
    execution_time_ms INT,
    -- SUCCESS | FAILED | CIRCUIT_BROKEN | DLQ
    status           VARCHAR(50) NOT NULL,
    error_message    TEXT,
    delivered_at     TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_deliveries_event_id ON webhook_deliveries(event_id);
CREATE INDEX IF NOT EXISTS idx_deliveries_status ON webhook_deliveries(status);
CREATE INDEX IF NOT EXISTS idx_deliveries_delivered_at ON webhook_deliveries(delivered_at DESC);
