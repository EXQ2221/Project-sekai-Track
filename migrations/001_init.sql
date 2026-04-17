
CREATE TABLE IF NOT EXISTS users (
                                     id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
                                     username VARCHAR(64) NOT NULL,
    email VARCHAR(128) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,

    created_at TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    PRIMARY KEY (id),
    UNIQUE KEY uk_users_username (username),
    UNIQUE KEY uk_users_email (email)
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;


CREATE TABLE IF NOT EXISTS songs (
                                     id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
                                     title VARCHAR(255) NOT NULL,
    aliases JSON NULL,

    created_at TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    PRIMARY KEY (id),
    UNIQUE KEY uk_songs_title (title)
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;


CREATE TABLE IF NOT EXISTS charts (
                                      id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
                                      song_id BIGINT UNSIGNED NOT NULL,
                                      difficulty TINYINT UNSIGNED NOT NULL,   
                                      level INT NOT NULL,
                                      constant DECIMAL(5,2) NOT NULL,

    created_at TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    PRIMARY KEY (id),
    UNIQUE KEY uk_charts_song_diff (song_id, difficulty),
    CONSTRAINT fk_charts_song FOREIGN KEY (song_id) REFERENCES songs(id)
                                                        ON DELETE CASCADE ON UPDATE CASCADE
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;


CREATE TABLE IF NOT EXISTS records (
                                       id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
                                       user_id BIGINT UNSIGNED NOT NULL,
                                       chart_id BIGINT UNSIGNED NOT NULL,
                                       grade TINYINT UNSIGNED NOT NULL DEFAULT 0,

                                       created_at TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP,
                                       updated_at TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

                                       PRIMARY KEY (id),
    UNIQUE KEY uk_records_user_chart (user_id, chart_id),
    KEY idx_records_user (user_id),
    KEY idx_records_chart (chart_id),

    CONSTRAINT fk_records_user FOREIGN KEY (user_id) REFERENCES users(id)
    ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT fk_records_chart FOREIGN KEY (chart_id) REFERENCES charts(id)
    ON DELETE CASCADE ON UPDATE CASCADE
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
