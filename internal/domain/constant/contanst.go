package constant

const (
	DefaultPage     = 1
	DefaultPageSize = 20

	RoleAdmin   = "admin"
	RoleTeacher = "teacher"
	RoleStudent = "student"

	GenderMale   = "male"
	GenderFemale = "female"
	GenderOther  = "other"

	WorkingStatusActive   = "active"
	WorkingStatusInactive = "inactive"
	WorkingStatusOnLeave  = "on_leave"

	EnvDevelopment = "development"
	EnvProduction  = "production"
	EnvTest        = "test"

	TokenTypeRefresh           = "refresh_token"
	TokenTypePasswordUpdate    = "password_update_token"
	TokenTypeEmailVerification = "email_verification_token"

	CookieAccessToken  = "access_token"
	CookieRefreshToken = "refresh_token"

	SubjectStatusActive   = "active"
	SubjectStatusInactive = "inactive"
)
