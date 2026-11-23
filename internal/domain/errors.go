package domain

import "errors"

var (
	ErrNotFound             = errors.New("profile not found")
	ErrProfessionalNotFound = errors.New("professional record not found")
	ErrInvalidInput         = errors.New("invalid profile input")
	ErrConflict             = errors.New("professional identifier already exists")
	ErrNoChanges            = errors.New("at least one profile field is required")
	ErrAvatarUnavailable    = errors.New("avatar object storage is not configured")
	ErrAvatarTooLarge       = errors.New("avatar exceeds the configured size limit")
	ErrInvalidAvatar        = errors.New("avatar must be a JPEG, PNG, or WebP image")
)
