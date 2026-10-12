CREATE TABLE IF NOT EXISTS api_keys (
    key_hash VARCHAR(255) PRIMARY KEY, -- We store the hash of the key, not the raw key!
    user_id UUID NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    is_active BOOLEAN DEFAULT TRUE
);

-- Index for fast user lookups
CREATE INDEX IF NOT EXISTS idx_api_keys_user_id ON api_keys(user_id);
