package course

type CreateInput struct {
	SubjectID int
	Name      string
	// Status defaults to active when nil.
	Status *Status
}

// UpdateInput is a partial update. SubjectID is deliberately absent: it is
// immutable.
type UpdateInput struct {
	Name   *string
	Status *Status
}

func (in UpdateInput) IsEmpty() bool { return in.Name == nil && in.Status == nil }

type Filter struct {
	SubjectID *int
	Status    *Status
	// Query matches the course name (case-insensitive, substring).
	Query *string
}
