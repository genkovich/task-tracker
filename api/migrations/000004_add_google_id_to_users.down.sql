ALTER TABLE users
    DROP COLUMN IF EXISTS google_token_expiry,
    DROP COLUMN IF EXISTS google_refresh_token,
    DROP COLUMN IF EXISTS google_access_token,
    DROP COLUMN IF EXISTS google_id;
