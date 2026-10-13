package postgres

import (
	"context"

	"github.com/uptrace/bun"

	"goschool/internal/domain/course"
	"goschool/internal/domain/shared"
	"goschool/internal/domain/subject"
)

type courseRow struct {
	bun.BaseModel `bun:"table:courses,alias:c"`

	ID        int    `bun:"id,pk,autoincrement"`
	SubjectID int    `bun:"subject_id,notnull"`
	Name      string `bun:"name,notnull"`
	Status    string `bun:"status,notnull"`
	timestamps
}

// courseViewRow is a course joined with its subject's name.
type courseViewRow struct {
	courseRow   `bun:",extend"`
	SubjectName string `bun:"subject_name,scanonly"`
}

func (r *courseViewRow) toDomain() *course.View {
	return &course.View{
		Course: course.Course{
			ID:        r.ID,
			SubjectID: r.SubjectID,
			Name:      r.Name,
			Status:    course.Status(r.Status),
			CreatedAt: r.CreatedAt,
			UpdatedAt: r.UpdatedAt,
		},
		SubjectName: r.SubjectName,
	}
}

var courseConstraints = constraintErrors{
	"courses_name_key": course.ErrNameTaken,
	// Inserting a course whose subject was deleted concurrently.
	"fk_courses_subject_id": subject.ErrNotFound,
}

// Foreign keys that block deleting a course. Teaching assignments and
// timetable entries reference class_courses, so they are covered too.
var courseDeleteConstraints = constraintErrors{
	"fk_teacher_courses_course_subject": course.ErrInUse,
	"fk_class_courses_course_id":        course.ErrInUse,
	"fk_score_columns_course_id":        course.ErrInUse,
}

type CourseRepo struct{ repo }

var _ course.Repository = CourseRepo{}

func NewCourseRepo(db *bun.DB) CourseRepo { return CourseRepo{repo{db}} }

func (r CourseRepo) Create(ctx context.Context, c *course.Course) error {
	row := &courseRow{SubjectID: c.SubjectID, Name: c.Name, Status: string(c.Status)}
	if _, err := r.idb(ctx).NewInsert().Model(row).Returning("id, created_at, updated_at").Exec(ctx); err != nil {
		return translate(err, courseConstraints)
	}
	c.ID, c.CreatedAt, c.UpdatedAt = row.ID, row.CreatedAt, row.UpdatedAt
	return nil
}

func (r CourseRepo) Update(ctx context.Context, c *course.Course) error {
	row := &courseRow{ID: c.ID, Name: c.Name, Status: string(c.Status)}
	// subject_id is immutable and deliberately not in the column list.
	res, err := r.idb(ctx).NewUpdate().Model(row).Column("name", "status", "updated_at").WherePK().Exec(ctx)
	if err != nil {
		return translate(err, courseConstraints)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return course.ErrNotFound
	}
	c.UpdatedAt = row.UpdatedAt
	return nil
}

func (r CourseRepo) Delete(ctx context.Context, id int) error {
	res, err := r.idb(ctx).NewDelete().Model((*courseRow)(nil)).Where("id = ?", id).Exec(ctx)
	if err != nil {
		return translate(err, courseDeleteConstraints)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return course.ErrNotFound
	}
	return nil
}

func (r CourseRepo) selectViews(ctx context.Context, rows *[]courseViewRow) *bun.SelectQuery {
	return r.idb(ctx).NewSelect().Model(rows).
		ColumnExpr("c.*").
		ColumnExpr("s.name AS subject_name").
		Join("JOIN subjects AS s ON s.id = c.subject_id")
}

func (r CourseRepo) GetByID(ctx context.Context, id int) (*course.View, error) {
	var rows []courseViewRow
	if err := r.selectViews(ctx, &rows).Where("c.id = ?", id).Scan(ctx); err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, course.ErrNotFound
	}
	return rows[0].toDomain(), nil
}

func (r CourseRepo) NameExists(ctx context.Context, name string, excludeID int) (bool, error) {
	return r.idb(ctx).NewSelect().Model((*courseRow)(nil)).
		Where("lower(c.name) = lower(?)", name).
		Where("c.id <> ?", excludeID).
		Exists(ctx)
}

func (r CourseRepo) InUse(ctx context.Context, id int) (bool, error) {
	var inUse bool
	err := r.idb(ctx).NewSelect().
		ColumnExpr("EXISTS (SELECT 1 FROM teacher_courses WHERE course_id = ?0)"+
			" OR EXISTS (SELECT 1 FROM class_courses WHERE course_id = ?0)"+
			" OR EXISTS (SELECT 1 FROM score_columns WHERE course_id = ?0)", id).
		Scan(ctx, &inUse)
	return inUse, err
}

func (r CourseRepo) List(ctx context.Context, f course.Filter, p shared.PageRequest) ([]course.View, int, error) {
	var rows []courseViewRow
	q := r.selectViews(ctx, &rows).
		OrderExpr("c.name ASC, c.id ASC").
		Limit(int64(p.PageSize)).
		Offset(int64(p.Offset()))
	if f.SubjectID != nil {
		q = q.Where("c.subject_id = ?", *f.SubjectID)
	}
	if f.Status != nil {
		q = q.Where("c.status = ?", string(*f.Status))
	}
	if f.Query != nil {
		q = q.Where("c.name ILIKE ?", "%"+escapeLike(*f.Query)+"%")
	}
	total, err := q.ScanAndCount(ctx)
	if err != nil {
		return nil, 0, err
	}
	out := make([]course.View, len(rows))
	for i := range rows {
		out[i] = *rows[i].toDomain()
	}
	return out, int(total), nil
}
