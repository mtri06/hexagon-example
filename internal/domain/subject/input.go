package subject

type CreateInput struct {
	Name string
}

// UpdateInput renames a subject; name is the subject's only editable field.
type UpdateInput struct {
	Name string
}

type Filter struct {
	// Query matches the name (case-insensitive, substring).
	Query *string
}
