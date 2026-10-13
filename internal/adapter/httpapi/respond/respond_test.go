package respond

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
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

func TestErrorContextErrors(t *testing.T) {
	req := func() *http.Request { return httptest.NewRequest(http.MethodGet, "/", nil) }
	canceledReq := func() *http.Request {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		return req().WithContext(ctx)
	}

	canceledCases := []struct {
		name string
		r    *http.Request
		err  error
	}{
		{"wrapped context.Canceled", req(), fmt.Errorf("query: %w", context.Canceled)},
		{"request canceled, unrelated error", canceledReq(), errors.New("write: broken pipe")},
	}
	for _, c := range canceledCases {
		rec := httptest.NewRecorder()
		Error(rec, c.r, c.err)
		if rec.Code != StatusClientClosedRequest || rec.Body.Len() != 0 {
			t.Errorf("%s: status=%d body=%q", c.name, rec.Code, rec.Body.String())
		}
	}

	timeoutCases := []struct {
		name string
		err  error
	}{
		{"wrapped context.DeadlineExceeded", fmt.Errorf("repo: %w", context.DeadlineExceeded)},
		// What pgdriver returns when a context deadline hits mid-query.
		{"socket i/o timeout", &net.OpError{Op: "read", Net: "tcp", Err: os.ErrDeadlineExceeded}},
	}
	for _, c := range timeoutCases {
		rec := httptest.NewRecorder()
		Error(rec, req(), c.err)
		var body APIError
		_ = json.Unmarshal(rec.Body.Bytes(), &body)
		if rec.Code != http.StatusServiceUnavailable || body.Code != CodeTimeout {
			t.Errorf("%s: status=%d body=%+v", c.name, rec.Code, body)
		}
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
