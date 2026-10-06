package respond

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"goschool/internal/domain/derr"
)

func TestErrorMapping(t *testing.T) {
	cases := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{"invalid", derr.Invalid("validation_failed", "bad", derr.FieldError{Field: "name", Message: "required"}), 400, "validation_failed"},
		{"unauthenticated", derr.Unauthenticated("invalid_credentials", "x"), 401, "invalid_credentials"},
		{"forbidden", derr.Forbidden("forbidden", "x"), 403, "forbidden"},
		{"not found wrapped", fmt.Errorf("ctx: %w", derr.NotFound("user_not_found", "x")), 404, "user_not_found"},
		{"conflict", derr.Conflict("username_already_exists", "x"), 409, "username_already_exists"},
		{"rule", derr.RuleViolation("cannot_deactivate_self", "x"), 422, "cannot_deactivate_self"},
		{"api error", NewAPIError(400, CodeInvalidBody, "x"), 400, CodeInvalidBody},
		{"unknown", errors.New("boom"), 500, CodeInternal},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			Error(rec, httptest.NewRequest(http.MethodGet, "/", nil), c.err)
			if rec.Code != c.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, c.wantStatus)
			}
			var body APIError
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if body.Code != c.wantCode {
				t.Fatalf("code = %q, want %q", body.Code, c.wantCode)
			}
		})
	}
}

func TestErrorFields(t *testing.T) {
	rec := httptest.NewRecorder()
	f := derr.FieldErrors{}
	f.Add("email", "must be a valid email address")
	Error(rec, httptest.NewRequest(http.MethodGet, "/", nil), f.Err())

	var body APIError
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if len(body.Fields) != 1 || body.Fields[0].Field != "email" {
		t.Fatalf("unexpected fields: %+v", body.Fields)
	}
}
