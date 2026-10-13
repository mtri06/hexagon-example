// Package course owns courses: concrete offerings within a subject, such as
// "Math 7 Advanced".
package course

import (
	"errors"
	"strings"
	"time"

	"goschool/internal/domain/derr"
	"goschool/internal/domain/shared"
)

const maxNameLen = 100

type Status string

const (
	StatusActive   Status = "active"
	StatusInactive Status = "inactive"
)

var errInvalidStatus = errors.New("must be one of: active, inactive")

// Validate returns an error whose message describes the allowed values.
func (s Status) Validate() error {
	switch s {
	case StatusActive, StatusInactive:
		return nil
	}
	return errInvalidStatus
}

// Course belongs to one subject for its whole life: teacher qualifications
// are keyed by (course, subject), so SubjectID is immutable.
type Course struct {
	ID        int
	SubjectID int
	Name      string
	Status    Status
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (c *Course) IsActive() bool { return c.Status == StatusActive }

// View is a course together with data read from related aggregates.
type View struct {
	Course
	SubjectName string
}

// New validates the input and builds a course; Status defaults to active.
func New(in CreateInput) (*Course, error) {
	c := &Course{
		SubjectID: in.SubjectID,
		Name:      strings.TrimSpace(in.Name),
		Status:    StatusActive,
	}
	if in.Status != nil {
		c.Status = *in.Status
	}

	f := derr.FieldErrors{}
	if c.SubjectID < 1 {
		f.Add("subject_id", "must be a positive integer")
	}
	f.AddErr("name", shared.ValidateText(c.Name, maxNameLen))
	f.AddErr("status", c.Status.Validate())
	if err := f.Err(); err != nil {
		return nil, err
	}
	return c, nil
}

// ApplyUpdate validates and applies a partial update; only provided fields
// are validated, and c is unchanged on error.
func (c *Course) ApplyUpdate(in UpdateInput) error {
	if in.IsEmpty() {
		return derr.ErrNothingToUpdate
	}
	next := *c
	f := derr.FieldErrors{}
	if in.Name != nil {
		next.Name = strings.TrimSpace(*in.Name)
		f.AddErr("name", shared.ValidateText(next.Name, maxNameLen))
	}
	if in.Status != nil {
		next.Status = *in.Status
		f.AddErr("status", next.Status.Validate())
	}
	if err := f.Err(); err != nil {
		return err
	}
	*c = next
	return nil
}
