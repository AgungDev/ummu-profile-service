package repositories

import (
	"context"
	"errors"
	"fmt"

	"github.com/AgungDev/ummu-profile-service/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ProfessionalRepository interface {
	GetStudent(ctx context.Context, userID uuid.UUID) (*domain.Student, error)
	SaveStudent(ctx context.Context, student *domain.Student) error
	DeleteStudent(ctx context.Context, userID uuid.UUID) error
	GetLecturer(ctx context.Context, userID uuid.UUID) (*domain.Lecturer, error)
	SaveLecturer(ctx context.Context, lecturer *domain.Lecturer) error
	DeleteLecturer(ctx context.Context, userID uuid.UUID) error
	GetEmployee(ctx context.Context, userID uuid.UUID) (*domain.Employee, error)
	SaveEmployee(ctx context.Context, employee *domain.Employee) error
	DeleteEmployee(ctx context.Context, userID uuid.UUID) error
}

type professionalRepository struct {
	db *gorm.DB
}

func NewProfessionalRepository(db *gorm.DB) ProfessionalRepository {
	return &professionalRepository{db: db}
}

func (r *professionalRepository) GetStudent(ctx context.Context, userID uuid.UUID) (*domain.Student, error) {
	var student domain.Student
	if err := r.db.WithContext(ctx).First(&student, "user_id = ?", userID).Error; err != nil {
		return nil, professionalReadError("get student", err)
	}
	return &student, nil
}

func (r *professionalRepository) SaveStudent(ctx context.Context, student *domain.Student) error {
	err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"nim", "study_program_id", "cohort_year", "admission_type", "academic_status", "updated_at",
		}),
	}).Create(student).Error
	return professionalWriteError("save student", err)
}

func (r *professionalRepository) DeleteStudent(ctx context.Context, userID uuid.UUID) error {
	return deleteProfessional(ctx, r.db, &domain.Student{}, userID)
}

func (r *professionalRepository) GetLecturer(ctx context.Context, userID uuid.UUID) (*domain.Lecturer, error) {
	var lecturer domain.Lecturer
	if err := r.db.WithContext(ctx).First(&lecturer, "user_id = ?", userID).Error; err != nil {
		return nil, professionalReadError("get lecturer", err)
	}
	return &lecturer, nil
}

func (r *professionalRepository) SaveLecturer(ctx context.Context, lecturer *domain.Lecturer) error {
	err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"nidn", "nidk", "expertise", "academic_position", "updated_at",
		}),
	}).Create(lecturer).Error
	return professionalWriteError("save lecturer", err)
}

func (r *professionalRepository) DeleteLecturer(ctx context.Context, userID uuid.UUID) error {
	return deleteProfessional(ctx, r.db, &domain.Lecturer{}, userID)
}

func (r *professionalRepository) GetEmployee(ctx context.Context, userID uuid.UUID) (*domain.Employee, error) {
	var employee domain.Employee
	if err := r.db.WithContext(ctx).First(&employee, "user_id = ?", userID).Error; err != nil {
		return nil, professionalReadError("get employee", err)
	}
	return &employee, nil
}

func (r *professionalRepository) SaveEmployee(ctx context.Context, employee *domain.Employee) error {
	err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"employee_number", "employment_status", "position",
			"organizational_unit_id", "employment_start_date", "updated_at",
		}),
	}).Create(employee).Error
	return professionalWriteError("save employee", err)
}

func (r *professionalRepository) DeleteEmployee(ctx context.Context, userID uuid.UUID) error {
	return deleteProfessional(ctx, r.db, &domain.Employee{}, userID)
}

func deleteProfessional(ctx context.Context, db *gorm.DB, model interface{}, userID uuid.UUID) error {
	result := db.WithContext(ctx).Where("user_id = ?", userID).Delete(model)
	if result.Error != nil {
		return fmt.Errorf("delete professional record: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return domain.ErrProfessionalNotFound
	}
	return nil
}

func professionalReadError(operation string, err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.ErrProfessionalNotFound
	}
	return fmt.Errorf("%s: %w", operation, err)
}

func professionalWriteError(operation string, err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return domain.ErrConflict
	}
	return fmt.Errorf("%s: %w", operation, err)
}
