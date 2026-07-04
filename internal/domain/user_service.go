package domain

type UserRepo interface {
}

type UserSvc interface {
}

type UserService struct {
	repo UserRepo
}

func NewUserService(repo UserRepo) *UserService {
	return &UserService{repo: repo}
}
