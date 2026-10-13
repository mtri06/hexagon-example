package course

import "goschool/internal/domain/derr"

const (
	CodeCourseNotFound          = "course_not_found"
	CodeCourseNameAlreadyExists = "course_name_already_exists"
	CodeCourseInUse             = "course_in_use"
)

var (
	ErrNotFound  = derr.NotFound(CodeCourseNotFound, "Course not found")
	ErrNameTaken = derr.Conflict(CodeCourseNameAlreadyExists, "A course with this name already exists")
	ErrInUse     = derr.Conflict(CodeCourseInUse, "Course is still used by curricula, teacher qualifications or score columns; deactivate it instead")
)
