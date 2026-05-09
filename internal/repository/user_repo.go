package repository

import (
	"context"

	"github.com/edaptix/server/internal/model"
	"gorm.io/gorm"
)

type UserRepo struct {
	db *gorm.DB
}

func NewUserRepo(db *gorm.DB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) Create(ctx context.Context, user *model.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *UserRepo) FindByPhone(ctx context.Context, phone string) (*model.User, error) {
	var user model.User
	if err := r.db.WithContext(ctx).Where("phone = ?", phone).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepo) FindByID(ctx context.Context, id int64) (*model.User, error) {
	var user model.User
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepo) UpdateInitialized(ctx context.Context, id int64, initialized bool) error {
	return r.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", id).Update("initialized", initialized).Error
}

func (r *UserRepo) CreateStudentProfile(ctx context.Context, profile *model.StudentProfile) error {
	return r.db.WithContext(ctx).Create(profile).Error
}

func (r *UserRepo) GetStudentProfile(ctx context.Context, userID int64) (*model.StudentProfile, error) {
	var profile model.StudentProfile
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&profile).Error; err != nil {
		return nil, err
	}
	return &profile, nil
}
