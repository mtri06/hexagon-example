package postgres_test

import (
	"testing"
	"time"

	"github.com/uptrace/bun"

	"goschool/internal/adapter/postgres"
	"goschool/internal/adapter/postgres/pgtest"
	"goschool/internal/domain/course"
	"goschool/internal/domain/derr"
	"goschool/internal/domain/shared"
	"goschool/internal/domain/subject"
)

func mustExec(t *testing.T, db *bun.DB, query string, args ...any) {
	t.Helper()
	if _, err := db.ExecContext(ctx, query, args...); err != nil {
		t.Fatalf("%s: %v", query, err)
	}
}

func createSubject(t *testing.T, repo postgres.SubjectRepo, name string) *subject.Subject {
	t.Helper()
	s := &subject.Subject{Name: name}
	if err := repo.Create(ctx, s); err != nil {
		t.Fatal(err)
	}
	return s
}

func createCourse(t *testing.T, repo postgres.CourseRepo, subjectID int, name string, status course.Status) *course.Course {
	t.Helper()
	c := &course.Course{SubjectID: subjectID, Name: name, Status: status}
	if err := repo.Create(ctx, c); err != nil {
		t.Fatal(err)
	}
	return c
}

func TestSubjectRepo(t *testing.T) {
	db := pgtest.DB(t)
	repo := postgres.NewSubjectRepo(db)

	math := createSubject(t, repo, "Math")
	createSubject(t, repo, "Physics")

	if err := repo.Create(ctx, &subject.Subject{Name: "Math"}); !derr.Is(err, subject.CodeSubjectNameAlreadyExists) {
		t.Fatalf("duplicate: got %v", err)
	}
	if ok, _ := repo.NameExists(ctx, "MATH", 0); !ok {
		t.Fatal("NameExists must be case-insensitive")
	}
	if ok, _ := repo.NameExists(ctx, "math", math.ID); ok {
		t.Fatal("NameExists must ignore the excluded id")
	}

	before := math.UpdatedAt
	math.Name = "Mathematics"
	if err := repo.Update(ctx, math); err != nil {
		t.Fatal(err)
	}
	got, _ := repo.GetByID(ctx, math.ID)
	if got.Name != "Mathematics" || !got.UpdatedAt.After(before) {
		t.Fatalf("update not persisted: %+v", got)
	}

	items, total, err := repo.List(ctx, subject.Filter{Query: ptr("phys")}, shared.PageRequest{Page: 1, PageSize: 10})
	if err != nil || total != 1 || items[0].Name != "Physics" {
		t.Fatalf("list: %v %d %+v", err, total, items)
	}

	if err := repo.Delete(ctx, 9999); !derr.Is(err, subject.CodeSubjectNotFound) {
		t.Fatalf("delete missing: got %v", err)
	}
	if _, err := repo.GetByID(ctx, 9999); !derr.Is(err, subject.CodeSubjectNotFound) {
		t.Fatalf("get missing: got %v", err)
	}
}

func TestSubjectInUse(t *testing.T) {
	db := pgtest.DB(t)
	subjects := postgres.NewSubjectRepo(db)
	courses := postgres.NewCourseRepo(db)

	math := createSubject(t, subjects, "Math")
	if inUse, err := subjects.InUse(ctx, math.ID); err != nil || inUse {
		t.Fatalf("fresh subject: %v %v", inUse, err)
	}
	createCourse(t, courses, math.ID, "Math 7", course.StatusActive)
	if inUse, _ := subjects.InUse(ctx, math.ID); !inUse {
		t.Fatal("subject with a course must be in use")
	}
	// The FK backstop maps to the same domain error.
	if err := subjects.Delete(ctx, math.ID); !derr.Is(err, subject.CodeSubjectInUse) {
		t.Fatalf("delete in-use subject: got %v", err)
	}
}

func TestCourseRepo(t *testing.T) {
	db := pgtest.DB(t)
	subjects := postgres.NewSubjectRepo(db)
	repo := postgres.NewCourseRepo(db)

	math := createSubject(t, subjects, "Math")
	lit := createSubject(t, subjects, "Literature")
	m7 := createCourse(t, repo, math.ID, "Math 7", course.StatusActive)
	createCourse(t, repo, math.ID, "Math 8", course.StatusInactive)
	createCourse(t, repo, lit.ID, "Literature 7", course.StatusActive)

	if err := repo.Create(ctx, &course.Course{SubjectID: math.ID, Name: "Math 7", Status: course.StatusActive}); !derr.Is(err, course.CodeCourseNameAlreadyExists) {
		t.Fatalf("duplicate: got %v", err)
	}
	if err := repo.Create(ctx, &course.Course{SubjectID: 9999, Name: "Ghost", Status: course.StatusActive}); !derr.Is(err, subject.CodeSubjectNotFound) {
		t.Fatalf("unknown subject: got %v", err)
	}

	v, err := repo.GetByID(ctx, m7.ID)
	if err != nil || v.SubjectName != "Math" || v.Status != course.StatusActive {
		t.Fatalf("get: %+v %v", v, err)
	}

	// Update never changes subject_id, even if the entity says otherwise.
	v.Name, v.Status, v.SubjectID = "Math 7 Advanced", course.StatusInactive, lit.ID
	time.Sleep(time.Millisecond)
	if err := repo.Update(ctx, &v.Course); err != nil {
		t.Fatal(err)
	}
	got, _ := repo.GetByID(ctx, m7.ID)
	if got.Name != "Math 7 Advanced" || got.Status != course.StatusInactive || got.SubjectID != math.ID {
		t.Fatalf("update: %+v", got)
	}

	page := shared.PageRequest{Page: 1, PageSize: 10}
	items, total, _ := repo.List(ctx, course.Filter{SubjectID: &math.ID}, page)
	if total != 2 || items[0].SubjectName != "Math" {
		t.Fatalf("subject filter: %d %+v", total, items)
	}
	active := course.StatusActive
	_, total, _ = repo.List(ctx, course.Filter{Status: &active}, page)
	if total != 1 {
		t.Fatalf("status filter: %d", total)
	}
	_, total, _ = repo.List(ctx, course.Filter{Query: ptr("advanced")}, page)
	if total != 1 {
		t.Fatalf("query filter: %d", total)
	}
}

func TestCourseInUse(t *testing.T) {
	db := pgtest.DB(t)
	subjects := postgres.NewSubjectRepo(db)
	repo := postgres.NewCourseRepo(db)

	math := createSubject(t, subjects, "Math")
	m7 := createCourse(t, repo, math.ID, "Math 7", course.StatusActive)
	m8 := createCourse(t, repo, math.ID, "Math 8", course.StatusActive)

	// Put Math 7 in a class curriculum. The class aggregate doesn't exist yet,
	// so insert the rows directly.
	mustExec(t, db, "INSERT INTO classes (name) VALUES ('7A')")
	mustExec(t, db, "INSERT INTO class_courses (class_id, course_id, academic_year) VALUES (1, ?, 2025)", m7.ID)

	if inUse, _ := repo.InUse(ctx, m7.ID); !inUse {
		t.Fatal("course in a curriculum must be in use")
	}
	if inUse, _ := repo.InUse(ctx, m8.ID); inUse {
		t.Fatal("unreferenced course must not be in use")
	}
	if err := repo.Delete(ctx, m7.ID); !derr.Is(err, course.CodeCourseInUse) {
		t.Fatalf("delete in-use course: got %v", err)
	}
	if err := repo.Delete(ctx, m8.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.GetByID(ctx, m8.ID); !derr.Is(err, course.CodeCourseNotFound) {
		t.Fatalf("deleted course: got %v", err)
	}
}
