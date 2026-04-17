package dto

type RegisterRequest struct {
	Username string `json:"username" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type LoginRequest struct {
	Username   string `json:"username" binding:"required"`
	Password   string `json:"password" binding:"required"`
	DeviceID   string `json:"device_id"`
	DeviceName string `json:"device_name"`
}

type ChangePassRequest struct {
	OldPass string `json:"old_pass" binding:"required"`
	NewPass string `json:"new_pass" binding:"required"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
	DeviceID     string `json:"device_id"`
}

type LogoutAllRequest struct {
	Password string `json:"password" binding:"required"`
}

type RevokeSessionRequest struct {
	SessionID string `json:"session_id" binding:"required"`
	Password  string `json:"password" binding:"required"`
}

type TokenPair struct {
	AccessToken      string `json:"access_token"`
	RefreshToken     string `json:"refresh_token"`
	SessionID        string `json:"session_id"`
	AccessExpiresAt  int64  `json:"access_expires_at"`
	RefreshExpiresAt int64  `json:"refresh_expires_at"`
}

type SessionInfo struct {
	SessionID  string `json:"session_id"`
	DeviceID   string `json:"device_id"`
	DeviceName string `json:"device_name"`
	UserAgent  string `json:"user_agent"`
	LoginIP    string `json:"login_ip"`
	LastIP     string `json:"last_ip"`
	Status     string `json:"status"`
	Current    bool   `json:"current"`
	CreatedAt  int64  `json:"created_at"`
	LastSeenAt int64  `json:"last_seen_at"`
}
