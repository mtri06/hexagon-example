package shared

const (
	DefaultPageSize = 20
	MaxPageSize     = 100
)

type PageRequest struct {
	Page     int
	PageSize int
}

// Normalize clamps the request to valid bounds.
func (p PageRequest) Normalize() PageRequest {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.PageSize < 1 {
		p.PageSize = DefaultPageSize
	}
	if p.PageSize > MaxPageSize {
		p.PageSize = MaxPageSize
	}
	return p
}

func (p PageRequest) Offset() int { return (p.Page - 1) * p.PageSize }

type Page[T any] struct {
	Items    []T
	Page     int
	PageSize int
	Total    int
}

func NewPage[T any](items []T, req PageRequest, total int) Page[T] {
	if items == nil {
		items = []T{}
	}
	return Page[T]{Items: items, Page: req.Page, PageSize: req.PageSize, Total: total}
}
