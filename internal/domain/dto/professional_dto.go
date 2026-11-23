package dto

import "time"

type StudentRequest struct {
	NIM            string `json:"nim"`
	StudyProgramID string `json:"study_program_id"`
	CohortYear     int    `json:"cohort_year"`
	AdmissionType  string `json:"admission_type"`
	AcademicStatus string `json:"academic_status"`
}

type StudentResponse struct {
	ID             string    `json:"id"`
	UserID         string    `json:"user_id"`
	NIM            string    `json:"nim"`
	StudyProgramID string    `json:"study_program_id"`
	CohortYear     int       `json:"cohort_year"`
	AdmissionType  string    `json:"admission_type"`
	AcademicStatus string    `json:"academic_status"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type LecturerRequest struct {
	NIDN             *string `json:"nidn"`
	NIDK             *string `json:"nidk"`
	Expertise        string  `json:"expertise"`
	AcademicPosition string  `json:"academic_position"`
}

type LecturerResponse struct {
	ID               string    `json:"id"`
	UserID           string    `json:"user_id"`
	NIDN             *string   `json:"nidn,omitempty"`
	NIDK             *string   `json:"nidk,omitempty"`
	Expertise        string    `json:"expertise"`
	AcademicPosition string    `json:"academic_position"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type EmployeeRequest struct {
	EmployeeNumber       string `json:"employee_number"`
	EmploymentStatus     string `json:"employment_status"`
	Position             string `json:"position"`
	OrganizationalUnitID string `json:"organizational_unit_id"`
	EmploymentStartDate  string `json:"employment_start_date"`
}

type EmployeeResponse struct {
	ID                   string    `json:"id"`
	UserID               string    `json:"user_id"`
	EmployeeNumber       string    `json:"employee_number"`
	EmploymentStatus     string    `json:"employment_status"`
	Position             string    `json:"position"`
	OrganizationalUnitID string    `json:"organizational_unit_id"`
	EmploymentStartDate  string    `json:"employment_start_date"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}
