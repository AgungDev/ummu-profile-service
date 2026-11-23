package usecases

import (
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"log/slog"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/AgungDev/ummu-profile-service/internal/config"
	"github.com/AgungDev/ummu-profile-service/internal/domain"
	"github.com/AgungDev/ummu-profile-service/internal/domain/dto"
	"github.com/AgungDev/ummu-profile-service/internal/repositories"
	"github.com/AgungDev/ummu-profile-service/internal/services"
	"github.com/google/uuid"
)

type ProfileUsecase interface {
	Get(ctx context.Context, userID uuid.UUID) (*dto.ProfileResponse, error)
	GetPublic(ctx context.Context, userID uuid.UUID) (*dto.PublicProfileResponse, error)
	Update(ctx context.Context, userID uuid.UUID, req dto.UpdateProfileRequest) (*dto.ProfileResponse, error)
	UploadAvatar(ctx context.Context, userID uuid.UUID, contentType string, data []byte) (*dto.ProfileResponse, error)
	DeleteAvatar(ctx context.Context, userID uuid.UUID) (*dto.ProfileResponse, error)
}

type profileUsecase struct {
	repository repositories.ProfileRepository
	avatars    services.AvatarStore
	avatarCfg  config.AvatarConfig
	logger     *slog.Logger
}

func NewProfileUsecase(
	repository repositories.ProfileRepository,
	avatars services.AvatarStore,
	avatarCfg config.AvatarConfig,
	logger *slog.Logger,
) ProfileUsecase {
	return &profileUsecase{repository: repository, avatars: avatars, avatarCfg: avatarCfg, logger: logger}
}

func (uc *profileUsecase) Get(ctx context.Context, userID uuid.UUID) (*dto.ProfileResponse, error) {
	profile, err := uc.repository.Ensure(ctx, userID)
	if err != nil {
		return nil, err
	}
	return uc.toResponse(profile), nil
}

func (uc *profileUsecase) GetPublic(ctx context.Context, userID uuid.UUID) (*dto.PublicProfileResponse, error) {
	profile, err := uc.repository.Get(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &dto.PublicProfileResponse{
		UserID:    profile.UserID.String(),
		FullName:  profile.FullName,
		AvatarURL: uc.avatarURL(profile.AvatarKey),
		UpdatedAt: profile.UpdatedAt,
	}, nil
}

func (uc *profileUsecase) Update(ctx context.Context, userID uuid.UUID, req dto.UpdateProfileRequest) (*dto.ProfileResponse, error) {
	if req.FullName == nil && req.PhoneNumber == nil && req.Addresses == nil {
		return nil, domain.ErrNoChanges
	}
	profile, err := uc.repository.Ensure(ctx, userID)
	if err != nil {
		return nil, err
	}
	if req.FullName != nil {
		name := strings.TrimSpace(*req.FullName)
		if name == "" || utf8.RuneCountInString(name) > 128 {
			return nil, fmt.Errorf("%w: full_name must contain 1-128 characters", domain.ErrInvalidInput)
		}
		profile.FullName = name
	}
	if req.PhoneNumber != nil {
		phone := strings.TrimSpace(*req.PhoneNumber)
		if utf8.RuneCountInString(phone) > 32 {
			return nil, fmt.Errorf("%w: phone_number must be at most 32 characters", domain.ErrInvalidInput)
		}
		if phone == "" {
			profile.PhoneNumber = nil
		} else {
			profile.PhoneNumber = &phone
		}
	}

	var addresses *[]domain.UserAddress
	if req.Addresses != nil {
		converted, err := convertAddresses(userID, *req.Addresses)
		if err != nil {
			return nil, err
		}
		addresses = &converted
	}
	profile.UpdatedAt = time.Now().UTC()
	if err := uc.repository.Update(ctx, profile, addresses); err != nil {
		return nil, err
	}
	return uc.Get(ctx, userID)
}

func (uc *profileUsecase) UploadAvatar(ctx context.Context, userID uuid.UUID, contentType string, data []byte) (*dto.ProfileResponse, error) {
	if uc.avatars == nil {
		return nil, domain.ErrAvatarUnavailable
	}
	if int64(len(data)) > uc.avatarCfg.MaxBytes {
		return nil, domain.ErrAvatarTooLarge
	}
	detectedType := http.DetectContentType(data)
	extension, ok := map[string]string{
		"image/jpeg": ".jpg",
		"image/png":  ".png",
	}[detectedType]
	if !ok || (contentType != "" && contentType != detectedType) {
		return nil, domain.ErrInvalidAvatar
	}
	imageConfig, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || imageConfig.Width < 1 || imageConfig.Height < 1 ||
		imageConfig.Width > 8192 || imageConfig.Height > 8192 ||
		int64(imageConfig.Width)*int64(imageConfig.Height) > 40_000_000 {
		return nil, domain.ErrInvalidAvatar
	}

	profile, err := uc.repository.Ensure(ctx, userID)
	if err != nil {
		return nil, err
	}
	newKey := userID.String() + "/" + uuid.NewString() + extension
	if err := uc.avatars.Put(ctx, newKey, detectedType, data); err != nil {
		return nil, err
	}
	if err := uc.repository.SetAvatar(ctx, userID, &newKey); err != nil {
		if cleanupErr := uc.avatars.Delete(ctx, newKey); cleanupErr != nil {
			uc.logger.Error("failed to clean up uploaded avatar after database error", "key", newKey, "error", cleanupErr)
		}
		return nil, err
	}
	if profile.AvatarKey != nil {
		if err := uc.avatars.Delete(ctx, *profile.AvatarKey); err != nil {
			uc.logger.Error("failed to delete replaced avatar object", "key", *profile.AvatarKey, "error", err)
		}
	}
	return uc.Get(ctx, userID)
}

func (uc *profileUsecase) DeleteAvatar(ctx context.Context, userID uuid.UUID) (*dto.ProfileResponse, error) {
	profile, err := uc.repository.Ensure(ctx, userID)
	if err != nil {
		return nil, err
	}
	if profile.AvatarKey == nil {
		return uc.toResponse(profile), nil
	}
	if uc.avatars == nil {
		return nil, domain.ErrAvatarUnavailable
	}
	oldKey := *profile.AvatarKey
	if err := uc.repository.SetAvatar(ctx, userID, nil); err != nil {
		return nil, err
	}
	if err := uc.avatars.Delete(ctx, oldKey); err != nil {
		uc.logger.Error("failed to delete avatar object", "key", oldKey, "error", err)
	}
	return uc.Get(ctx, userID)
}

func (uc *profileUsecase) toResponse(profile *domain.UserProfile) *dto.ProfileResponse {
	addresses := make([]dto.AddressResponse, 0, len(profile.Addresses))
	for _, address := range profile.Addresses {
		addresses = append(addresses, dto.AddressResponse{
			ID:          address.ID.String(),
			Label:       address.Label,
			AddressLine: address.AddressLine,
			City:        address.City,
			Province:    address.Province,
			PostalCode:  address.PostalCode,
			Country:     address.Country,
			IsPrimary:   address.IsPrimary,
		})
	}
	return &dto.ProfileResponse{
		UserID:      profile.UserID.String(),
		FullName:    profile.FullName,
		PhoneNumber: profile.PhoneNumber,
		AvatarURL:   uc.avatarURL(profile.AvatarKey),
		Addresses:   addresses,
		CreatedAt:   profile.CreatedAt,
		UpdatedAt:   profile.UpdatedAt,
	}
}

func (uc *profileUsecase) avatarURL(key *string) *string {
	if key == nil || uc.avatars == nil {
		return nil
	}
	value := uc.avatars.URL(*key)
	return &value
}

func convertAddresses(userID uuid.UUID, inputs []dto.AddressInput) ([]domain.UserAddress, error) {
	addresses := make([]domain.UserAddress, 0, len(inputs))
	primaryCount := 0
	for _, input := range inputs {
		label := strings.TrimSpace(input.Label)
		if label == "" {
			label = "home"
		}
		line := strings.TrimSpace(input.AddressLine)
		city := strings.TrimSpace(input.City)
		province := strings.TrimSpace(input.Province)
		country := strings.ToUpper(strings.TrimSpace(input.Country))
		if country == "" {
			country = "ID"
		}
		if line == "" || city == "" || province == "" ||
			utf8.RuneCountInString(label) > 32 ||
			utf8.RuneCountInString(city) > 128 ||
			utf8.RuneCountInString(province) > 128 ||
			len(country) != 2 {
			return nil, fmt.Errorf("%w: address requires address_line, city, province and a two-letter country code", domain.ErrInvalidInput)
		}
		if input.IsPrimary {
			primaryCount++
		}
		addresses = append(addresses, domain.UserAddress{
			UserID: userID, Label: label, AddressLine: line, City: city,
			Province: province, PostalCode: input.PostalCode, Country: country,
			IsPrimary: input.IsPrimary,
		})
	}
	if primaryCount > 1 {
		return nil, fmt.Errorf("%w: only one address can be primary", domain.ErrInvalidInput)
	}
	return addresses, nil
}
