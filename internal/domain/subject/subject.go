// Package subject owns subjects (Math, Literature, ...), the top level of the
// course catalog.
package subject

import (
	"strings"
	"time"

	"goschool/internal/domain/derr"
	"goschool/internal/domain/shared"
)

const maxNameLen = 100

type Subject struct {
	ID        int
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func New(in CreateInput) (*Subject, error) {
	s := &Subject{Name: strings.TrimSpace(in.Name)}
	f := derr.FieldErrors{}
	f.AddErr("name", shared.ValidateText(s.Name, maxNameLen))
	if err := f.Err(); err != nil {
		return nil, err
	}
	return s, nil
}

// ApplyUpdate validates and applies the update; s is unchanged on error.
func (s *Subject) ApplyUpdate(in UpdateInput) error {
	name := strings.TrimSpace(in.Name)
	f := derr.FieldErrors{}
	f.AddErr("name", shared.ValidateText(name, maxNameLen))
	if err := f.Err(); err != nil {
		return err
	}
	s.Name = name
	return nil
}
