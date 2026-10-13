package subject

import (
	"context"
	"strings"
	"testing"

	"goschool/internal/domain/derr"
	"goschool/internal/domain/shared"
)

type fakeRepo struct {
	byID   map[int]*Subject
	inUse  map[int]bool
	nextID int
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{byID: map[int]*Subject{}, inUse: map[int]bool{}, nextID: 1}
}

func (f *fakeRepo) Create(_ context.Context, s *Subject) error {
	s.ID = f.nextID
	f.nextID++
	c := *s
	f.byID[s.ID] = &c
	return nil
}

func (f *fakeRepo) Update(_ context.Context, s *Subject) error {
	if _, ok := f.byID[s.ID]; !ok {
		return ErrNotFound
	}
	c := *s
	f.byID[s.ID] = &c
	return nil
}

func (f *fakeRepo) Delete(_ context.Context, id int) error {
	if _, ok := f.byID[id]; !ok {
		return ErrNotFound
	}
	delete(f.byID, id)
	return nil
}

func (f *fakeRepo) GetByID(_ context.Context, id int) (*Subject, error) {
	s, ok := f.byID[id]
	if !ok {
		return nil, ErrNotFound
	}
	c := *s
	return &c, nil
}

func (f *fakeRepo) NameExists(_ context.Context, name string, excludeID int) (bool, error) {
	for _, s := range f.byID {
		if s.ID != excludeID && strings.EqualFold(s.Name, name) {
			return true, nil
		}
	}
	return false, nil
}

func (f *fakeRepo) InUse(_ context.Context, id int) (bool, error) { return f.inUse[id], nil }

func (f *fakeRepo) List(_ context.Context, _ Filter, _ shared.PageRequest) ([]Subject, int, error) {
	var out []Subject
	for _, s := range f.byID {
		out = append(out, *s)
	}
	return out, len(out), nil
}

type fakeTx struct{}

func (fakeTx) WithinTx(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) }

var (
	ctx     = context.Background()
	admin   = shared.Actor{UserID: 1, Role: shared.RoleAdmin}
	teacher = shared.Actor{UserID: 2, Role: shared.RoleTeacher}
)

func TestCreate(t *testing.T) {
	svc := NewService(newFakeRepo(), fakeTx{})

	if _, err := svc.Create(ctx, teacher, CreateInput{Name: "Math"}); !derr.IsKind(err, derr.KindForbidden) {
		t.Fatalf("non-admin: got %v", err)
	}
	if _, err := svc.Create(ctx, admin, CreateInput{Name: "   "}); !derr.IsKind(err, derr.KindInvalid) {
		t.Fatalf("blank name: got %v", err)
	}
	if _, err := svc.Create(ctx, admin, CreateInput{Name: strings.Repeat("x", 101)}); !derr.IsKind(err, derr.KindInvalid) {
		t.Fatalf("long name: got %v", err)
	}

	s, err := svc.Create(ctx, admin, CreateInput{Name: "  Math  "})
	if err != nil || s.Name != "Math" || s.ID == 0 {
		t.Fatalf("create: %+v %v", s, err)
	}
	if _, err := svc.Create(ctx, admin, CreateInput{Name: "MATH"}); !derr.Is(err, CodeSubjectNameAlreadyExists) {
		t.Fatalf("duplicate (case-insensitive): got %v", err)
	}
}

func TestUpdate(t *testing.T) {
	svc := NewService(newFakeRepo(), fakeTx{})
	math, _ := svc.Create(ctx, admin, CreateInput{Name: "Math"})
	_, _ = svc.Create(ctx, admin, CreateInput{Name: "Physics"})

	if _, err := svc.Update(ctx, admin, math.ID, UpdateInput{Name: "physics"}); !derr.Is(err, CodeSubjectNameAlreadyExists) {
		t.Fatalf("rename onto existing: got %v", err)
	}
	// Changing only the case of its own name is allowed.
	got, err := svc.Update(ctx, admin, math.ID, UpdateInput{Name: "MATH"})
	if err != nil || got.Name != "MATH" {
		t.Fatalf("own rename: %+v %v", got, err)
	}
	if _, err := svc.Update(ctx, admin, 99, UpdateInput{Name: "X"}); !derr.Is(err, CodeSubjectNotFound) {
		t.Fatalf("missing: got %v", err)
	}
}

func TestDelete(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo, fakeTx{})
	math, _ := svc.Create(ctx, admin, CreateInput{Name: "Math"})

	repo.inUse[math.ID] = true
	if err := svc.Delete(ctx, admin, math.ID); !derr.Is(err, CodeSubjectInUse) {
		t.Fatalf("in use: got %v", err)
	}
	repo.inUse[math.ID] = false
	if err := svc.Delete(ctx, teacher, math.ID); !derr.IsKind(err, derr.KindForbidden) {
		t.Fatalf("non-admin: got %v", err)
	}
	if err := svc.Delete(ctx, admin, math.ID); err != nil {
		t.Fatal(err)
	}
	if err := svc.Delete(ctx, admin, math.ID); !derr.Is(err, CodeSubjectNotFound) {
		t.Fatalf("already deleted: got %v", err)
	}
}
