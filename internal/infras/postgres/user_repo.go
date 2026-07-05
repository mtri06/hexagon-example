package postgres

import (
	"goschool/internal/domain/entity"

	"github.com/jmoiron/sqlx"
)

type UserRepository struct {
	db *sqlx.DB
}

func NewUserRepository(db *sqlx.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) CreateStudent(user *entity.User, student *entity.UserStudent) error {
	return nil
}
