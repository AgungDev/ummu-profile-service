package usecases

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/AgungDev/ummu-profile-service/internal/config"
	"github.com/AgungDev/ummu-profile-service/internal/domain"
	"github.com/AgungDev/ummu-profile-service/internal/domain/dto"
	"github.com/AgungDev/ummu-profile-service/internal/repositories"
	"github.com/google/uuid"
)

type profileRepositoryFake struct {
	profile     *domain.UserProfile
	ensureCalls int
}

func (r *profileRepositoryFake) Ensure(_ context.Context, userID uuid.UUID) (*domain.UserProfile, error) {
	r.ensureCalls++
	if r.profile == nil {
		r.profile = &domain.UserProfile{UserID: userID, FullName: "Alya"}
	}
	copy := *r.profile
	return &copy, nil
}

func (r *profileRepositoryFake) Get(_ context.Context, userID uuid.UUID) (*domain.UserProfile, error) {
	if r.profile == nil || r.profile.UserID != userID {
		return nil, domain.ErrNotFound
	}
	copy := *r.profile
	return &copy, nil
}

func (r *profileRepositoryFake) Update(_ context.Context, profile *domain.UserProfile, addresses *[]domain.UserAddress) error {
	if r.profile == nil {
		return errors.New("profile not provisioned")
	}
	r.profile.FullName = profile.FullName
	r.profile.PhoneNumber = profile.PhoneNumber
	if addresses != nil {
		r.profile.Addresses = *addresses
	}
	return nil
}

func (r *profileRepositoryFake) SetAvatar(_ context.Context, userID uuid.UUID, key *string) error {
	if r.profile == nil || r.profile.UserID != userID {
		return domain.ErrNotFound
	}
	r.profile.AvatarKey = key
	return nil
}

var _ repositories.ProfileRepository = (*profileRepositoryFake)(nil)

func TestGetEnsuresProfileForUserAndReturnsExistingData(t *testing.T) {
	userID := uuid.New()
	repository := &profileRepositoryFake{}
	usecase := NewProfileUsecase(repository, nil, config.AvatarConfig{}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	first, err := usecase.Get(context.Background(), userID)
	if err != nil {
		t.Fatalf("first Get() error = %v", err)
	}
	second, err := usecase.Get(context.Background(), userID)
	if err != nil {
		t.Fatalf("second Get() error = %v", err)
	}
	if repository.ensureCalls != 2 || first.UserID != userID.String() || second.FullName != "Alya" {
		t.Fatalf("unexpected idempotent profile result: calls=%d first=%+v second=%+v", repository.ensureCalls, first, second)
	}
}

func TestUpdateRejectsMoreThanOnePrimaryAddress(t *testing.T) {
	userID := uuid.New()
	repository := &profileRepositoryFake{profile: &domain.UserProfile{UserID: userID}}
	usecase := NewProfileUsecase(repository, nil, config.AvatarConfig{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	addresses := []dto.AddressInput{
		{AddressLine: "Jl. A", City: "Ternate", Province: "Maluku Utara", IsPrimary: true},
		{AddressLine: "Jl. B", City: "Ternate", Province: "Maluku Utara", IsPrimary: true},
	}

	_, err := usecase.Update(context.Background(), userID, dto.UpdateProfileRequest{Addresses: &addresses})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("Update() error = %v, want ErrInvalidInput", err)
	}
	if len(repository.profile.Addresses) != 0 {
		t.Fatalf("invalid address update changed stored addresses: %+v", repository.profile.Addresses)
	}
}

func TestPublicProfileOmitsPrivateFields(t *testing.T) {
	userID := uuid.New()
	phone := "+628123456789"
	repository := &profileRepositoryFake{profile: &domain.UserProfile{
		UserID: userID, FullName: "Alya", PhoneNumber: &phone,
		Addresses: []domain.UserAddress{{AddressLine: "private address"}},
	}}
	usecase := NewProfileUsecase(repository, nil, config.AvatarConfig{}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	publicProfile, err := usecase.GetPublic(context.Background(), userID)
	if err != nil {
		t.Fatalf("GetPublic() error = %v", err)
	}
	if publicProfile.UserID != userID.String() || publicProfile.FullName != "Alya" {
		t.Fatalf("unexpected public profile: %+v", publicProfile)
	}
}
