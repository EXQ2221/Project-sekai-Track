CREATE TABLE IF NOT EXISTS sessions (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    session_id VARCHAR(64) NOT NULL,
    user_id BIGINT UNSIGNED NOT NULL,
    status VARCHAR(32) NOT NULL,
    device_id VARCHAR(128) NULL,
    device_name VARCHAR(128) NULL,
    user_agent VARCHAR(512) NULL,
    browser_name VARCHAR(64) NULL,
    browser_version VARCHAR(64) NULL,
    os_name VARCHAR(64) NULL,
    device_type VARCHAR(32) NULL,
    browser_key VARCHAR(191) NULL,
    login_ip VARCHAR(64) NULL,
    last_ip VARCHAR(64) NULL,
    last_seen_at DATETIME NOT NULL,
    current_access_jti VARCHAR(64) NULL,
    current_access_expires DATETIME NOT NULL,
    revoked_at DATETIME NULL,
    revoke_reason VARCHAR(128) NULL,
    created_at TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uk_sessions_session_id (session_id),
    KEY idx_sessions_user_id (user_id),
    KEY idx_sessions_status (status),
    KEY idx_sessions_browser_key (browser_key),
    KEY idx_sessions_access_jti (current_access_jti),
    CONSTRAINT fk_sessions_user FOREIGN KEY (user_id) REFERENCES users(id)
        ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS refresh_tokens (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    session_id VARCHAR(64) NOT NULL,
    user_id BIGINT UNSIGNED NOT NULL,
    token_hash CHAR(64) NOT NULL,
    status VARCHAR(32) NOT NULL,
    expires_at DATETIME NOT NULL,
    used_at DATETIME NULL,
    revoked_at DATETIME NULL,
    revoke_reason VARCHAR(128) NULL,
    rotated_to CHAR(64) NULL,
    last_used_ip VARCHAR(64) NULL,
    last_used_user_agent VARCHAR(512) NULL,
    created_at TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uk_refresh_tokens_hash (token_hash),
    KEY idx_refresh_tokens_session_id (session_id),
    KEY idx_refresh_tokens_user_id (user_id),
    KEY idx_refresh_tokens_status (status),
    CONSTRAINT fk_refresh_tokens_user FOREIGN KEY (user_id) REFERENCES users(id)
        ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT fk_refresh_tokens_session FOREIGN KEY (session_id) REFERENCES sessions(session_id)
        ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS security_events (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    user_id BIGINT UNSIGNED NOT NULL,
    session_id VARCHAR(64) NULL,
    event_type VARCHAR(64) NOT NULL,
    ip VARCHAR(64) NULL,
    device_id VARCHAR(128) NULL,
    user_agent VARCHAR(512) NULL,
    detail TEXT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY idx_security_events_user_id (user_id),
    KEY idx_security_events_session_id (session_id),
    KEY idx_security_events_event_type (event_type),
    CONSTRAINT fk_security_events_user FOREIGN KEY (user_id) REFERENCES users(id)
        ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

