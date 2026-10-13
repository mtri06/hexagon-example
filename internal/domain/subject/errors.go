package subject

import "goschool/internal/domain/derr"

const (
	CodeSubjectNotFound          = "subject_not_found"
	CodeSubjectNameAlreadyExists = "subject_name_already_exists"
	CodeSubjectInUse             = "subject_in_use"
)

var (
	ErrNotFound  = derr.NotFound(CodeSubjectNotFound, "Subject not found")
	ErrNameTaken = derr.Conflict(CodeSubjectNameAlreadyExists, "A subject with this name already exists")
	ErrInUse     = derr.Conflict(CodeSubjectInUse, "Subject is still used by courses or teacher qualifications")
)
