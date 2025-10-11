package repository

import (
	"context"
	"time"

	"github.com/dbehnke/allstar-nexus/backend/models"
	"gorm.io/gorm"
)

type UserRepo struct{ DB *gorm.DB }

func NewUserRepo(db *gorm.DB) *UserRepo { return &UserRepo{DB: db} }

func (r *UserRepo) Create(ctx context.Context, email, passwordHash, role string) (*models.User, error) {
	user := &models.User{
		Email:        email,
		PasswordHash: passwordHash,
		Role:         role,
		CreatedAt:    time.Now(),
	}
	if err := r.DB.WithContext(ctx).Create(user).Error; err != nil {
		return nil, err
	}
	return user, nil
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	err := r.DB.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// Count returns total number of users.
func (r *UserRepo) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.DB.WithContext(ctx).Model(&models.User{}).Count(&count).Error
	return count, err
}

// RoleCounts returns map[role]count.
func (r *UserRepo) RoleCounts(ctx context.Context) (map[string]int64, error) {
	type Result struct {
		Role  string
		Count int64
	}
	var results []Result
	err := r.DB.WithContext(ctx).Model(&models.User{}).
		Select("role, count(*) as count").
		Group("role").
		Scan(&results).Error
	if err != nil {
		return nil, err
	}
	res := make(map[string]int64)
	for _, r := range results {
		res[r.Role] = r.Count
	}
	return res, nil
}

// NewUsersSince returns number of users created at or after 'since'.
func (r *UserRepo) NewUsersSince(ctx context.Context, since time.Time) (int64, error) {
	var count int64
	err := r.DB.WithContext(ctx).Model(&models.User{}).
		Where("created_at >= ?", since).
		Count(&count).Error
	return count, err
}

// SafeUser minimal representation.
type SafeUser struct {
	ID    int64
	Email string
	Role  string
}
