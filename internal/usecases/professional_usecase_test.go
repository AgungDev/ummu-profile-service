package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/AgungDev/ummu-profile-service/internal/domain"
	"github.com/AgungDev/ummu-profile-service/internal/domain/dto"
	"github.com/google/uuid"
)

type professionalRepositoryFake struct {
	student  *domain.Student
	lecturer *domain.Lecturer
	employee *domain.Employee
	saves    int
}

func (r *professionalRepositoryFake) GetStudent(_ context.Context, userID uuid.UUID) (*domain.Student, error) {
	if r.student == nil || r.student.UserID != userID {
		return nil, domain.ErrProfessionalNotFound
	}
	copy := *r.student
	return &copy, nil
}

func (r *professionalRepositoryFake) SaveStudent(_ context.Context, value *domain.Student) error {
	r.saves++
	value.ID = uuid.New()
	r.student = value
	return nil
}

func (r *professionalRepositoryFake) DeleteStudent(context.Context, uuid.UUID) error {
	return nil
}

func (r *professionalRepositoryFake) GetLecturer(_ context.Context, userID uuid.UUID) (*domain.Lecturer, error) {
	if r.lecturer == nil || r.lecturer.UserID != userID {
		return nil, domain.ErrProfessionalNotFound
	}
	copy := *r.lecturer
	return &copy, nil
}

func (r *professionalRepositoryFake) SaveLecturer(_ context.Context, value *domain.Lecturer) error {
	r.saves++
	value.ID = uuid.New()
	r.lecturer = value
	return nil
}

func (r *professionalRepositoryFake) DeleteLecturer(context.Context, uuid.UUID) error {
	return nil
}

func (r *professionalRepositoryFake) GetEmployee(_ context.Context, userID uuid.UUID) (*domain.Employee, error) {
	if r.employee == nil || r.employee.UserID != userID {
		return nil, domain.ErrProfessionalNotFound
	}
	copy := *r.employee
	return &copy, nil
}

func (r *professionalRepositoryFake) SaveEmployee(_ context.Context, value *domain.Employee) error {
	r.saves++
	value.ID = uuid.New()
	r.employee = value
	return nil
}

func (r *professionalRepositoryFake) DeleteEmployee(context.Context, uuid.UUID) error {
	return nil
}

func TestSaveMyStudentAndLecturerShareOneProfile(t *testing.T) {
	userID := uuid.New()
	profileRepository := &profileRepositoryFake{}
	professionalRepository := &professionalRepositoryFake{}
	usecase := NewProfessionalUsecase(profileRepository, professionalRepository)

	student, err := usecase.SaveMyStudent(context.Background(), userID, dto.StudentRequest{
		NIM: "20240001", StudyProgramID: uuid.NewString(), CohortYear: 2024,
		AdmissionType: "regular", AcademicStatus: "active",
	})
	if err != nil {
		t.Fatalf("SaveMyStudent() error = %v", err)
	}
	lecturer, err := usecase.SaveMyLecturer(context.Background(), userID, dto.LecturerRequest{
		NIDN: stringPointer("1234567890"), Expertise: "Computer Science", AcademicPosition: "assistant",
	})
	if err != nil {
		t.Fatalf("SaveMyLecturer() error = %v", err)
	}
	if student.UserID != userID.String() || lecturer.UserID != userID.String() {
		t.Fatalf("professional records do not share profile user ID: student=%+v lecturer=%+v", student, lecturer)
	}
	if profileRepository.ensureCalls != 2 || professionalRepository.saves != 2 {
		t.Fatalf("unexpected persistence calls: ensure=%d saves=%d", profileRepository.ensureCalls, professionalRepository.saves)
	}
}

func TestSaveMyStudentRejectsInvalidExternalUUIDWithoutSaving(t *testing.T) {
	userID := uuid.New()
	profileRepository := &profileRepositoryFake{}
	professionalRepository := &professionalRepositoryFake{}
	usecase := NewProfessionalUsecase(profileRepository, professionalRepository)

	_, err := usecase.SaveMyStudent(context.Background(), userID, dto.StudentRequest{
		NIM: "20240001", StudyProgramID: "not-a-uuid", CohortYear: 2024,
		AdmissionType: "regular", AcademicStatus: "active",
	})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("SaveMyStudent() error = %v, want ErrInvalidInput", err)
	}
	if professionalRepository.saves != 0 || profileRepository.ensureCalls != 0 {
		t.Fatalf("invalid request changed persistence: saves=%d profile provisions=%d", professionalRepository.saves, profileRepository.ensureCalls)
	}
}

func TestSaveMyEmployeeRequiresCalendarDate(t *testing.T) {
	userID := uuid.New()
	profileRepository := &profileRepositoryFake{}
	professionalRepository := &professionalRepositoryFake{}
	usecase := NewProfessionalUsecase(profileRepository, professionalRepository)

	_, err := usecase.SaveMyEmployee(context.Background(), userID, dto.EmployeeRequest{
		EmployeeNumber: "EMP-1", EmploymentStatus: "active", Position: "staff",
		OrganizationalUnitID: uuid.NewString(), EmploymentStartDate: "2024-02-30",
	})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("SaveMyEmployee() error = %v, want ErrInvalidInput", err)
	}
	if professionalRepository.saves != 0 {
		t.Fatalf("invalid request reached repository: saves=%d", professionalRepository.saves)
	}
}

func stringPointer(value string) *string { return &value }
