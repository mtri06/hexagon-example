// Package dto holds request/response bodies that mirror api/openapi.yaml and
// their mapping to and from domain types.
package dto

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"goschool/internal/domain/shared"
)

// Date is a calendar date serialized as "YYYY-MM-DD".
type Date struct{ time.Time }

func NewDate(t time.Time) Date { return Date{shared.DateOf(t)} }

func (d Date) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.Format(time.DateOnly))
}

func (d *Date) UnmarshalJSON(b []byte) error {
	if bytes.Equal(b, []byte("null")) {
		return nil
	}
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return dateError{string(b)}
	}
	t, err := time.Parse(time.DateOnly, s)
	if err != nil {
		return dateError{s}
	}
	d.Time = t
	return nil
}

// DatePtr returns the underlying time, or nil.
func (d *Date) TimePtr() *time.Time {
	if d == nil {
		return nil
	}
	t := d.Time
	return &t
}

type dateError struct{ value string }

func (e dateError) Error() string { return fmt.Sprintf("invalid date %q", e.value) }

// UserMessage is picked up by httpx.DecodeBody for the 400 message.
func (e dateError) UserMessage() string {
	return fmt.Sprintf("%s is not a valid date, expected YYYY-MM-DD", e.value)
}

// Nullable distinguishes an absent JSON field (Set=false) from an explicit
// null (Set=true, Value=nil) for PATCH requests.
type Nullable[T any] struct {
	Set   bool
	Value *T
}

func (n *Nullable[T]) UnmarshalJSON(b []byte) error {
	n.Set = true
	if bytes.Equal(b, []byte("null")) {
		n.Value = nil
		return nil
	}
	var v T
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	n.Value = &v
	return nil
}

func (n Nullable[T]) ToDomain() shared.Nullable[T] {
	return shared.Nullable[T]{Set: n.Set, Value: n.Value}
}

type PageMeta struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
	Total    int `json:"total"`
}

type PageResponse[T any] struct {
	Data []T      `json:"data"`
	Meta PageMeta `json:"meta"`
}

// NewPageResponse maps a domain page with fn.
func NewPageResponse[D, T any](p shared.Page[D], fn func(*D) T) PageResponse[T] {
	data := make([]T, len(p.Items))
	for i := range p.Items {
		data[i] = fn(&p.Items[i])
	}
	return PageResponse[T]{
		Data: data,
		Meta: PageMeta{Page: p.Page, PageSize: p.PageSize, Total: p.Total},
	}
}
