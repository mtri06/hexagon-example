-- +goose Up
CREATE TABLE users (
  id            SERIAL       PRIMARY KEY,
  username      VARCHAR(100) NOT NULL UNIQUE,
  password_hash VARCHAR(255) NOT NULL,
  email         VARCHAR(100),
  role          VARCHAR(100) NOT NULL CHECK (role IN ('student', 'teacher', 'admin')),
  name          VARCHAR(255) NOT NULL,
  date_of_birth DATE         NOT NULL,
  gender        VARCHAR(10)  NOT NULL CHECK (gender IN ('male', 'female', 'other')),
  is_active     BOOLEAN      NOT NULL DEFAULT TRUE,
  created_at    TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at    TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE teacher_profiles (
  user_id        INT         PRIMARY KEY,
  hire_date      DATE        NOT NULL
);

CREATE TABLE student_profiles (
  user_id         INT         PRIMARY KEY,
  admission_date  DATE        NOT NULL,
  graduated_date  DATE
);

CREATE TABLE subjects (
  id         SERIAL       PRIMARY KEY,
  name       VARCHAR(100) NOT NULL UNIQUE,
  created_at TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE teacher_subjects (
  id         SERIAL      PRIMARY KEY,
  teacher_id INT         NOT NULL,
  subject_id INT         NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE classes (
  id          SERIAL        PRIMARY KEY,
  name        VARCHAR(100)  NOT NULL UNIQUE,
  grade       INT,
  created_at  TIMESTAMPTZ   NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at  TIMESTAMPTZ   NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE courses (
  id         SERIAL       PRIMARY KEY,
  subject_id INT          NOT NULL,
  name       VARCHAR(100) NOT NULL UNIQUE,   -- e.g. 'Math 7 Advanced'
  status     VARCHAR(20)  NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive')),
  created_at TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- a teacher may be qualified for a subject (e.g. Math) but not every course
-- in that subject (e.g. teaches Math 6-7 but not Math 8-9)
CREATE TABLE teacher_courses (
  id         SERIAL      PRIMARY KEY,
  teacher_id INT         NOT NULL,
  subject_id INT         NOT NULL,
  course_id  INT         NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE class_courses (
  id            SERIAL      PRIMARY KEY,
  class_id      INT         NOT NULL,
  course_id     INT         NOT NULL,
  academic_year INT         NOT NULL,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE teaching_assignments (
  id                  SERIAL      PRIMARY KEY,
  teacher_id          INT,
  class_id            INT         NOT NULL,
  course_id           INT         NOT NULL,
  academic_year       INT         NOT NULL,   -- e.g. 2025 for 2025-2026
  start_date          DATE        NOT NULL,
  end_date            DATE,
  created_at          TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at          TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE class_homeroom_assignments (
  id            SERIAL      PRIMARY KEY,
  teacher_id    INT         NOT NULL,
  class_id      INT         NOT NULL,
  academic_year INT         NOT NULL,   -- e.g. 2025 for 2025-2026
  start_date    DATE        NOT NULL,
  end_date      DATE,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE student_class_history (
  id            SERIAL      PRIMARY KEY,
  student_id    INT         NOT NULL,
  class_id      INT         NOT NULL,
  academic_year INT         NOT NULL,   -- e.g. 2025 for 2025-2026
  start_date    DATE        NOT NULL,
  end_date      DATE,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE timetable_entries (
  id            SERIAL      PRIMARY KEY,
  class_id      INT         NOT NULL,
  course_id     INT         NOT NULL,
  academic_year INT         NOT NULL,   -- e.g. 2025 for 2025-2026
  semester      SMALLINT    NOT NULL,
  day_of_week   SMALLINT    NOT NULL CHECK (day_of_week BETWEEN 1 AND 7),
  lesson_number SMALLINT    NOT NULL CHECK (lesson_number BETWEEN 1 AND 8),
  created_at    TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE score_columns (
  id            SERIAL        PRIMARY KEY,
  course_id     INT           NOT NULL,
  academic_year INT           NOT NULL,
  semester      SMALLINT      NOT NULL,
  name          VARCHAR(50)   NOT NULL,   -- 'TB', 'HK', '1T', ...
  ratio_percent NUMERIC(5,2)  NOT NULL CHECK (ratio_percent > 0 AND ratio_percent <= 100),
  created_at    TIMESTAMPTZ   NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at    TIMESTAMPTZ   NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE scores (
  id              SERIAL      PRIMARY KEY,
  student_id      INT         NOT NULL,
  score_column_id INT         NOT NULL,
  score           NUMERIC(4,2) NOT NULL CHECK (score BETWEEN 0 AND 10),
  recorded_by     INT         NOT NULL,   -- user (teacher/admin) who entered/uploaded the score
  created_at      TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE tokens (
	id 							SERIAL 			PRIMARY KEY,
	body 						TEXT 				NOT NULL,
	user_id 				INT 				NOT NULL,
	type 						VARCHAR(50) NOT NULL CHECK (type IN ('refresh_token', 'password_update_token', 'email_verification_token')),
	expires_at 			TIMESTAMPTZ NOT NULL,
	is_revoked 			BOOLEAN 		NOT NULL DEFAULT FALSE,
	is_used 				BOOLEAN 		NOT NULL DEFAULT FALSE,
	is_blacklisted 	BOOLEAN 		NOT NULL DEFAULT FALSE,
	created_at 			TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at 			TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);


CREATE UNIQUE INDEX idx_users_email ON users (email) WHERE email IS NOT NULL;
CREATE INDEX idx_tokens_body ON tokens (body);
CREATE UNIQUE INDEX idx_teacher_subjects ON teacher_subjects (teacher_id, subject_id);
CREATE UNIQUE INDEX idx_teacher_courses_teacher_course ON teacher_courses (teacher_id, course_id);
CREATE UNIQUE INDEX idx_class_courses ON class_courses (class_id, course_id, academic_year);
CREATE INDEX idx_teaching_assignments_class_id_academic_year ON teaching_assignments (class_id, academic_year);
CREATE INDEX idx_teaching_assignments_teacher_id_academic_year ON teaching_assignments (teacher_id, academic_year);
CREATE INDEX idx_student_class_history_class_id ON student_class_history (class_id);
-- at most one open (not yet ended) enrollment per student per academic year
CREATE UNIQUE INDEX idx_student_class_history_open ON student_class_history (student_id, academic_year) WHERE end_date IS NULL;
-- at most one open (not yet ended) homeroom assignment per class per academic year
CREATE UNIQUE INDEX idx_class_homeroom_assignments_open ON class_homeroom_assignments (class_id, academic_year) WHERE end_date IS NULL;
CREATE UNIQUE INDEX idx_timetable_class_slot ON timetable_entries (academic_year, semester, class_id, day_of_week, lesson_number);
CREATE UNIQUE INDEX idx_scores_unique_entry ON scores (student_id, score_column_id);


-- extra unique index needed so (course_id, subject_id) can be a composite FK target
-- (id is already unique via PK; this just exposes subject_id alongside it)
CREATE UNIQUE INDEX idx_courses_id_subject_id ON courses (id, subject_id);

ALTER TABLE teacher_profiles ADD CONSTRAINT fk_teacher_profiles_user_id FOREIGN KEY (user_id)
REFERENCES users(id) ON DELETE RESTRICT;
ALTER TABLE student_profiles ADD CONSTRAINT fk_student_profiles_user_id FOREIGN KEY (user_id)
REFERENCES users(id) ON DELETE RESTRICT;
ALTER TABLE teacher_subjects ADD CONSTRAINT fk_teacher_subjects_teacher_id FOREIGN KEY (teacher_id)
REFERENCES teacher_profiles(user_id) ON DELETE RESTRICT;
ALTER TABLE teacher_subjects ADD CONSTRAINT fk_teacher_subjects_subject_id FOREIGN KEY (subject_id)
REFERENCES subjects(id) ON DELETE RESTRICT;
ALTER TABLE courses ADD CONSTRAINT fk_courses_subject_id FOREIGN KEY (subject_id)
REFERENCES subjects(id) ON DELETE RESTRICT;
ALTER TABLE teacher_courses ADD CONSTRAINT fk_teacher_courses_teacher_subject FOREIGN KEY (teacher_id, subject_id)
REFERENCES teacher_subjects(teacher_id, subject_id) ON DELETE RESTRICT;
ALTER TABLE teacher_courses ADD CONSTRAINT fk_teacher_courses_course_subject FOREIGN KEY (course_id, subject_id)
REFERENCES courses(id, subject_id) ON DELETE RESTRICT;
ALTER TABLE class_courses ADD CONSTRAINT fk_class_courses_class_id FOREIGN KEY (class_id)
REFERENCES classes(id) ON DELETE RESTRICT;
ALTER TABLE class_courses ADD CONSTRAINT fk_class_courses_course_id FOREIGN KEY (course_id)
REFERENCES courses(id) ON DELETE RESTRICT;
ALTER TABLE teaching_assignments ADD CONSTRAINT fk_teaching_assignments_course_class_academic_year FOREIGN KEY (class_id, course_id, academic_year)
REFERENCES class_courses(class_id, course_id, academic_year) ON DELETE RESTRICT;
ALTER TABLE teaching_assignments ADD CONSTRAINT fk_teaching_assignments_teacher_course FOREIGN KEY (teacher_id, course_id)
REFERENCES teacher_courses(teacher_id, course_id) ON DELETE RESTRICT;
ALTER TABLE class_homeroom_assignments ADD CONSTRAINT fk_class_homeroom_assignments_teacher_id FOREIGN KEY (teacher_id)
REFERENCES teacher_profiles(user_id) ON DELETE RESTRICT;
ALTER TABLE class_homeroom_assignments ADD CONSTRAINT fk_class_homeroom_assignments_class_id FOREIGN KEY (class_id)
REFERENCES classes(id) ON DELETE RESTRICT;
ALTER TABLE student_class_history ADD CONSTRAINT fk_student_class_history_student_id FOREIGN KEY (student_id)
REFERENCES student_profiles(user_id) ON DELETE RESTRICT;
ALTER TABLE student_class_history ADD CONSTRAINT fk_student_class_history_class_id FOREIGN KEY (class_id)
REFERENCES classes(id) ON DELETE RESTRICT;
ALTER TABLE timetable_entries ADD CONSTRAINT fk_timetable_entries_assignment FOREIGN KEY (class_id, course_id, academic_year)
REFERENCES class_courses(class_id, course_id, academic_year) ON DELETE RESTRICT;
ALTER TABLE tokens ADD CONSTRAINT fk_tokens_user_id FOREIGN KEY (user_id)
REFERENCES users(id) ON DELETE CASCADE;
ALTER TABLE score_columns ADD CONSTRAINT fk_score_columns_course_id FOREIGN KEY (course_id)
REFERENCES courses(id) ON DELETE RESTRICT;
ALTER TABLE scores ADD CONSTRAINT fk_scores_student_id FOREIGN KEY (student_id)
REFERENCES student_profiles(user_id) ON DELETE RESTRICT;
ALTER TABLE scores ADD CONSTRAINT fk_scores_score_column_id FOREIGN KEY (score_column_id)
REFERENCES score_columns(id) ON DELETE CASCADE;
ALTER TABLE scores ADD CONSTRAINT fk_scores_recorded_by FOREIGN KEY (recorded_by)
REFERENCES users(id) ON DELETE RESTRICT;

-- +goose Down

ALTER TABLE scores DROP CONSTRAINT fk_scores_recorded_by;
ALTER TABLE scores DROP CONSTRAINT fk_scores_score_column_id;
ALTER TABLE scores DROP CONSTRAINT fk_scores_student_id;
ALTER TABLE score_columns DROP CONSTRAINT fk_score_columns_course_id;
ALTER TABLE tokens DROP CONSTRAINT fk_tokens_user_id;
ALTER TABLE timetable_entries DROP CONSTRAINT fk_timetable_entries_assignment;
ALTER TABLE student_class_history DROP CONSTRAINT fk_student_class_history_class_id;
ALTER TABLE student_class_history DROP CONSTRAINT fk_student_class_history_student_id;
ALTER TABLE class_homeroom_assignments DROP CONSTRAINT fk_class_homeroom_assignments_class_id;
ALTER TABLE class_homeroom_assignments DROP CONSTRAINT fk_class_homeroom_assignments_teacher_id;
ALTER TABLE teaching_assignments DROP CONSTRAINT fk_teaching_assignments_teacher_course;
ALTER TABLE teaching_assignments DROP CONSTRAINT fk_teaching_assignments_course_class_academic_year;
ALTER TABLE class_courses DROP CONSTRAINT fk_class_courses_course_id;
ALTER TABLE class_courses DROP CONSTRAINT fk_class_courses_class_id;
ALTER TABLE teacher_courses DROP CONSTRAINT fk_teacher_courses_course_subject;
ALTER TABLE teacher_courses DROP CONSTRAINT fk_teacher_courses_teacher_subject;
ALTER TABLE courses DROP CONSTRAINT fk_courses_subject_id;
ALTER TABLE teacher_subjects DROP CONSTRAINT fk_teacher_subjects_subject_id;
ALTER TABLE teacher_subjects DROP CONSTRAINT fk_teacher_subjects_teacher_id;
ALTER TABLE student_profiles DROP CONSTRAINT fk_student_profiles_user_id;
ALTER TABLE teacher_profiles DROP CONSTRAINT fk_teacher_profiles_user_id;

DROP INDEX IF EXISTS idx_users_email;
DROP INDEX IF EXISTS idx_tokens_body;
DROP INDEX IF EXISTS idx_teacher_subjects;
DROP INDEX IF EXISTS idx_teacher_courses_teacher_course;
DROP INDEX IF EXISTS idx_class_courses;
DROP INDEX IF EXISTS idx_teaching_assignments_class_id_academic_year;
DROP INDEX IF EXISTS idx_teaching_assignments_teacher_id_academic_year;
DROP INDEX IF EXISTS idx_student_class_history_class_id;
DROP INDEX IF EXISTS idx_student_class_history_open;
DROP INDEX IF EXISTS idx_class_homeroom_assignments_open;
DROP INDEX IF EXISTS idx_timetable_class_slot;
DROP INDEX IF EXISTS idx_courses_id_subject_id;
DROP INDEX IF EXISTS idx_scores_unique_entry;

DROP TABLE IF EXISTS tokens;
DROP TABLE IF EXISTS scores;
DROP TABLE IF EXISTS score_columns;
DROP TABLE IF EXISTS timetable_entries;
DROP TABLE IF EXISTS student_class_history;
DROP TABLE IF EXISTS class_homeroom_assignments;
DROP TABLE IF EXISTS teaching_assignments;
DROP TABLE IF EXISTS class_courses;
DROP TABLE IF EXISTS teacher_courses;
DROP TABLE IF EXISTS courses;
DROP TABLE IF EXISTS classes;
DROP TABLE IF EXISTS teacher_subjects;
DROP TABLE IF EXISTS subjects;
DROP TABLE IF EXISTS student_profiles;
DROP TABLE IF EXISTS teacher_profiles;
DROP TABLE IF EXISTS users;
