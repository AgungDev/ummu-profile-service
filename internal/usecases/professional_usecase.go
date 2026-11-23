package usecases

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/AgungDev/ummu-profile-service/internal/domain"
	"github.com/AgungDev/ummu-profile-service/internal/domain/dto"
	"github.com/AgungDev/ummu-profile-service/internal/repositories"
	"github.com/google/uuid"
)

type ProfessionalUsecase interface {
	GetStudent(ctx context.Context, userID uuid.UUID) (*dto.StudentResponse, error)
	SaveMyStudent(ctx context.Context, userID uuid.UUID, req dto.StudentRequest) (*dto.StudentResponse, error)
	SaveStudentForUser(ctx context.Context, userID uuid.UUID, req dto.StudentRequest) (*dto.StudentResponse, error)
	DeleteStudent(ctx context.Context, userID uuid.UUID) error
	GetLecturer(ctx context.Context, userID uuid.UUID) (*dto.LecturerResponse, error)
	SaveMyLecturer(ctx context.Context, userID uuid.UUID, req dto.LecturerRequest) (*dto.LecturerResponse, error)
	SaveLecturerForUser(ctx context.Context, userID uuid.UUID, req dto.LecturerRequest) (*dto.LecturerResponse, error)
	DeleteLecturer(ctx context.Context, userID uuid.UUID) error
	GetEmployee(ctx context.Context, userID uuid.UUID) (*dto.EmployeeResponse, error)
	SaveMyEmployee(ctx context.Context, userID uuid.UUID, req dto.EmployeeRequest) (*dto.EmployeeResponse, error)
	SaveEmployeeForUser(ctx context.Context, userID uuid.UUID, req dto.EmployeeRequest) (*dto.EmployeeResponse, error)
	DeleteEmployee(ctx context.Context, userID uuid.UUID) error
}

type professionalUsecase struct {
	profiles      repositories.ProfileRepository
	professionals repositories.ProfessionalRepository
}

func NewProfessionalUsecase(
	profiles repositories.ProfileRepository,
	professionals repositories.ProfessionalRepository,
) ProfessionalUsecase {
	return &professionalUsecase{profiles: profiles, professionals: professionals}
}

func (uc *professionalUsecase) GetStudent(ctx context.Context, userID uuid.UUID) (*dto.StudentResponse, error) {
	student, err := uc.professionals.GetStudent(ctx, userID)
	if err != nil {
		return nil, err
	}
	return studentResponse(student), nil
}

func (uc *professionalUsecase) SaveMyStudent(ctx context.Context, userID uuid.UUID, req dto.StudentRequest) (*dto.StudentResponse, error) {
	return uc.saveStudent(ctx, userID, req, true)
}

func (uc *professionalUsecase) SaveStudentForUser(ctx context.Context, userID uuid.UUID, req dto.StudentRequest) (*dto.StudentResponse, error) {
	return uc.saveStudent(ctx, userID, req, false)
}

func (uc *professionalUsecase) saveStudent(ctx context.Context, userID uuid.UUID, req dto.StudentRequest, ensureProfile bool) (*dto.StudentResponse, error) {
	nim := strings.TrimSpace(req.NIM)
	studyProgramID, err := uuid.Parse(strings.TrimSpace(req.StudyProgramID))
	if err != nil || studyProgramID == uuid.Nil || !validNIM(nim) ||
		req.CohortYear < 1900 || req.CohortYear > 9999 {
		return nil, invalidProfessionalInput("student requires a valid nim, study_program_id, and cohort_year")
	}
	admissionType := strings.TrimSpace(req.AdmissionType)
	academicStatus := strings.TrimSpace(req.AcademicStatus)
	if !validText(admissionType, 32) || !validText(academicStatus, 32) {
		return nil, invalidProfessionalInput("admission_type and academic_status must contain 1-32 characters")
	}
	if err := uc.checkProfile(ctx, userID, ensureProfile); err != nil {
		return nil, err
	}
	student := &domain.Student{
		UserID: userID, NIM: nim, StudyProgramID: studyProgramID,
		CohortYear: req.CohortYear, AdmissionType: admissionType, AcademicStatus: academicStatus,
		UpdatedAt: time.Now().UTC(),
	}
	if err := uc.professionals.SaveStudent(ctx, student); err != nil {
		return nil, err
	}
	return uc.GetStudent(ctx, userID)
}

func (uc *professionalUsecase) DeleteStudent(ctx context.Context, userID uuid.UUID) error {
	return uc.professionals.DeleteStudent(ctx, userID)
}

func (uc *professionalUsecase) GetLecturer(ctx context.Context, userID uuid.UUID) (*dto.LecturerResponse, error) {
	lecturer, err := uc.professionals.GetLecturer(ctx, userID)
	if err != nil {
		return nil, err
	}
	return lecturerResponse(lecturer), nil
}

func (uc *professionalUsecase) SaveMyLecturer(ctx context.Context, userID uuid.UUID, req dto.LecturerRequest) (*dto.LecturerResponse, error) {
	return uc.saveLecturer(ctx, userID, req, true)
}

func (uc *professionalUsecase) SaveLecturerForUser(ctx context.Context, userID uuid.UUID, req dto.LecturerRequest) (*dto.LecturerResponse, error) {
	return uc.saveLecturer(ctx, userID, req, false)
}

func (uc *professionalUsecase) saveLecturer(ctx context.Context, userID uuid.UUID, req dto.LecturerRequest, ensureProfile bool) (*dto.LecturerResponse, error) {
	nidn, nidk := cleanOptionalText(req.NIDN), cleanOptionalText(req.NIDK)
	if (nidn == nil && nidk == nil) || !validOptionalText(nidn, 32) || !validOptionalText(nidk, 32) {
		return nil, invalidProfessionalInput("lecturer requires nidn or nidk; each identifier must be at most 32 characters")
	}
	expertise := strings.TrimSpace(req.Expertise)
	position := strings.TrimSpace(req.AcademicPosition)
	if !validText(expertise, 255) || !validText(position, 128) {
		return nil, invalidProfessionalInput("expertise and academic_position are required and limited to 255 and 128 characters")
	}
	if err := uc.checkProfile(ctx, userID, ensureProfile); err != nil {
		return nil, err
	}
	lecturer := &domain.Lecturer{
		UserID: userID, NIDN: nidn, NIDK: nidk,
		Expertise: expertise, AcademicPosition: position, UpdatedAt: time.Now().UTC(),
	}
	if err := uc.professionals.SaveLecturer(ctx, lecturer); err != nil {
		return nil, err
	}
	return uc.GetLecturer(ctx, userID)
}

func (uc *professionalUsecase) DeleteLecturer(ctx context.Context, userID uuid.UUID) error {
	return uc.professionals.DeleteLecturer(ctx, userID)
}

func (uc *professionalUsecase) GetEmployee(ctx context.Context, userID uuid.UUID) (*dto.EmployeeResponse, error) {
	employee, err := uc.professionals.GetEmployee(ctx, userID)
	if err != nil {
		return nil, err
	}
	return employeeResponse(employee), nil
}

func (uc *professionalUsecase) SaveMyEmployee(ctx context.Context, userID uuid.UUID, req dto.EmployeeRequest) (*dto.EmployeeResponse, error) {
	return uc.saveEmployee(ctx, userID, req, true)
}

func (uc *professionalUsecase) SaveEmployeeForUser(ctx context.Context, userID uuid.UUID, req dto.EmployeeRequest) (*dto.EmployeeResponse, error) {
	return uc.saveEmployee(ctx, userID, req, false)
}

func (uc *professionalUsecase) saveEmployee(ctx context.Context, userID uuid.UUID, req dto.EmployeeRequest, ensureProfile bool) (*dto.EmployeeResponse, error) {
	number := strings.TrimSpace(req.EmployeeNumber)
	status := strings.TrimSpace(req.EmploymentStatus)
	position := strings.TrimSpace(req.Position)
	unitID, unitErr := uuid.Parse(strings.TrimSpace(req.OrganizationalUnitID))
	startDate, dateErr := time.Parse("2006-01-02", strings.TrimSpace(req.EmploymentStartDate))
	if !validText(number, 64) || !validText(status, 32) || !validText(position, 128) ||
		unitErr != nil || unitID == uuid.Nil || dateErr != nil {
		return nil, invalidProfessionalInput("employee requires employee_number, employment_status, position, organizational_unit_id, and employment_start_date (YYYY-MM-DD)")
	}
	if err := uc.checkProfile(ctx, userID, ensureProfile); err != nil {
		return nil, err
	}
	employee := &domain.Employee{
		UserID: userID, EmployeeNumber: number, EmploymentStatus: status, Position: position,
		OrganizationalUnitID: unitID, EmploymentStartDate: startDate,
		UpdatedAt: time.Now().UTC(),
	}
	if err := uc.professionals.SaveEmployee(ctx, employee); err != nil {
		return nil, err
	}
	return uc.GetEmployee(ctx, userID)
}

func (uc *professionalUsecase) DeleteEmployee(ctx context.Context, userID uuid.UUID) error {
	return uc.professionals.DeleteEmployee(ctx, userID)
}

func (uc *professionalUsecase) checkProfile(ctx context.Context, userID uuid.UUID, ensure bool) error {
	if ensure {
		_, err := uc.profiles.Ensure(ctx, userID)
		return err
	}
	_, err := uc.profiles.Get(ctx, userID)
	return err
}

func studentResponse(student *domain.Student) *dto.StudentResponse {
	return &dto.StudentResponse{
		ID: student.ID.String(), UserID: student.UserID.String(), NIM: student.NIM,
		StudyProgramID: student.StudyProgramID.String(), CohortYear: student.CohortYear,
		AdmissionType: student.AdmissionType, AcademicStatus: student.AcademicStatus,
		CreatedAt: student.CreatedAt, UpdatedAt: student.UpdatedAt,
	}
}

func lecturerResponse(lecturer *domain.Lecturer) *dto.LecturerResponse {
	return &dto.LecturerResponse{
		ID: lecturer.ID.String(), UserID: lecturer.UserID.String(),
		NIDN: lecturer.NIDN, NIDK: lecturer.NIDK, Expertise: lecturer.Expertise,
		AcademicPosition: lecturer.AcademicPosition,
		CreatedAt:        lecturer.CreatedAt, UpdatedAt: lecturer.UpdatedAt,
	}
}

func employeeResponse(employee *domain.Employee) *dto.EmployeeResponse {
	return &dto.EmployeeResponse{
		ID: employee.ID.String(), UserID: employee.UserID.String(),
		EmployeeNumber: employee.EmployeeNumber, EmploymentStatus: employee.EmploymentStatus,
		Position: employee.Position, OrganizationalUnitID: employee.OrganizationalUnitID.String(),
		EmploymentStartDate: employee.EmploymentStartDate.Format("2006-01-02"),
		CreatedAt:           employee.CreatedAt, UpdatedAt: employee.UpdatedAt,
	}
}

func validNIM(value string) bool {
	if value == "" || len(value) > 20 {
		return false
	}
	for _, char := range value {
		if !(char >= 'a' && char <= 'z') && !(char >= 'A' && char <= 'Z') &&
			!(char >= '0' && char <= '9') && char != '_' && char != '-' {
			return false
		}
	}
	return true
}

func validText(value string, max int) bool {
	return value != "" && utf8.RuneCountInString(value) <= max
}

func cleanOptionalText(value *string) *string {
	if value == nil {
		return nil
	}
	cleaned := strings.TrimSpace(*value)
	if cleaned == "" {
		return nil
	}
	return &cleaned
}

func validOptionalText(value *string, max int) bool {
	return value == nil || validText(*value, max)
}

func invalidProfessionalInput(message string) error {
	return fmt.Errorf("%w: %s", domain.ErrInvalidInput, message)
}
