package entity

const (
	RoleAdmin   = "admin"
	RoleTeacher = "teacher"
	RoleStudent = "student"

	GenderMale   = "male"
	GenderFemale = "female"
	GenderOther  = "other"

	WorkingStatusActive   = "active"
	WorkingStatusInactive = "inactive"
	WorkingStatusOnLeave  = "on_leave"
)

type User struct {
}

type UserStudent struct {
}

type UserTeacher struct {
}

func NewStudent() (*User, *UserStudent, error) {
	return &User{}, &UserStudent{}, nil
}

func NewTeacher() (*User, *UserTeacher, error) {
	return &User{}, &UserTeacher{}, nil
}
