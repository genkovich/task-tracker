ALTER TABLE users ADD COLUMN google_id VARCHAR(255) UNIQUE;
ALTER TABLE users
    ADD COLUMN google_access_token TEXT,
    ADD COLUMN google_refresh_token TEXT,
    ADD COLUMN google_token_expiry TIMESTAMPTZ;
