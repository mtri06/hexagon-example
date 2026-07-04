package env

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
)

type EnvConfig struct {
	Environment string `validate:"required,oneof=dev test prod"`
	ListenIP    string `validate:"required,ip"`
	ListenPort  int    `validate:"required,min=1"`

	PgHost        string `validate:"required"`
	PgPort        int    `validate:"required,min=1"`
	PgUser        string `validate:"required"`
	PgPassword    string `validate:"required"`
	PgDBName      string `validate:"required"`
	PgSSLMode     string `validate:"required,oneof=disable allow prefer require verify-ca verify-full"`
	PgConnTimeout time.Duration

	AllowedOrigins []string

	JWTSecret         string        `validate:"required"`
	JWTAccessExpires  time.Duration `validate:"required"`
	JWTRefreshExpires time.Duration `validate:"required"`

	AdminUsername string `validate:"required"`
	AdminPassword string `validate:"required"`
}

var Env EnvConfig

var validate = validator.New(validator.WithRequiredStructEnabled())

func Init(envFiles ...string) {
	if err := godotenv.Load(envFiles...); err != nil {
		log.Warn().Msgf("No %v file found, relying on environment variables", strings.Join(envFiles, ", "))
	}

	Env = EnvConfig{
		Environment: envOrDefault("ENVIRONMENT", "dev"),
		ListenIP:    envOrDefault("LISTEN_IP", "0.0.0.0"),
		ListenPort:  envIntOrDefault("LISTEN_PORT", 8080),

		PgHost:        envOrPanic("PG_HOST"),
		PgPort:        envIntOrPanic("PG_PORT"),
		PgUser:        envOrPanic("PG_USER"),
		PgPassword:    envOrPanic("PG_PASSWORD"),
		PgDBName:      envOrPanic("PG_DB_NAME"),
		PgSSLMode:     envOrPanic("PG_SSL_MODE"),
		PgConnTimeout: envDurationOrDefault("PG_CONN_TIMEOUT", 10*time.Second),

		AllowedOrigins: strings.Split(envOrPanic("CORS_ALLOWED_ORIGINS"), ","),

		JWTSecret:         envOrPanic("JWT_SECRET"),
		JWTAccessExpires:  envDurationOrPanic("JWT_ACCESS_EXPIRES"),
		JWTRefreshExpires: envDurationOrPanic("JWT_REFRESH_EXPIRES"),

		AdminUsername: envOrPanic("ADMIN_USERNAME"),
		AdminPassword: envOrPanic("ADMIN_PASSWORD"),
	}

	if err := validate.Struct(Env); err != nil {
		log.Fatal().Err(err).Msg("invalid environment configuration")
	}
}

func envOrPanic(key string) string {
	val := os.Getenv(key)
	if val == "" {
		log.Panic().Msgf("%v env is not set", key)
	}
	return val
}

func envOrDefault(key, defaultVal string) string {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	return val
}

func envIntOrPanic(key string) int {
	val := os.Getenv(key)
	if val == "" {
		log.Panic().Msgf("%v env is not set", key)
	}
	intVal, err := strconv.Atoi(val)
	if err != nil {
		log.Panic().Msgf("%v env is not a valid integer: %v", key, err)
	}
	return intVal
}

func envIntOrDefault(key string, defaultVal int) int {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	intVal, err := strconv.Atoi(val)
	if err != nil {
		log.Panic().Msgf("%v env is not a valid integer: %v", key, err)
	}
	return intVal
}

func envDurationOrPanic(key string) time.Duration {
	val := os.Getenv(key)
	if val == "" {
		log.Panic().Msgf("%v env is not set", key)
	}
	duration, err := time.ParseDuration(val)
	if err != nil {
		log.Panic().Msgf("%v env is not a valid duration: %v", key, err)
	}
	return duration
}

func envDurationOrDefault(key string, defaultVal time.Duration) time.Duration {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	duration, err := time.ParseDuration(val)
	if err != nil {
		log.Panic().Msgf("%v env is not a valid duration: %v", key, err)
	}
	return duration
}
