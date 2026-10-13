package identity

import "context"

// EnsureUnique checks that username (when non-empty) and email (when non-nil)
// are not used by another user. Aggregates that create users (student,
// teacher) call it before inserting.
func EnsureUnique(ctx context.Context, users UserRepository, username string, email *string, excludeUserID int) error {
	if username != "" {
		exists, err := users.UsernameExists(ctx, username)
		if err != nil {
			return err
		}
		if exists {
			return ErrUsernameTaken
		}
	}
	if email != nil {
		exists, err := users.EmailExists(ctx, *email, excludeUserID)
		if err != nil {
			return err
		}
		if exists {
			return ErrEmailTaken
		}
	}
	return nil
}
