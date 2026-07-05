package domain

import "goschool/internal/domain/entity"

type UserRepo interface {
	CreateStudent(user *entity.User, student *entity.UserStudent) error
}

type UserService struct {
	repo UserRepo
}

func NewUserService(repo UserRepo) *UserService {
	return &UserService{repo: repo}
}

type NewStudentInput struct {
}
type NewStudent struct {
}

func (s *UserService) NewStudent(input NewStudentInput) (*entity.User, *entity.UserStudent, error) {
	user, student, err := entity.NewStudent()
	if err != nil {
		return nil, nil, err
	}
	err = s.repo.CreateStudent(user, student)
	if err != nil {
		return nil, nil, err
	}
	return user, student, nil
}
