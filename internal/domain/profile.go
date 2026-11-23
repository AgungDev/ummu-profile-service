package domain

import (
	"time"

	"github.com/google/uuid"
)

type UserProfile struct {
	UserID      uuid.UUID `gorm:"type:uuid;primaryKey"`
	FullName    string    `gorm:"type:varchar(128);not null;default:''"`
	PhoneNumber *string   `gorm:"type:varchar(32)"`
	AvatarKey   *string   `gorm:"type:text"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Addresses   []UserAddress `gorm:"foreignKey:UserID;references:UserID"`
}

func (UserProfile) TableName() string { return "user_profiles" }

type UserAddress struct {
	ID          uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID      uuid.UUID `gorm:"type:uuid;not null;index"`
	Label       string    `gorm:"type:varchar(32);not null;default:'home'"`
	AddressLine string    `gorm:"type:text;not null"`
	City        string    `gorm:"type:varchar(128);not null"`
	Province    string    `gorm:"type:varchar(128);not null"`
	PostalCode  *string   `gorm:"type:varchar(16)"`
	Country     string    `gorm:"type:char(2);not null;default:'ID'"`
	IsPrimary   bool      `gorm:"not null;default:false"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (UserAddress) TableName() string { return "user_addresses" }
