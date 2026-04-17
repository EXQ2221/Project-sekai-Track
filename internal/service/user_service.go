package service

import (
	"context"
	"errors"
	"net/mail"
	"path/filepath"
	"strings"

	"Project_sekai_search/internal/dto"
	"Project_sekai_search/internal/model"
	"Project_sekai_search/internal/pkg/characters"
	"Project_sekai_search/internal/pkg/errcode"
	"Project_sekai_search/internal/repository"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserService struct {
	userRepo repository.UserRepository
	authSvc  *AuthService
}

const characterBaseDir = "static/characters"

func NewUserService(userRepo repository.UserRepository) *UserService {
	return &UserService{userRepo: userRepo}
}

func (s *UserService) SetAuthService(authSvc *AuthService) {
	s.authSvc = authSvc
}

func (s *UserService) RegisterService(ctx context.Context, req dto.RegisterRequest) (*model.User, error) {
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if _, err := mail.ParseAddress(req.Email); err != nil {
		return nil, errcode.ErrBadRequest
	}

	exists, err := s.userRepo.ExistsByUsername(ctx, req.Username)
	if err != nil {
		return nil, errcode.ErrInternal
	}
	if exists {
		return nil, errcode.ErrConflict
	}
	emailExists, err := s.userRepo.ExistsByEmail(ctx, req.Email)
	if err != nil {
		return nil, errcode.ErrInternal
	}
	if emailExists {
		return nil, errcode.ErrConflict
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errcode.ErrInternal
	}

	user := &model.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
	}

	if err := s.userRepo.CreateUser(ctx, user); err != nil {
		return nil, errcode.ErrInternal
	}
	return user, nil
}

func (s *UserService) ChangePassService(ctx context.Context, req dto.ChangePassRequest, id uint) error {
	var user model.User
	if err := s.userRepo.FindUserByID(ctx, id, &user); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errcode.ErrNotFound
		}
		return errcode.ErrInternal
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.OldPass)); err != nil {
		return errcode.ErrForbidden
	}

	if req.NewPass == "" || req.OldPass == req.NewPass {
		return errcode.ErrForbidden
	}

	newHashBytes, err := bcrypt.GenerateFromPassword([]byte(req.NewPass), bcrypt.DefaultCost)
	if err != nil {
		return errcode.ErrInternal
	}
	if err := s.userRepo.ChangePassword(ctx, id, string(newHashBytes)); err != nil {
		return errcode.ErrInternal
	}

	if s.authSvc != nil {
		if err := s.authSvc.RevokeAllUserSessions(ctx, id, "password_changed"); err != nil {
			return err
		}
	}
	return nil
}

func (s *UserService) GetMyProfile(ctx context.Context, id uint) (*dto.MyProfileResponse, error) {
	var user model.User
	if err := s.userRepo.FindUserByID(ctx, id, &user); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errcode.ErrNotFound
		}
		return nil, errcode.ErrInternal
	}
	characterName, characterImageURL := resolveCharacterDetail(user.Character)

	return &dto.MyProfileResponse{
		ID:                user.ID,
		Username:          user.Username,
		AvatarURL:         user.AvatarURL,
		Profile:           user.Profile,
		Character:         user.Character,
		CharacterName:     characterName,
		CharacterImageURL: characterImageURL,
		B30Avg:            user.B30Avg,
	}, nil
}

func (s *UserService) UpdateAvatarURL(ctx context.Context, id uint, avatarURL string) error {
	if err := s.userRepo.UpdateAvatarURL(ctx, id, avatarURL); err != nil {
		return errcode.ErrInternal
	}
	return nil
}

func (s *UserService) UpdateProfile(ctx context.Context, id uint, profile string) error {
	profile = strings.TrimSpace(profile)
	if len(profile) > 255 {
		return errcode.ErrBadRequest
	}
	if err := s.userRepo.UpdateProfile(ctx, id, profile); err != nil {
		return errcode.ErrInternal
	}
	return nil
}

func (s *UserService) UpdateCharacter(ctx context.Context, id uint, character string) error {
	character = strings.TrimSpace(character)
	if len(character) > 255 {
		return errcode.ErrBadRequest
	}

	if character != "" {
		_, ok, err := characters.FindByKey(filepath.Clean(characterBaseDir), character)
		if err != nil {
			return errcode.ErrInternal
		}
		if !ok {
			return errcode.ErrBadRequest
		}
	}

	if err := s.userRepo.UpdateCharacter(ctx, id, character); err != nil {
		return errcode.ErrInternal
	}
	return nil
}

func (s *UserService) ListCharacters() ([]dto.CharacterOption, error) {
	list, err := characters.List(filepath.Clean(characterBaseDir))
	if err != nil {
		return nil, errcode.ErrInternal
	}

	options := make([]dto.CharacterOption, 0, len(list))
	for _, item := range list {
		options = append(options, dto.CharacterOption{
			Key:      item.Key,
			Name:     item.Name,
			ImageURL: item.ImageURL,
		})
	}
	return options, nil
}

func resolveCharacterDetail(characterKey string) (string, string) {
	characterKey = strings.TrimSpace(characterKey)
	if characterKey == "" {
		return "", ""
	}

	item, ok, err := characters.FindByKey(filepath.Clean(characterBaseDir), characterKey)
	if err != nil || !ok {
		return characterKey, ""
	}
	return item.Name, item.ImageURL
}
