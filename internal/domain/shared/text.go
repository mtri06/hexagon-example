package shared

import (
	"errors"
	"fmt"
	"unicode/utf8"
)

var errTextRequired = errors.New("is required")

// ValidateText checks a required free-text value (already trimmed) of at most
// maxLen characters. Like the other Validate functions, the message omits the
// field name so it can be used with derr.FieldErrors.AddErr.
func ValidateText(s string, maxLen int) error {
	switch n := utf8.RuneCountInString(s); {
	case n == 0:
		return errTextRequired
	case n > maxLen:
		return fmt.Errorf("must be at most %d characters", maxLen)
	}
	return nil
}
