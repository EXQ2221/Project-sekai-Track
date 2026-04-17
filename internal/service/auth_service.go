package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"Project_sekai_search/internal/dto"
	"Project_sekai_search/internal/model"
	"Project_sekai_search/internal/pkg/browser"
	"Project_sekai_search/internal/pkg/errcode"
	"Project_sekai_search/internal/pkg/token"
	"Project_sekai_search/internal/pkg/utils"
	"Project_sekai_search/internal/repository"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const (
	sessionStatusActive  = "active"
	sessionStatusRevoked = "revoked"

	refreshStatusActive  = "active"
	refreshStatusUsed    = "used"
	refreshStatusRevoked = "revoked"
)

type AuthIdentity struct {
	UserID    uint
	Username  string
	SessionID string
	TokenID   string
}

type AuthService struct {
	userRepo    repository.UserRepository
	sessionRepo repository.SessionRepository
	refreshRepo repository.RefreshTokenRepository
	eventRepo   repository.SecurityEventRepository
	db          *gorm.DB
}

func NewAuthService(
	userRepo repository.UserRepository,
	sessionRepo repository.SessionRepository,
	refreshRepo repository.RefreshTokenRepository,
	eventRepo repository.SecurityEventRepository,
	db *gorm.DB,
) *AuthService {
	return &AuthService{
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
		refreshRepo: refreshRepo,
		eventRepo:   eventRepo,
		db:          db,
	}
}

func (s *AuthService) Login(ctx context.Context, req dto.LoginRequest, ip, userAgent string) (*dto.TokenPair, *model.User, string, error) {
	user, err := s.userRepo.FindUserByUsername(ctx, strings.TrimSpace(req.Username))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, "", errcode.ErrUsernameIncorrect
		}
		return nil, nil, "", errcode.ErrInternal
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, nil, "", errcode.ErrPasswordIncorrect
	}

	sessionID, err := utils.NewSID()
	if err != nil {
		return nil, nil, "", errcode.ErrInternal
	}
	refreshTokenValue, err := token.GenerateRefreshToken()
	if err != nil {
		return nil, nil, "", errcode.ErrInternal
	}

	accessToken, accessJTI, accessExpiresAt, err := token.GenerateToken(user.Username, user.ID, sessionID)
	if err != nil {
		return nil, nil, "", errcode.ErrInternal
	}

	now := time.Now()
	refreshExpiresAt := now.Add(token.RefreshTTL())
	deviceID := strings.TrimSpace(req.DeviceID)
	if deviceID == "" {
		deviceID = sessionID
	}
	deviceName := strings.TrimSpace(req.DeviceName)
	if deviceName == "" {
		deviceName = "web-client"
	}

	browserInfo := browser.Parse(userAgent)
	session := &model.Session{
		SessionID:            sessionID,
		UserID:               int64(user.ID),
		Status:               sessionStatusActive,
		DeviceID:             deviceID,
		DeviceName:           deviceName,
		UserAgent:            userAgent,
		BrowserName:          browserInfo.BrowserName,
		BrowserVersion:       browserInfo.BrowserVersion,
		OSName:               browserInfo.OSName,
		DeviceType:           browserInfo.DeviceType,
		BrowserKey:           browserInfo.Key,
		LoginIP:              ip,
		LastIP:               ip,
		LastSeenAt:           now,
		CurrentAccessJTI:     accessJTI,
		CurrentAccessExpires: accessExpiresAt,
	}

	refreshRecord := &model.RefreshToken{
		SessionID: sessionID,
		UserID:    int64(user.ID),
		TokenHash: utils.HashToken(refreshTokenValue),
		Status:    refreshStatusActive,
		ExpiresAt: refreshExpiresAt,
	}

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		sessionRepo := s.sessionRepo.WithTx(tx)
		refreshRepo := s.refreshRepo.WithTx(tx)
		if err := sessionRepo.Create(ctx, session); err != nil {
			return err
		}
		if err := refreshRepo.Create(ctx, refreshRecord); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, nil, "", errcode.ErrInternal
	}

	return &dto.TokenPair{
		AccessToken:      accessToken,
		RefreshToken:     refreshTokenValue,
		SessionID:        sessionID,
		AccessExpiresAt:  accessExpiresAt.Unix(),
		RefreshExpiresAt: refreshExpiresAt.Unix(),
	}, user, deviceID, nil
}

func (s *AuthService) ValidateAccessToken(ctx context.Context, accessToken string) (*AuthIdentity, error) {
	claims, err := token.ValidateToken(strings.TrimSpace(accessToken))
	if err != nil {
		return nil, errcode.ErrUnauthorized
	}

	session, err := s.sessionRepo.GetBySessionID(ctx, claims.SessionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errcode.ErrUnauthorized
		}
		return nil, errcode.ErrInternal
	}

	if session.Status != sessionStatusActive || session.RevokedAt != nil {
		return nil, errcode.ErrSessionRevoked
	}
	if session.UserID != int64(claims.UserID) || session.CurrentAccessJTI != claims.TokenID {
		return nil, errcode.ErrUnauthorized
	}

	return &AuthIdentity{
		UserID:    claims.UserID,
		Username:  claims.Username,
		SessionID: claims.SessionID,
		TokenID:   claims.TokenID,
	}, nil
}

func (s *AuthService) Refresh(ctx context.Context, req dto.RefreshRequest, ip, userAgent string) (*dto.TokenPair, error) {
	refreshTokenValue := strings.TrimSpace(req.RefreshToken)
	if refreshTokenValue == "" {
		return nil, errcode.ErrUnauthorized
	}

	refreshHash := utils.HashToken(refreshTokenValue)
	now := time.Now()
	deviceID := strings.TrimSpace(req.DeviceID)
	currentBrowser := browser.Parse(userAgent)

	var (
		pair     *dto.TokenPair
		finalErr error
	)

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		refreshRepo := s.refreshRepo.WithTx(tx)
		sessionRepo := s.sessionRepo.WithTx(tx)
		eventRepo := s.eventRepo.WithTx(tx)

		record, err := refreshRepo.GetByTokenHashForUpdate(ctx, refreshHash)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				finalErr = errcode.ErrUnauthorized
				return nil
			}
			return err
		}

		session, err := sessionRepo.GetBySessionIDForUpdate(ctx, record.SessionID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				finalErr = errcode.ErrUnauthorized
				return nil
			}
			return err
		}

		if record.Status != refreshStatusActive {
			if err := s.recordEventWithRepo(ctx, eventRepo, record.UserID, record.SessionID, "refresh_token_reuse", ip, deviceID, userAgent, "used or revoked refresh token was presented again"); err != nil {
				return err
			}
			if err := s.revokeAllSessionsWithRepo(ctx, sessionRepo, refreshRepo, record.UserID, "refresh_token_reuse", now); err != nil {
				return err
			}
			finalErr = errcode.ErrRefreshReuse
			return nil
		}

		if now.After(record.ExpiresAt) {
			record.Status = refreshStatusRevoked
			record.RevokedAt = &now
			record.RevokeReason = "expired"
			if err := refreshRepo.Update(ctx, record); err != nil {
				return err
			}
			finalErr = errcode.ErrUnauthorized
			return nil
		}

		if session.Status != sessionStatusActive || session.RevokedAt != nil {
			finalErr = errcode.ErrSessionRevoked
			return nil
		}

		if session.DeviceID != "" && deviceID != "" && session.DeviceID != deviceID {
			if err := s.recordEventWithRepo(ctx, eventRepo, session.UserID, session.SessionID, "device_mismatch", ip, deviceID, userAgent, "refresh attempted from another device id"); err != nil {
				return err
			}
			if err := s.revokeSessionWithRepo(ctx, sessionRepo, refreshRepo, session, "device_mismatch", now); err != nil {
				return err
			}
			finalErr = errcode.ErrDeviceMismatch
			return nil
		}

		if session.BrowserKey != "" && currentBrowser.Key != "" && session.BrowserKey != currentBrowser.Key {
			if err := s.recordEventWithRepo(ctx, eventRepo, session.UserID, session.SessionID, "browser_mismatch", ip, deviceID, userAgent, "refresh attempted from another browser identity"); err != nil {
				return err
			}
			if err := s.revokeSessionWithRepo(ctx, sessionRepo, refreshRepo, session, "browser_mismatch", now); err != nil {
				return err
			}
			finalErr = errcode.ErrDeviceMismatch
			return nil
		}

		if session.LastIP != "" && ip != "" && session.LastIP != ip {
			if err := s.recordEventWithRepo(ctx, eventRepo, session.UserID, session.SessionID, "ip_changed", ip, deviceID, userAgent, "refresh token used from a new ip"); err != nil {
				return err
			}
		}

		var user model.User
		if err := s.userRepo.FindUserByID(ctx, uint(session.UserID), &user); err != nil {
			return err
		}

		newRefreshToken, err := token.GenerateRefreshToken()
		if err != nil {
			return err
		}
		newAccessToken, newAccessJTI, accessExpiresAt, err := token.GenerateToken(user.Username, user.ID, session.SessionID)
		if err != nil {
			return err
		}

		record.Status = refreshStatusUsed
		record.UsedAt = &now
		record.LastUsedIP = ip
		record.LastUsedUserAgent = userAgent
		record.RotatedTo = utils.HashToken(newRefreshToken)
		if err := refreshRepo.Update(ctx, record); err != nil {
			return err
		}

		refreshExpiresAt := now.Add(token.RefreshTTL())
		if err := refreshRepo.Create(ctx, &model.RefreshToken{
			SessionID: session.SessionID,
			UserID:    session.UserID,
			TokenHash: utils.HashToken(newRefreshToken),
			Status:    refreshStatusActive,
			ExpiresAt: refreshExpiresAt,
		}); err != nil {
			return err
		}

		session.LastSeenAt = now
		session.LastIP = ip
		session.UserAgent = userAgent
		if currentBrowser.BrowserName != "" {
			session.BrowserName = currentBrowser.BrowserName
		}
		if currentBrowser.BrowserVersion != "" {
			session.BrowserVersion = currentBrowser.BrowserVersion
		}
		if currentBrowser.OSName != "" {
			session.OSName = currentBrowser.OSName
		}
		if currentBrowser.DeviceType != "" {
			session.DeviceType = currentBrowser.DeviceType
		}
		if currentBrowser.Key != "" {
			session.BrowserKey = currentBrowser.Key
		}
		session.CurrentAccessJTI = newAccessJTI
		session.CurrentAccessExpires = accessExpiresAt
		if err := sessionRepo.Update(ctx, session); err != nil {
			return err
		}

		pair = &dto.TokenPair{
			AccessToken:      newAccessToken,
			RefreshToken:     newRefreshToken,
			SessionID:        session.SessionID,
			AccessExpiresAt:  accessExpiresAt.Unix(),
			RefreshExpiresAt: refreshExpiresAt.Unix(),
		}
		return nil
	})
	if err != nil {
		return nil, errcode.ErrInternal
	}
	if finalErr != nil {
		return nil, finalErr
	}
	return pair, nil
}

func (s *AuthService) Logout(ctx context.Context, accessToken string) error {
	identity, err := s.ValidateAccessToken(ctx, accessToken)
	if err != nil {
		return err
	}
	session, err := s.sessionRepo.GetBySessionID(ctx, identity.SessionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errcode.ErrNotFound
		}
		return errcode.ErrInternal
	}
	return s.revokeSession(ctx, session, "logout")
}

func (s *AuthService) LogoutAll(ctx context.Context, userID uint, password string) error {
	user, err := s.findUser(ctx, userID)
	if err != nil {
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return errcode.ErrPasswordIncorrect
	}
	return s.RevokeAllUserSessions(ctx, userID, "logout_all")
}

func (s *AuthService) ListSessions(ctx context.Context, userID uint, currentSessionID string) ([]dto.SessionInfo, error) {
	sessions, err := s.sessionRepo.ListByUserID(ctx, int64(userID))
	if err != nil {
		return nil, errcode.ErrInternal
	}

	result := make([]dto.SessionInfo, 0, len(sessions))
	for _, session := range sessions {
		if session.Status != sessionStatusActive {
			continue
		}
		result = append(result, dto.SessionInfo{
			SessionID:  session.SessionID,
			DeviceID:   session.DeviceID,
			DeviceName: session.DeviceName,
			UserAgent:  session.UserAgent,
			LoginIP:    session.LoginIP,
			LastIP:     session.LastIP,
			Status:     session.Status,
			Current:    session.SessionID == currentSessionID,
			CreatedAt:  session.CreatedAt.Unix(),
			LastSeenAt: session.LastSeenAt.Unix(),
		})
	}
	return result, nil
}

func (s *AuthService) RevokeSession(ctx context.Context, userID uint, sessionID, password string) error {
	user, err := s.findUser(ctx, userID)
	if err != nil {
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return errcode.ErrPasswordIncorrect
	}

	session, err := s.sessionRepo.GetBySessionID(ctx, strings.TrimSpace(sessionID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errcode.ErrNotFound
		}
		return errcode.ErrInternal
	}
	if session.UserID != int64(userID) {
		return errcode.ErrNotFound
	}
	return s.revokeSession(ctx, session, "revoke_session")
}

func (s *AuthService) RevokeAllUserSessions(ctx context.Context, userID uint, reason string) error {
	now := time.Now()
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		sessionRepo := s.sessionRepo.WithTx(tx)
		refreshRepo := s.refreshRepo.WithTx(tx)
		return s.revokeAllSessionsWithRepo(ctx, sessionRepo, refreshRepo, int64(userID), reason, now)
	})
	if err != nil {
		return errcode.ErrInternal
	}
	return nil
}

func (s *AuthService) findUser(ctx context.Context, userID uint) (*model.User, error) {
	var user model.User
	if err := s.userRepo.FindUserByID(ctx, userID, &user); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errcode.ErrNotFound
		}
		return nil, errcode.ErrInternal
	}
	return &user, nil
}

func (s *AuthService) revokeSession(ctx context.Context, session *model.Session, reason string) error {
	now := time.Now()
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		sessionRepo := s.sessionRepo.WithTx(tx)
		refreshRepo := s.refreshRepo.WithTx(tx)
		return s.revokeSessionWithRepo(ctx, sessionRepo, refreshRepo, session, reason, now)
	})
	if err != nil {
		return errcode.ErrInternal
	}
	return nil
}

func (s *AuthService) revokeSessionWithRepo(
	ctx context.Context,
	sessionRepo repository.SessionRepository,
	refreshRepo repository.RefreshTokenRepository,
	session *model.Session,
	reason string,
	revokedAt time.Time,
) error {
	session.Status = sessionStatusRevoked
	session.RevokedAt = &revokedAt
	session.RevokeReason = reason
	if err := sessionRepo.Update(ctx, session); err != nil {
		return err
	}
	return refreshRepo.RevokeActiveBySessionID(ctx, session.SessionID, reason, revokedAt)
}

func (s *AuthService) revokeAllSessionsWithRepo(
	ctx context.Context,
	sessionRepo repository.SessionRepository,
	refreshRepo repository.RefreshTokenRepository,
	userID int64,
	reason string,
	revokedAt time.Time,
) error {
	if err := sessionRepo.RevokeActiveByUserID(ctx, userID, reason, revokedAt); err != nil {
		return err
	}
	return refreshRepo.RevokeActiveByUserID(ctx, userID, reason, revokedAt)
}

func (s *AuthService) recordEventWithRepo(
	ctx context.Context,
	eventRepo repository.SecurityEventRepository,
	userID int64,
	sessionID, eventType, ip, deviceID, userAgent, detail string,
) error {
	return eventRepo.Create(ctx, &model.SecurityEvent{
		UserID:    userID,
		SessionID: sessionID,
		EventType: eventType,
		IP:        ip,
		DeviceID:  deviceID,
		UserAgent: userAgent,
		Detail:    detail,
		CreatedAt: time.Now(),
	})
}
