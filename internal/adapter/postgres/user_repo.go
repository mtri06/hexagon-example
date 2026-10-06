package postgres

import (
	"context"
	"strings"
	"time"

	"github.com/uptrace/bun"

	"goschool/internal/domain/identity"
	"goschool/internal/domain/shared"
)

type userRow struct {
	bun.BaseModel `bun:"table:users,alias:u"`

	ID           int       `bun:"id,pk,autoincrement"`
	Username     string    `bun:"username,notnull"`
	PasswordHash string    `bun:"password_hash,notnull"`
	Email        *string   `bun:"email"`
	Role         string    `bun:"role,notnull"`
	Name         string    `bun:"name,notnull"`
	DateOfBirth  time.Time `bun:"date_of_birth,notnull,type:date"`
	Gender       string    `bun:"gender,notnull"`
	IsActive     bool      `bun:"is_active,notnull"`
	timestamps
}

func newUserRow(u *identity.User) *userRow {
	return &userRow{
		ID:           u.ID,
		Username:     u.Username,
		PasswordHash: u.PasswordHash,
		Email:        u.Email,
		Role:         string(u.Role),
		Name:         u.Name,
		DateOfBirth:  u.DateOfBirth,
		Gender:       string(u.Gender),
		IsActive:     u.IsActive,
		timestamps:   timestamps{CreatedAt: u.CreatedAt, UpdatedAt: u.UpdatedAt},
	}
}

func (r *userRow) toDomain() *identity.User {
	return &identity.User{
		ID:           r.ID,
		Username:     r.Username,
		PasswordHash: r.PasswordHash,
		Email:        r.Email,
		Role:         shared.Role(r.Role),
		Name:         r.Name,
		DateOfBirth:  shared.DateOf(r.DateOfBirth),
		Gender:       identity.Gender(r.Gender),
		IsActive:     r.IsActive,
		CreatedAt:    r.CreatedAt,
		UpdatedAt:    r.UpdatedAt,
	}
}

var userConstraints = constraintErrors{
	"users_username_key": identity.ErrUsernameTaken,
	"idx_users_email":    identity.ErrEmailTaken,
}

type UserRepo struct{ repo }

var _ identity.UserRepository = UserRepo{}

func NewUserRepo(db *bun.DB) UserRepo { return UserRepo{repo{db}} }

func (r UserRepo) Create(ctx context.Context, u *identity.User) error {
	row := newUserRow(u)
	_, err := r.idb(ctx).NewInsert().Model(row).Returning("id, created_at, updated_at").Exec(ctx)
	if err != nil {
		return translate(err, userConstraints)
	}
	u.ID, u.CreatedAt, u.UpdatedAt = row.ID, row.CreatedAt, row.UpdatedAt
	return nil
}

func (r UserRepo) Update(ctx context.Context, u *identity.User) error {
	row := newUserRow(u)
	res, err := r.idb(ctx).NewUpdate().Model(row).
		ExcludeColumn("id", "created_at").
		WherePK().
		Exec(ctx)
	if err != nil {
		return translate(err, userConstraints)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return identity.ErrUserNotFound
	}
	u.UpdatedAt = row.UpdatedAt
	return nil
}

func (r UserRepo) GetByID(ctx context.Context, id int) (*identity.User, error) {
	return r.getBy(ctx, "u.id = ?", id)
}

func (r UserRepo) GetByUsername(ctx context.Context, username string) (*identity.User, error) {
	return r.getBy(ctx, "u.username = ?", username)
}

func (r UserRepo) getBy(ctx context.Context, where string, arg any) (*identity.User, error) {
	row := new(userRow)
	err := r.idb(ctx).NewSelect().Model(row).Where(where, arg).Limit(1).Scan(ctx)
	if isNoRows(err) {
		return nil, identity.ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return row.toDomain(), nil
}

func (r UserRepo) UsernameExists(ctx context.Context, username string) (bool, error) {
	return r.idb(ctx).NewSelect().Model((*userRow)(nil)).Where("u.username = ?", username).Exists(ctx)
}

func (r UserRepo) EmailExists(ctx context.Context, email string, excludeUserID int) (bool, error) {
	return r.idb(ctx).NewSelect().Model((*userRow)(nil)).
		Where("u.email = ?", email).
		Where("u.id <> ?", excludeUserID).
		Exists(ctx)
}

func (r UserRepo) List(ctx context.Context, f identity.UserFilter, p shared.PageRequest) ([]identity.User, int, error) {
	var rows []userRow
	q := r.idb(ctx).NewSelect().Model(&rows).
		OrderExpr("u.id ASC").
		Limit(int64(p.PageSize)).
		Offset(int64(p.Offset()))
	if f.Role != nil {
		q = q.Where("u.role = ?", string(*f.Role))
	}
	if f.IsActive != nil {
		q = q.Where("u.is_active = ?", *f.IsActive)
	}
	if f.Query != nil {
		pattern := "%" + escapeLike(*f.Query) + "%"
		q = q.WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Where("u.username ILIKE ?", pattern).
				WhereOr("u.name ILIKE ?", pattern).
				WhereOr("u.email ILIKE ?", pattern)
		})
	}
	total, err := q.ScanAndCount(ctx)
	if err != nil {
		return nil, 0, err
	}
	users := make([]identity.User, len(rows))
	for i := range rows {
		users[i] = *rows[i].toDomain()
	}
	return users, int(total), nil
}

var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// escapeLike escapes LIKE wildcards so user input matches literally.
func escapeLike(s string) string { return likeEscaper.Replace(s) }
