package course

import (
	"context"

	"goschool/internal/domain/shared"
	"goschool/internal/domain/subject"
)

type Repository interface {
	// Create inserts c and sets its ID and timestamps.
	Create(ctx context.Context, c *Course) error
	// Update persists the mutable fields of c (never SubjectID) and refreshes
	// UpdatedAt; ErrNotFound when missing.
	Update(ctx context.Context, c *Course) error
	// Delete returns ErrNotFound when missing.
	Delete(ctx context.Context, id int) error
	// GetByID returns ErrNotFound when missing.
	GetByID(ctx context.Context, id int) (*View, error)
	// NameExists compares case-insensitively and ignores excludeID (0 for none).
	NameExists(ctx context.Context, name string, excludeID int) (bool, error)
	// InUse reports whether curricula, teacher qualifications or score columns
	// reference the course.
	InUse(ctx context.Context, id int) (bool, error)
	List(ctx context.Context, f Filter, p shared.PageRequest) ([]View, int, error)
}

// SubjectReader is the part of the subject aggregate a course needs.
type SubjectReader interface {
	// GetByID returns subject.ErrNotFound when missing.
	GetByID(ctx context.Context, id int) (*subject.Subject, error)
}
