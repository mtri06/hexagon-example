package course

import (
	"context"

	"goschool/internal/domain/derr"
	"goschool/internal/domain/port"
	"goschool/internal/domain/shared"
)

// Service implements the course use cases. Any authenticated actor can read;
// only admins can write.
type Service struct {
	repo     Repository
	subjects SubjectReader
	tx       port.TxManager
}

func NewService(repo Repository, subjects SubjectReader, tx port.TxManager) *Service {
	return &Service{repo: repo, subjects: subjects, tx: tx}
}

func (s *Service) List(ctx context.Context, _ shared.Actor, f Filter, p shared.PageRequest) (shared.Page[View], error) {
	if f.Status != nil {
		if err := f.Status.Validate(); err != nil {
			return shared.Page[View]{}, derr.InvalidField("status", err)
		}
	}
	p = p.Normalize()
	items, total, err := s.repo.List(ctx, f, p)
	if err != nil {
		return shared.Page[View]{}, err
	}
	return shared.NewPage(items, p, total), nil
}

func (s *Service) Get(ctx context.Context, _ shared.Actor, id int) (*View, error) {
	return s.repo.GetByID(ctx, id)
}

// Create adds a course to an existing subject. Names are unique, compared
// case-insensitively.
func (s *Service) Create(ctx context.Context, actor shared.Actor, in CreateInput) (*View, error) {
	if err := actor.RequireAdmin(); err != nil {
		return nil, err
	}
	c, err := New(in)
	if err != nil {
		return nil, err
	}
	sub, err := s.subjects.GetByID(ctx, c.SubjectID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureNameFree(ctx, c.Name, 0); err != nil {
		return nil, err
	}
	if err := s.repo.Create(ctx, c); err != nil {
		return nil, err
	}
	return &View{Course: *c, SubjectName: sub.Name}, nil
}

// Update renames and/or (de)activates a course. The subject cannot change.
func (s *Service) Update(ctx context.Context, actor shared.Actor, id int, in UpdateInput) (*View, error) {
	if err := actor.RequireAdmin(); err != nil {
		return nil, err
	}
	var v *View
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		var err error
		if v, err = s.repo.GetByID(ctx, id); err != nil {
			return err
		}
		if err := v.ApplyUpdate(in); err != nil {
			return err
		}
		if in.Name != nil {
			if err := s.ensureNameFree(ctx, v.Name, v.ID); err != nil {
				return err
			}
		}
		return s.repo.Update(ctx, &v.Course)
	})
	if err != nil {
		return nil, err
	}
	return v, nil
}

// Delete removes a course nothing references. Courses with history should be
// deactivated instead.
func (s *Service) Delete(ctx context.Context, actor shared.Actor, id int) error {
	if err := actor.RequireAdmin(); err != nil {
		return err
	}
	return s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if _, err := s.repo.GetByID(ctx, id); err != nil {
			return err
		}
		inUse, err := s.repo.InUse(ctx, id)
		if err != nil {
			return err
		}
		if inUse {
			return ErrInUse
		}
		return s.repo.Delete(ctx, id)
	})
}

func (s *Service) ensureNameFree(ctx context.Context, name string, excludeID int) error {
	exists, err := s.repo.NameExists(ctx, name, excludeID)
	if err != nil {
		return err
	}
	if exists {
		return ErrNameTaken
	}
	return nil
}
