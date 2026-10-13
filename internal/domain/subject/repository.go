package subject

import (
	"context"

	"goschool/internal/domain/shared"
)

type Repository interface {
	// Create inserts s and sets its ID and timestamps.
	Create(ctx context.Context, s *Subject) error
	// Update persists s and refreshes UpdatedAt; ErrNotFound when missing.
	Update(ctx context.Context, s *Subject) error
	// Delete returns ErrNotFound when missing.
	Delete(ctx context.Context, id int) error
	// GetByID returns ErrNotFound when missing.
	GetByID(ctx context.Context, id int) (*Subject, error)
	// NameExists compares case-insensitively and ignores excludeID (0 for none).
	NameExists(ctx context.Context, name string, excludeID int) (bool, error)
	// InUse reports whether courses or teacher qualifications reference the subject.
	InUse(ctx context.Context, id int) (bool, error)
	List(ctx context.Context, f Filter, p shared.PageRequest) ([]Subject, int, error)
}
