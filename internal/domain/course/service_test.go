package course

import (
	"context"
	"strings"
	"testing"

	"goschool/internal/domain/derr"
	"goschool/internal/domain/shared"
	"goschool/internal/domain/subject"
)

type fakeRepo struct {
	byID   map[int]*Course
	inUse  map[int]bool
	nextID int
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{byID: map[int]*Course{}, inUse: map[int]bool{}, nextID: 1}
}

func (f *fakeRepo) Create(_ context.Context, c *Course) error {
	c.ID = f.nextID
	f.nextID++
	cp := *c
	f.byID[c.ID] = &cp
	return nil
}

func (f *fakeRepo) Update(_ context.Context, c *Course) error {
	if _, ok := f.byID[c.ID]; !ok {
		return ErrNotFound
	}
	cp := *c
	f.byID[c.ID] = &cp
	return nil
}

func (f *fakeRepo) Delete(_ context.Context, id int) error {
	if _, ok := f.byID[id]; !ok {
		return ErrNotFound
	}
	delete(f.byID, id)
	return nil
}

func (f *fakeRepo) GetByID(_ context.Context, id int) (*View, error) {
	c, ok := f.byID[id]
	if !ok {
		return nil, ErrNotFound
	}
	return &View{Course: *c, SubjectName: "Math"}, nil
}

func (f *fakeRepo) NameExists(_ context.Context, name string, excludeID int) (bool, error) {
	for _, c := range f.byID {
		if c.ID != excludeID && strings.EqualFold(c.Name, name) {
			return true, nil
		}
	}
	return false, nil
}

func (f *fakeRepo) InUse(_ context.Context, id int) (bool, error) { return f.inUse[id], nil }

func (f *fakeRepo) List(_ context.Context, _ Filter, _ shared.PageRequest) ([]View, int, error) {
	return nil, 0, nil
}

type fakeSubjects map[int]string

func (f fakeSubjects) GetByID(_ context.Context, id int) (*subject.Subject, error) {
	name, ok := f[id]
	if !ok {
		return nil, subject.ErrNotFound
	}
	return &subject.Subject{ID: id, Name: name}, nil
}

type fakeTx struct{}

func (fakeTx) WithinTx(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) }

var (
	ctx     = context.Background()
	admin   = shared.Actor{UserID: 1, Role: shared.RoleAdmin}
	student = shared.Actor{UserID: 2, Role: shared.RoleStudent}
)

func newService() (*Service, *fakeRepo) {
	repo := newFakeRepo()
	return NewService(repo, fakeSubjects{1: "Math"}, fakeTx{}), repo
}

func statusPtr(s Status) *Status { return &s }

func TestCreate(t *testing.T) {
	svc, _ := newService()

	if _, err := svc.Create(ctx, student, CreateInput{SubjectID: 1, Name: "Math 7"}); !derr.IsKind(err, derr.KindForbidden) {
		t.Fatalf("non-admin: got %v", err)
	}
	if _, err := svc.Create(ctx, admin, CreateInput{SubjectID: 0, Name: " ", Status: statusPtr("archived")}); !derr.IsKind(err, derr.KindInvalid) {
		t.Fatalf("invalid input: got %v", err)
	}
	if _, err := svc.Create(ctx, admin, CreateInput{SubjectID: 42, Name: "Math 7"}); !derr.Is(err, subject.CodeSubjectNotFound) {
		t.Fatalf("unknown subject: got %v", err)
	}

	v, err := svc.Create(ctx, admin, CreateInput{SubjectID: 1, Name: " Math 7 "})
	if err != nil {
		t.Fatal(err)
	}
	if v.Name != "Math 7" || v.Status != StatusActive || v.SubjectName != "Math" {
		t.Fatalf("unexpected course %+v", v)
	}
	if _, err := svc.Create(ctx, admin, CreateInput{SubjectID: 1, Name: "math 7"}); !derr.Is(err, CodeCourseNameAlreadyExists) {
		t.Fatalf("duplicate name: got %v", err)
	}
}

func TestUpdate(t *testing.T) {
	svc, _ := newService()
	v, _ := svc.Create(ctx, admin, CreateInput{SubjectID: 1, Name: "Math 7"})
	_, _ = svc.Create(ctx, admin, CreateInput{SubjectID: 1, Name: "Math 8"})

	if _, err := svc.Update(ctx, admin, v.ID, UpdateInput{}); !derr.Is(err, derr.CodeNothingToUpdate) {
		t.Fatalf("empty update: got %v", err)
	}
	name := "MATH 8"
	if _, err := svc.Update(ctx, admin, v.ID, UpdateInput{Name: &name}); !derr.Is(err, CodeCourseNameAlreadyExists) {
		t.Fatalf("rename onto existing: got %v", err)
	}
	if _, err := svc.Update(ctx, admin, v.ID, UpdateInput{Status: statusPtr("archived")}); !derr.IsKind(err, derr.KindInvalid) {
		t.Fatalf("bad status: got %v", err)
	}
	got, err := svc.Update(ctx, admin, v.ID, UpdateInput{Status: statusPtr(StatusInactive)})
	if err != nil || got.Status != StatusInactive || got.Name != "Math 7" || got.IsActive() {
		t.Fatalf("deactivate: %+v %v", got, err)
	}
}

func TestDelete(t *testing.T) {
	svc, repo := newService()
	v, _ := svc.Create(ctx, admin, CreateInput{SubjectID: 1, Name: "Math 7"})

	repo.inUse[v.ID] = true
	if err := svc.Delete(ctx, admin, v.ID); !derr.Is(err, CodeCourseInUse) {
		t.Fatalf("in use: got %v", err)
	}
	repo.inUse[v.ID] = false
	if err := svc.Delete(ctx, admin, v.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Get(ctx, student, v.ID); !derr.Is(err, CodeCourseNotFound) {
		t.Fatalf("deleted course: got %v", err)
	}
}

func TestListRejectsBadStatus(t *testing.T) {
	svc, _ := newService()
	_, err := svc.List(ctx, student, Filter{Status: statusPtr("archived")}, shared.PageRequest{})
	e, ok := derr.As(err)
	if !ok || e.Kind != derr.KindInvalid || len(e.Fields) != 1 || e.Fields[0].Field != "status" {
		t.Fatalf("expected invalid status field error, got %v", err)
	}
}
