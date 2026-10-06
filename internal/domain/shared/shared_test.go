package shared

import (
	"testing"
	"time"
)

func TestAcademicYearContains(t *testing.T) {
	y := AcademicYear(2025)
	cases := []struct {
		date time.Time
		want bool
	}{
		{NewDate(2025, time.July, 31), false},
		{NewDate(2025, time.August, 1), true},
		{time.Date(2025, time.December, 31, 23, 59, 0, 0, time.UTC), true},
		{NewDate(2026, time.July, 31), true},
		{NewDate(2026, time.August, 1), false},
	}
	for _, c := range cases {
		if got := y.Contains(c.date); got != c.want {
			t.Errorf("Contains(%s) = %v, want %v", c.date.Format(time.DateOnly), got, c.want)
		}
	}
}

func TestPageRequestNormalize(t *testing.T) {
	cases := []struct {
		in, want PageRequest
	}{
		{PageRequest{0, 0}, PageRequest{1, DefaultPageSize}},
		{PageRequest{3, 500}, PageRequest{3, MaxPageSize}},
		{PageRequest{2, 10}, PageRequest{2, 10}},
	}
	for _, c := range cases {
		if got := c.in.Normalize(); got != c.want {
			t.Errorf("Normalize(%v) = %v, want %v", c.in, got, c.want)
		}
	}
	if off := (PageRequest{3, 10}).Offset(); off != 20 {
		t.Errorf("Offset = %d, want 20", off)
	}
}

func TestValidate(t *testing.T) {
	cases := []struct {
		name    string
		err     error
		wantErr bool
	}{
		{"role ok", RoleTeacher.Validate(), false},
		{"role bad", Role("wizard").Validate(), true},
		{"year ok", AcademicYear(2025).Validate(), false},
		{"year bad", AcademicYear(1999).Validate(), true},
		{"semester ok", Semester(2).Validate(), false},
		{"semester bad", Semester(3).Validate(), true},
	}
	for _, c := range cases {
		if (c.err != nil) != c.wantErr {
			t.Errorf("%s: err = %v, wantErr %v", c.name, c.err, c.wantErr)
		}
	}
	if msg := Role("x").Validate().Error(); msg != "must be one of: student, teacher, admin" {
		t.Errorf("message must describe allowed values without a field name, got %q", msg)
	}
}
