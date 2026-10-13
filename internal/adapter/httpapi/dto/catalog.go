package dto

import (
	"time"

	"goschool/internal/domain/course"
	"goschool/internal/domain/subject"
)

type Subject struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func NewSubject(s *subject.Subject) Subject {
	return Subject{ID: s.ID, Name: s.Name, CreatedAt: s.CreatedAt, UpdatedAt: s.UpdatedAt}
}

// SubjectInput is the body of both POST and PATCH /subjects.
type SubjectInput struct {
	Name string `json:"name" validate:"required"`
}

type Course struct {
	ID          int       `json:"id"`
	SubjectID   int       `json:"subject_id"`
	SubjectName string    `json:"subject_name"`
	Name        string    `json:"name"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func NewCourse(v *course.View) Course {
	return Course{
		ID:          v.ID,
		SubjectID:   v.SubjectID,
		SubjectName: v.SubjectName,
		Name:        v.Name,
		Status:      string(v.Status),
		CreatedAt:   v.CreatedAt,
		UpdatedAt:   v.UpdatedAt,
	}
}

type CourseCreateRequest struct {
	SubjectID int     `json:"subject_id" validate:"required"`
	Name      string  `json:"name" validate:"required"`
	Status    *string `json:"status" validate:"omitnil,oneof=active inactive"`
}

func (r CourseCreateRequest) ToDomain() course.CreateInput {
	return course.CreateInput{SubjectID: r.SubjectID, Name: r.Name, Status: toStatus(r.Status)}
}

type CourseUpdateRequest struct {
	Name   *string `json:"name"`
	Status *string `json:"status" validate:"omitnil,oneof=active inactive"`
}

func (r CourseUpdateRequest) ToDomain() course.UpdateInput {
	return course.UpdateInput{Name: r.Name, Status: toStatus(r.Status)}
}

func toStatus(s *string) *course.Status {
	if s == nil {
		return nil
	}
	st := course.Status(*s)
	return &st
}
