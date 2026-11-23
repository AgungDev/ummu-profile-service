CREATE TABLE user_profiles (
    user_id UUID PRIMARY KEY,
    full_name VARCHAR(128) NOT NULL DEFAULT '',
    phone_number VARCHAR(32),
    avatar_key TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE user_addresses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES user_profiles(user_id) ON DELETE CASCADE,
    label VARCHAR(32) NOT NULL DEFAULT 'home',
    address_line TEXT NOT NULL,
    city VARCHAR(128) NOT NULL,
    province VARCHAR(128) NOT NULL,
    postal_code VARCHAR(16),
    country VARCHAR(2) NOT NULL DEFAULT 'ID',
    is_primary BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_user_addresses_user_id ON user_addresses(user_id);
CREATE UNIQUE INDEX uq_user_addresses_primary
    ON user_addresses(user_id) WHERE is_primary = TRUE;
