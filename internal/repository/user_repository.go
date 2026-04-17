package repository

import (
	"context"

	"Project_sekai_search/internal/model"

	"gorm.io/gorm"
)

type UserRepository interface {
	ExistsByUsername(ctx context.Context, username string) (bool, error)
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	FindUserByID(ctx context.Context, id uint, user *model.User) error
	FindUserByUsername(ctx context.Context, username string) (*model.User, error)
	ChangePassword(ctx context.Context, id uint, newHash string) error
	UpdateAvatarURL(ctx context.Context, id uint, avatarURL string) error
	UpdateProfile(ctx context.Context, id uint, profile string) error
	UpdateCharacter(ctx context.Context, id uint, character string) error
	CreateUser(ctx context.Context, user *model.User) error
}

type userRepo struct {
	db *gorm.DB
}

func NewUserRepo(db *gorm.DB) UserRepository {
	return &userRepo{db: db}
}

func (r *userRepo) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("username = ?", username).
		Count(&count).Error
	return count > 0, err
}

func (r *userRepo) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("email = ?", email).
		Count(&count).Error
	return count > 0, err
}

func (r *userRepo) FindUserByID(ctx context.Context, id uint, user *model.User) error {
	return r.db.WithContext(ctx).Where("id = ?", id).First(user).Error
}

func (r *userRepo) FindUserByUsername(ctx context.Context, username string) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepo) ChangePassword(ctx context.Context, id uint, newHash string) error {
	return r.db.WithContext(ctx).Model(&model.User{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"password_hash": newHash,
			"token_version": gorm.Expr("token_version + 1"),
		}).Error
}

func (r *userRepo) UpdateAvatarURL(ctx context.Context, id uint, avatarURL string) error {
	return r.db.WithContext(ctx).Model(&model.User{}).
		Where("id = ?", id).
		Update("avatar_url", avatarURL).Error
}

func (r *userRepo) UpdateProfile(ctx context.Context, id uint, profile string) error {
	return r.db.WithContext(ctx).Model(&model.User{}).
		Where("id = ?", id).
		Update("profile", profile).Error
}

func (r *userRepo) UpdateCharacter(ctx context.Context, id uint, character string) error {
	return r.db.WithContext(ctx).Model(&model.User{}).
		Where("id = ?", id).
		Update("character", character).Error
}

func (r *userRepo) CreateUser(ctx context.Context, user *model.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}
