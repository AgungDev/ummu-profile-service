package repositories

import (
	"context"
	"errors"
	"fmt"

	"github.com/AgungDev/ummu-profile-service/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ProfileRepository interface {
	Ensure(ctx context.Context, userID uuid.UUID) (*domain.UserProfile, error)
	Get(ctx context.Context, userID uuid.UUID) (*domain.UserProfile, error)
	Update(ctx context.Context, profile *domain.UserProfile, addresses *[]domain.UserAddress) error
	SetAvatar(ctx context.Context, userID uuid.UUID, key *string) error
}

type profileRepository struct {
	db *gorm.DB
}

func NewProfileRepository(db *gorm.DB) ProfileRepository {
	return &profileRepository{db: db}
}

func (r *profileRepository) Ensure(ctx context.Context, userID uuid.UUID) (*domain.UserProfile, error) {
	profile := domain.UserProfile{UserID: userID}
	if err := r.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&profile).Error; err != nil {
		return nil, fmt.Errorf("ensure profile: %w", err)
	}
	return r.Get(ctx, userID)
}

func (r *profileRepository) Get(ctx context.Context, userID uuid.UUID) (*domain.UserProfile, error) {
	var profile domain.UserProfile
	err := r.db.WithContext(ctx).Preload("Addresses", func(db *gorm.DB) *gorm.DB {
		return db.Order("is_primary DESC, created_at ASC")
	}).First(&profile, "user_id = ?", userID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get profile: %w", err)
	}
	return &profile, nil
}

func (r *profileRepository) Update(ctx context.Context, profile *domain.UserProfile, addresses *[]domain.UserAddress) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&domain.UserProfile{}).Where("user_id = ?", profile.UserID).Updates(map[string]interface{}{
			"full_name":    profile.FullName,
			"phone_number": profile.PhoneNumber,
			"updated_at":   profile.UpdatedAt,
		}).Error; err != nil {
			return fmt.Errorf("update profile: %w", err)
		}
		if addresses != nil {
			if err := tx.Where("user_id = ?", profile.UserID).Delete(&domain.UserAddress{}).Error; err != nil {
				return fmt.Errorf("replace profile addresses: %w", err)
			}
			for i := range *addresses {
				if err := tx.Create(&(*addresses)[i]).Error; err != nil {
					return fmt.Errorf("save profile address: %w", err)
				}
			}
		}
		return nil
	})
}

func (r *profileRepository) SetAvatar(ctx context.Context, userID uuid.UUID, key *string) error {
	result := r.db.WithContext(ctx).Model(&domain.UserProfile{}).Where("user_id = ?", userID).Updates(map[string]interface{}{
		"avatar_key": key,
		"updated_at": gorm.Expr("NOW()"),
	})
	if result.Error != nil {
		return fmt.Errorf("update profile avatar: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}
