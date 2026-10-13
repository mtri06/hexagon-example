package subject

import (
	"context"

	"goschool/internal/domain/port"
	"goschool/internal/domain/shared"
)

// Service implements the subject use cases. Any authenticated actor can read;
// only admins can write.
type Service struct {
	repo Repository
	tx   port.TxManager
}

func NewService(repo Repository, tx port.TxManager) *Service {
	return &Service{repo: repo, tx: tx}
}

func (s *Service) List(ctx context.Context, _ shared.Actor, f Filter, p shared.PageRequest) (shared.Page[Subject], error) {
	p = p.Normalize()
	items, total, err := s.repo.List(ctx, f, p)
	if err != nil {
		return shared.Page[Subject]{}, err
	}
	return shared.NewPage(items, p, total), nil
}

func (s *Service) Get(ctx context.Context, _ shared.Actor, id int) (*Subject, error) {
	return s.repo.GetByID(ctx, id)
}

// Create adds a subject. Names are unique, compared case-insensitively.
func (s *Service) Create(ctx context.Context, actor shared.Actor, in CreateInput) (*Subject, error) {
	if err := actor.RequireAdmin(); err != nil {
		return nil, err
	}
	sub, err := New(in)
	if err != nil {
		return nil, err
	}
	if err := s.ensureNameFree(ctx, sub.Name, 0); err != nil {
		return nil, err
	}
	if err := s.repo.Create(ctx, sub); err != nil {
		return nil, err
	}
	return sub, nil
}

// Update renames a subject.
func (s *Service) Update(ctx context.Context, actor shared.Actor, id int, in UpdateInput) (*Subject, error) {
	if err := actor.RequireAdmin(); err != nil {
		return nil, err
	}
	var sub *Subject
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		var err error
		if sub, err = s.repo.GetByID(ctx, id); err != nil {
			return err
		}
		if err := sub.ApplyUpdate(in); err != nil {
			return err
		}
		if err := s.ensureNameFree(ctx, sub.Name, sub.ID); err != nil {
			return err
		}
		return s.repo.Update(ctx, sub)
	})
	if err != nil {
		return nil, err
	}
	return sub, nil
}

// Delete removes a subject that no course or teacher qualification uses.
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
