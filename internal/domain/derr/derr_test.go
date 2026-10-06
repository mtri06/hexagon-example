package derr

import (
	"errors"
	"fmt"
	"testing"
)

func TestFieldErrorsErr(t *testing.T) {
	if err := (FieldErrors{}).Err(); err != nil {
		t.Fatalf("empty FieldErrors should yield nil, got %v", err)
	}

	f := FieldErrors{}
	f.Add("name", "is required")
	f.Add("email", "is invalid")
	f.Add("name", "ignored second message")

	e, ok := As(f.Err())
	if !ok {
		t.Fatal("expected *Error")
	}
	if e.Kind != KindInvalid || e.Code != CodeValidationFailed {
		t.Fatalf("unexpected kind/code: %v %v", e.Kind, e.Code)
	}
	want := []FieldError{{"email", "is invalid"}, {"name", "is required"}}
	if len(e.Fields) != len(want) {
		t.Fatalf("got %v, want %v", e.Fields, want)
	}
	for i := range want {
		if e.Fields[i] != want[i] {
			t.Fatalf("got %v, want %v", e.Fields, want)
		}
	}
}

func TestWrapAndAs(t *testing.T) {
	base := NotFound("user_not_found", "User not found")
	cause := errors.New("sql: no rows")
	wrapped := fmt.Errorf("repo: %w", base.Wrap(cause))

	if !Is(wrapped, "user_not_found") {
		t.Fatal("Is should find code through wrapping")
	}
	if !IsKind(wrapped, KindNotFound) {
		t.Fatal("IsKind should find kind through wrapping")
	}
	if !errors.Is(wrapped, cause) {
		t.Fatal("cause should be reachable via errors.Is")
	}
	if base.Err != nil {
		t.Fatal("Wrap must not mutate the receiver")
	}
}
