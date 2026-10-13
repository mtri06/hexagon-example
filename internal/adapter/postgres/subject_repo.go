package postgres

import (
	"context"

	"github.com/uptrace/bun"

	"goschool/internal/domain/shared"
	"goschool/internal/domain/subject"
)

type subjectRow struct {
	bun.BaseModel `bun:"table:subjects,alias:s"`

	ID   int    `bun:"id,pk,autoincrement"`
	Name string `bun:"name,notnull"`
	timestamps
}

func (r *subjectRow) toDomain() *subject.Subject {
	return &subject.Subject{ID: r.ID, Name: r.Name, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
}

var subjectConstraints = constraintErrors{
	"subjects_name_key": subject.ErrNameTaken,
}

// Foreign keys that block deleting a subject.
var subjectDeleteConstraints = constraintErrors{
	"fk_courses_subject_id":          subject.ErrInUse,
	"fk_teacher_subjects_subject_id": subject.ErrInUse,
}

type SubjectRepo struct{ repo }

var _ subject.Repository = SubjectRepo{}

func NewSubjectRepo(db *bun.DB) SubjectRepo { return SubjectRepo{repo{db}} }

func (r SubjectRepo) Create(ctx context.Context, s *subject.Subject) error {
	row := &subjectRow{Name: s.Name}
	if _, err := r.idb(ctx).NewInsert().Model(row).Returning("id, created_at, updated_at").Exec(ctx); err != nil {
		return translate(err, subjectConstraints)
	}
	s.ID, s.CreatedAt, s.UpdatedAt = row.ID, row.CreatedAt, row.UpdatedAt
	return nil
}

func (r SubjectRepo) Update(ctx context.Context, s *subject.Subject) error {
	row := &subjectRow{ID: s.ID, Name: s.Name}
	res, err := r.idb(ctx).NewUpdate().Model(row).Column("name", "updated_at").WherePK().Exec(ctx)
	if err != nil {
		return translate(err, subjectConstraints)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return subject.ErrNotFound
	}
	s.UpdatedAt = row.UpdatedAt
	return nil
}

func (r SubjectRepo) Delete(ctx context.Context, id int) error {
	res, err := r.idb(ctx).NewDelete().Model((*subjectRow)(nil)).Where("id = ?", id).Exec(ctx)
	if err != nil {
		return translate(err, subjectDeleteConstraints)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return subject.ErrNotFound
	}
	return nil
}

func (r SubjectRepo) GetByID(ctx context.Context, id int) (*subject.Subject, error) {
	row := new(subjectRow)
	err := r.idb(ctx).NewSelect().Model(row).Where("s.id = ?", id).Scan(ctx)
	if isNoRows(err) {
		return nil, subject.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return row.toDomain(), nil
}

func (r SubjectRepo) NameExists(ctx context.Context, name string, excludeID int) (bool, error) {
	return r.idb(ctx).NewSelect().Model((*subjectRow)(nil)).
		Where("lower(s.name) = lower(?)", name).
		Where("s.id <> ?", excludeID).
		Exists(ctx)
}

func (r SubjectRepo) InUse(ctx context.Context, id int) (bool, error) {
	var inUse bool
	err := r.idb(ctx).NewSelect().
		ColumnExpr("EXISTS (SELECT 1 FROM courses WHERE subject_id = ?0)"+
			" OR EXISTS (SELECT 1 FROM teacher_subjects WHERE subject_id = ?0)", id).
		Scan(ctx, &inUse)
	return inUse, err
}

func (r SubjectRepo) List(ctx context.Context, f subject.Filter, p shared.PageRequest) ([]subject.Subject, int, error) {
	var rows []subjectRow
	q := r.idb(ctx).NewSelect().Model(&rows).
		OrderExpr("s.name ASC, s.id ASC").
		Limit(int64(p.PageSize)).
		Offset(int64(p.Offset()))
	if f.Query != nil {
		q = q.Where("s.name ILIKE ?", "%"+escapeLike(*f.Query)+"%")
	}
	total, err := q.ScanAndCount(ctx)
	if err != nil {
		return nil, 0, err
	}
	out := make([]subject.Subject, len(rows))
	for i := range rows {
		out[i] = *rows[i].toDomain()
	}
	return out, int(total), nil
}
