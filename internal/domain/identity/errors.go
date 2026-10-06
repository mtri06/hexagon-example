package identity

import "goschool/internal/domain/derr"

const (
	CodeUserNotFound             = "user_not_found"
	CodeUsernameAlreadyExists    = "username_already_exists"
	CodeEmailAlreadyExists       = "email_already_exists"
	CodeInvalidCredentials       = "invalid_credentials"
	CodeUserInactive             = "user_inactive"
	CodeInvalidRefreshToken      = "invalid_refresh_token"
	CodeRefreshTokenExpired      = "refresh_token_expired"
	CodeRefreshTokenReused       = "refresh_token_reused"
	CodeTokenNotFound            = "token_not_found"
	CodeCurrentPasswordIncorrect = "current_password_incorrect"
	CodeCannotDeactivateSelf     = "cannot_deactivate_self"
	CodeNothingToUpdate          = "nothing_to_update"
)

var (
	ErrUserNotFound        = derr.NotFound(CodeUserNotFound, "User not found")
	ErrUsernameTaken       = derr.Conflict(CodeUsernameAlreadyExists, "Username is already taken")
	ErrEmailTaken          = derr.Conflict(CodeEmailAlreadyExists, "Email is already in use")
	ErrInvalidCredentials  = derr.Unauthenticated(CodeInvalidCredentials, "Invalid username or password")
	ErrUserInactive        = derr.Unauthenticated(CodeUserInactive, "User account is deactivated")
	ErrInvalidRefreshToken = derr.Unauthenticated(CodeInvalidRefreshToken, "Refresh token is invalid")
	ErrRefreshTokenExpired = derr.Unauthenticated(CodeRefreshTokenExpired, "Refresh token has expired")
	ErrRefreshTokenReused  = derr.Unauthenticated(CodeRefreshTokenReused, "Refresh token was already used; all sessions have been revoked")
	ErrTokenNotFound       = derr.NotFound(CodeTokenNotFound, "Token not found")
	ErrCannotDeactivate    = derr.RuleViolation(CodeCannotDeactivateSelf, "You cannot deactivate your own account")
	ErrNothingToUpdate     = derr.Invalid(CodeNothingToUpdate, "At least one field must be provided")
)
