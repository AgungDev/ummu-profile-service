package domain

import (
	"time"

	"github.com/google/uuid"
)

type Student struct {
	ID             uuid.UUID   `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID         uuid.UUID   `gorm:"type:uuid;not null;uniqueIndex"`
	Profile        UserProfile `gorm:"foreignKey:UserID;references:UserID" json:"-"`
	NIM            string      `gorm:"type:varchar(20);not null;uniqueIndex"`
	StudyProgramID uuid.UUID   `gorm:"type:uuid;not null"`
	CohortYear     int         `gorm:"not null"`
	AdmissionType  string      `gorm:"type:varchar(32);not null"`
	AcademicStatus string      `gorm:"type:varchar(32);not null"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (Student) TableName() string { return "students" }

type Lecturer struct {
	ID               uuid.UUID   `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID           uuid.UUID   `gorm:"type:uuid;not null;uniqueIndex"`
	Profile          UserProfile `gorm:"foreignKey:UserID;references:UserID" json:"-"`
	NIDN             *string     `gorm:"type:varchar(32);uniqueIndex"`
	NIDK             *string     `gorm:"type:varchar(32);uniqueIndex"`
	Expertise        string      `gorm:"type:varchar(255);not null"`
	AcademicPosition string      `gorm:"type:varchar(128);not null"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (Lecturer) TableName() string { return "lecturers" }

type Employee struct {
	ID                   uuid.UUID   `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID               uuid.UUID   `gorm:"type:uuid;not null;uniqueIndex"`
	Profile              UserProfile `gorm:"foreignKey:UserID;references:UserID" json:"-"`
	EmployeeNumber       string      `gorm:"type:varchar(64);not null;uniqueIndex"`
	EmploymentStatus     string      `gorm:"type:varchar(32);not null"`
	Position             string      `gorm:"type:varchar(128);not null"`
	OrganizationalUnitID uuid.UUID   `gorm:"type:uuid;not null"`
	EmploymentStartDate  time.Time   `gorm:"type:date;not null"`
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

func (Employee) TableName() string { return "employees" }
