ALTER TABLE users
    ADD COLUMN IF NOT EXISTS email VARCHAR(128) NULL;

UPDATE users
SET email = CONCAT('user_', id, '@example.com')
WHERE email IS NULL OR email = '';

ALTER TABLE users
    MODIFY COLUMN email VARCHAR(128) NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uk_users_email ON users (email);

