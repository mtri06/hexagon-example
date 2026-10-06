package shared

import (
	"errors"
	"time"
)

// Dates (date_of_birth, start_date, ...) are represented as time.Time at
// midnight UTC. Use DateOf to normalize any instant to its calendar date.

func DateOf(t time.Time) time.Time {
	y, m, d := t.UTC().Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func NewDate(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

// AcademicYear is the starting year: 2025 means the 2025–2026 school year,
// which runs from Aug 1, 2025 to Jul 31, 2026 inclusive.
type AcademicYear int

const (
	minAcademicYear = 2000
	maxAcademicYear = 2100
)

var errInvalidAcademicYear = errors.New("must be between 2000 and 2100")

// Validate returns an error whose message describes the allowed range.
func (y AcademicYear) Validate() error {
	if y < minAcademicYear || y > maxAcademicYear {
		return errInvalidAcademicYear
	}
	return nil
}

func (y AcademicYear) Start() time.Time { return NewDate(int(y), time.August, 1) }
func (y AcademicYear) End() time.Time   { return NewDate(int(y)+1, time.July, 31) }

// Contains reports whether the calendar date of t falls within the academic year.
func (y AcademicYear) Contains(t time.Time) bool {
	d := DateOf(t)
	return !d.Before(y.Start()) && !d.After(y.End())
}

type Semester int

var errInvalidSemester = errors.New("must be 1 or 2")

// Validate returns an error whose message describes the allowed values.
func (s Semester) Validate() error {
	if s != 1 && s != 2 {
		return errInvalidSemester
	}
	return nil
}
