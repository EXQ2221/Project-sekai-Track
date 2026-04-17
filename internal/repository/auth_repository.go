package repository

import (
	"context"
	"time"

	"Project_sekai_search/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SessionRepository interface {
	WithTx(tx *gorm.DB) SessionRepository
	Create(ctx context.Context, session *model.Session) error
	Update(ctx context.Context, session *model.Session) error
	GetBySessionID(ctx context.Context, sessionID string) (*model.Session, error)
	GetBySessionIDForUpdate(ctx context.Context, sessionID string) (*model.Session, error)
	ListByUserID(ctx context.Context, userID int64) ([]model.Session, error)
	RevokeActiveByUserID(ctx context.Context, userID int64, reason string, revokedAt time.Time) error
}

type RefreshTokenRepository interface {
	WithTx(tx *gorm.DB) RefreshTokenRepository
	Create(ctx context.Context, token *model.RefreshToken) error
	Update(ctx context.Context, token *model.RefreshToken) error
	GetByTokenHashForUpdate(ctx context.Context, tokenHash string) (*model.RefreshToken, error)
	RevokeActiveBySessionID(ctx context.Context, sessionID string, reason string, revokedAt time.Time) error
	RevokeActiveByUserID(ctx context.Context, userID int64, reason string, revokedAt time.Time) error
}

type SecurityEventRepository interface {
	WithTx(tx *gorm.DB) SecurityEventRepository
	Create(ctx context.Context, event *model.SecurityEvent) error
}

type sessionRepo struct {
	db *gorm.DB
}

func NewSessionRepo(db *gorm.DB) SessionRepository {
	return &sessionRepo{db: db}
}

func (r *sessionRepo) WithTx(tx *gorm.DB) SessionRepository {
	return &sessionRepo{db: tx}
}

func (r *sessionRepo) Create(ctx context.Context, session *model.Session) error {
	return r.db.WithContext(ctx).Create(session).Error
}

func (r *sessionRepo) Update(ctx context.Context, session *model.Session) error {
	return r.db.WithContext(ctx).Save(session).Error
}

func (r *sessionRepo) GetBySessionID(ctx context.Context, sessionID string) (*model.Session, error) {
	var session model.Session
	if err := r.db.WithContext(ctx).Where("session_id = ?", sessionID).First(&session).Error; err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *sessionRepo) GetBySessionIDForUpdate(ctx context.Context, sessionID string) (*model.Session, error) {
	var session model.Session
	if err := r.db.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("session_id = ?", sessionID).
		First(&session).Error; err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *sessionRepo) ListByUserID(ctx context.Context, userID int64) ([]model.Session, error) {
	var sessions []model.Session
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("last_seen_at desc, created_at desc").
		Find(&sessions).Error; err != nil {
		return nil, err
	}
	return sessions, nil
}

func (r *sessionRepo) RevokeActiveByUserID(ctx context.Context, userID int64, reason string, revokedAt time.Time) error {
	return r.db.WithContext(ctx).
		Model(&model.Session{}).
		Where("user_id = ? AND status = ?", userID, "active").
		Updates(map[string]any{
			"status":        "revoked",
			"revoked_at":    revokedAt,
			"revoke_reason": reason,
		}).Error
}

type refreshTokenRepo struct {
	db *gorm.DB
}

func NewRefreshTokenRepo(db *gorm.DB) RefreshTokenRepository {
	return &refreshTokenRepo{db: db}
}

func (r *refreshTokenRepo) WithTx(tx *gorm.DB) RefreshTokenRepository {
	return &refreshTokenRepo{db: tx}
}

func (r *refreshTokenRepo) Create(ctx context.Context, token *model.RefreshToken) error {
	return r.db.WithContext(ctx).Create(token).Error
}

func (r *refreshTokenRepo) Update(ctx context.Context, token *model.RefreshToken) error {
	return r.db.WithContext(ctx).Save(token).Error
}

func (r *refreshTokenRepo) GetByTokenHashForUpdate(ctx context.Context, tokenHash string) (*model.RefreshToken, error) {
	var token model.RefreshToken
	if err := r.db.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("token_hash = ?", tokenHash).
		First(&token).Error; err != nil {
		return nil, err
	}
	return &token, nil
}

func (r *refreshTokenRepo) RevokeActiveBySessionID(ctx context.Context, sessionID string, reason string, revokedAt time.Time) error {
	return r.db.WithContext(ctx).
		Model(&model.RefreshToken{}).
		Where("session_id = ? AND status = ?", sessionID, "active").
		Updates(map[string]any{
			"status":        "revoked",
			"revoked_at":    revokedAt,
			"revoke_reason": reason,
		}).Error
}

func (r *refreshTokenRepo) RevokeActiveByUserID(ctx context.Context, userID int64, reason string, revokedAt time.Time) error {
	return r.db.WithContext(ctx).
		Model(&model.RefreshToken{}).
		Where("user_id = ? AND status = ?", userID, "active").
		Updates(map[string]any{
			"status":        "revoked",
			"revoked_at":    revokedAt,
			"revoke_reason": reason,
		}).Error
}

type securityEventRepo struct {
	db *gorm.DB
}

func NewSecurityEventRepo(db *gorm.DB) SecurityEventRepository {
	return &securityEventRepo{db: db}
}

func (r *securityEventRepo) WithTx(tx *gorm.DB) SecurityEventRepository {
	return &securityEventRepo{db: tx}
}

func (r *securityEventRepo) Create(ctx context.Context, event *model.SecurityEvent) error {
	return r.db.WithContext(ctx).Create(event).Error
}
