-- 005_create_users_table.sql
-- Stores authenticated users from OAuth providers (GitHub, Google, Dev).

CREATE TABLE IF NOT EXISTS users (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email            VARCHAR(255) UNIQUE NOT NULL,
    name             VARCHAR(255) NOT NULL,
    provider         VARCHAR(50) NOT NULL, -- 'github' | 'google' | 'dev'
    provider_user_id VARCHAR(255) NOT NULL,
    avatar_url       TEXT,
    created_at       TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at       TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_users_provider ON users(provider, provider_user_id);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
