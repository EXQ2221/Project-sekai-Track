package model

import "time"

type User struct {
	ID           uint      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Username     string    `gorm:"column:username;size:64;uniqueIndex;not null" json:"username"`
	Email        string    `gorm:"column:email;size:128;uniqueIndex;not null" json:"email"`
	PasswordHash string    `gorm:"column:password_hash;size:255;not null" json:"-"`
	TokenVersion int       `gorm:"column:token_version;not null;default:0" json:"-"`
	AvatarURL    string    `gorm:"column:avatar_url;size:255" json:"avatar_url"`
	Profile      string    `gorm:"column:profile;size:255" json:"profile"`
	Character    string    `gorm:"column:character;size:255" json:"character"`
	B30Avg       float64   `gorm:"column:b30_avg;type:decimal(6,2);not null;default:0" json:"b30_avg"`
	B30AvgConst  float64   `gorm:"column:b30_avg_const;type:decimal(6,2);not null;default:0" json:"b30_avg_const"`
	CreatedAt    time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (User) TableName() string {
	return "users"
}

type Session struct {
	ID                   int64      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	SessionID            string     `gorm:"column:session_id;size:64;uniqueIndex;not null" json:"session_id"`
	UserID               int64      `gorm:"column:user_id;index;not null" json:"user_id"`
	Status               string     `gorm:"column:status;size:32;index;not null" json:"status"`
	DeviceID             string     `gorm:"column:device_id;size:128;index" json:"device_id"`
	DeviceName           string     `gorm:"column:device_name;size:128" json:"device_name"`
	UserAgent            string     `gorm:"column:user_agent;size:512" json:"user_agent"`
	BrowserName          string     `gorm:"column:browser_name;size:64" json:"browser_name"`
	BrowserVersion       string     `gorm:"column:browser_version;size:64" json:"browser_version"`
	OSName               string     `gorm:"column:os_name;size:64" json:"os_name"`
	DeviceType           string     `gorm:"column:device_type;size:32" json:"device_type"`
	BrowserKey           string     `gorm:"column:browser_key;size:191;index" json:"browser_key"`
	LoginIP              string     `gorm:"column:login_ip;size:64" json:"login_ip"`
	LastIP               string     `gorm:"column:last_ip;size:64" json:"last_ip"`
	LastSeenAt           time.Time  `gorm:"column:last_seen_at" json:"last_seen_at"`
	CurrentAccessJTI     string     `gorm:"column:current_access_jti;size:64;index" json:"current_access_jti"`
	CurrentAccessExpires time.Time  `gorm:"column:current_access_expires" json:"current_access_expires"`
	RevokedAt            *time.Time `gorm:"column:revoked_at" json:"revoked_at,omitempty"`
	RevokeReason         string     `gorm:"column:revoke_reason;size:128" json:"revoke_reason"`
	CreatedAt            time.Time  `gorm:"column:created_at" json:"created_at"`
	UpdatedAt            time.Time  `gorm:"column:updated_at" json:"updated_at"`
}

func (Session) TableName() string {
	return "sessions"
}

type RefreshToken struct {
	ID                int64      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	SessionID         string     `gorm:"column:session_id;size:64;index;not null" json:"session_id"`
	UserID            int64      `gorm:"column:user_id;index;not null" json:"user_id"`
	TokenHash         string     `gorm:"column:token_hash;size:64;uniqueIndex;not null" json:"token_hash"`
	Status            string     `gorm:"column:status;size:32;index;not null" json:"status"`
	ExpiresAt         time.Time  `gorm:"column:expires_at" json:"expires_at"`
	UsedAt            *time.Time `gorm:"column:used_at" json:"used_at,omitempty"`
	RevokedAt         *time.Time `gorm:"column:revoked_at" json:"revoked_at,omitempty"`
	RevokeReason      string     `gorm:"column:revoke_reason;size:128" json:"revoke_reason"`
	RotatedTo         string     `gorm:"column:rotated_to;size:64" json:"rotated_to"`
	LastUsedIP        string     `gorm:"column:last_used_ip;size:64" json:"last_used_ip"`
	LastUsedUserAgent string     `gorm:"column:last_used_user_agent;size:512" json:"last_used_user_agent"`
	CreatedAt         time.Time  `gorm:"column:created_at" json:"created_at"`
	UpdatedAt         time.Time  `gorm:"column:updated_at" json:"updated_at"`
}

func (RefreshToken) TableName() string {
	return "refresh_tokens"
}

type SecurityEvent struct {
	ID        int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	UserID    int64     `gorm:"column:user_id;index;not null" json:"user_id"`
	SessionID string    `gorm:"column:session_id;size:64;index" json:"session_id"`
	EventType string    `gorm:"column:event_type;size:64;index;not null" json:"event_type"`
	IP        string    `gorm:"column:ip;size:64" json:"ip"`
	DeviceID  string    `gorm:"column:device_id;size:128" json:"device_id"`
	UserAgent string    `gorm:"column:user_agent;size:512" json:"user_agent"`
	Detail    string    `gorm:"column:detail;type:text" json:"detail"`
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
}

func (SecurityEvent) TableName() string {
	return "security_events"
}
