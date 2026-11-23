CREATE TABLE students (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL UNIQUE REFERENCES user_profiles(user_id) ON DELETE CASCADE,
    nim VARCHAR(20) NOT NULL UNIQUE,
    study_program_id UUID NOT NULL,
    cohort_year INTEGER NOT NULL,
    admission_type VARCHAR(32) NOT NULL,
    academic_status VARCHAR(32) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE lecturers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL UNIQUE REFERENCES user_profiles(user_id) ON DELETE CASCADE,
    nidn VARCHAR(32) UNIQUE,
    nidk VARCHAR(32) UNIQUE,
    expertise VARCHAR(255) NOT NULL,
    academic_position VARCHAR(128) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT lecturers_nidn_or_nidk_required CHECK (nidn IS NOT NULL OR nidk IS NOT NULL)
);

CREATE TABLE employees (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL UNIQUE REFERENCES user_profiles(user_id) ON DELETE CASCADE,
    employee_number VARCHAR(64) NOT NULL UNIQUE,
    employment_status VARCHAR(32) NOT NULL,
    position VARCHAR(128) NOT NULL,
    organizational_unit_id UUID NOT NULL,
    employment_start_date DATE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
