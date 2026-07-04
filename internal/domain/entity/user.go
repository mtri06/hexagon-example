package entity

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
