package dto

import "time"

type AddressInput struct {
	Label       string  `json:"label"`
	AddressLine string  `json:"address_line" binding:"required"`
	City        string  `json:"city" binding:"required"`
	Province    string  `json:"province" binding:"required"`
	PostalCode  *string `json:"postal_code"`
	Country     string  `json:"country"`
	IsPrimary   bool    `json:"is_primary"`
}

type UpdateProfileRequest struct {
	FullName    *string         `json:"full_name"`
	PhoneNumber *string         `json:"phone_number"`
	Addresses   *[]AddressInput `json:"addresses"`
}

type AddressResponse struct {
	ID          string  `json:"id"`
	Label       string  `json:"label"`
	AddressLine string  `json:"address_line"`
	City        string  `json:"city"`
	Province    string  `json:"province"`
	PostalCode  *string `json:"postal_code,omitempty"`
	Country     string  `json:"country"`
	IsPrimary   bool    `json:"is_primary"`
}

type ProfileResponse struct {
	UserID      string            `json:"user_id"`
	FullName    string            `json:"full_name"`
	PhoneNumber *string           `json:"phone_number,omitempty"`
	AvatarURL   *string           `json:"avatar_url,omitempty"`
	Addresses   []AddressResponse `json:"addresses"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

type PublicProfileResponse struct {
	UserID    string    `json:"user_id"`
	FullName  string    `json:"full_name"`
	AvatarURL *string   `json:"avatar_url,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
}
