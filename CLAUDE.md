# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

`goschool` is a Go (1.25) REST API backend for school management: users (student/teacher/admin), subjects, courses, classes, teaching assignments, timetables and scores. Stack: chi router, **bun** over pgdriver (PostgreSQL), goose migrations, zerolog, go-playground/validator, golang-jwt, testcontainers-go for integration tests.

- `api/openapi.yaml` is the HTTP contract. Keep handlers and DTOs in sync with it.
- `docs/architecture.md` describes the hexagonal layout, the error model, the business rules and the build status (which phases are done).

## Commands

```bash
cp .env.example .env                  # first time
docker compose up -d --wait           # local Postgres 16

go build ./...
go vet ./...
go test ./...                         # unit + integration (integration needs Docker running)
go test -short ./...                  # unit tests only (pgtest skips)
go test ./internal/domain/identity -run TestRefresh

go run ./cmd/goschool migrate         # goose up (embedded SQL)
go run ./cmd/goschool rollback        # goose down one step
go run ./cmd/goschool seed-admin      # create ADMIN_USERNAME/ADMIN_PASSWORD admin if missing
go run ./cmd/goschool serve           # HTTP API on LISTEN_IP:LISTEN_PORT, routes under /api/v1
BUNDEBUG=2 go run ./cmd/goschool serve  # log every SQL query
```

There is no Makefile or linter config.

## Configuration

`internal/config` loads `.env` (if present) and the environment into one `Config`. Each command validates only what it needs: `ValidateDB` for every command, `ValidateHTTP` and `ValidateJWT` for `serve`, `ValidateAdmin` for `seed-admin`. The variables are `ENVIRONMENT` (dev|test|prod), `PG_*`, `LISTEN_IP`, `LISTEN_PORT`, `CORS_ALLOWED_ORIGINS`, `JWT_SECRET` (≥ 32 chars), `JWT_ACCESS_EXPIRES`, `JWT_REFRESH_EXPIRES` and `ADMIN_*`. See `.env.example`.

## Architecture (hexagonal)

Dependencies point inward toward `internal/domain`. The domain must not import bun, net/http, chi, jwt or bcrypt.

- **`internal/domain/derr`**: the only error type the domain returns. `*derr.Error{Kind, Code, Msg, Fields}`. Use the constructors `derr.Invalid/NotFound/Conflict/RuleViolation/Forbidden/Unauthenticated`, `derr.FieldErrors` to collect several field errors (`f.AddErr("role", role.Validate())`), `derr.InvalidField("status", err)` for a single one, `derr.ErrNothingToUpdate` for an empty PATCH, and `derr.Is(err, code)` / `derr.IsKind` to check them. Each kind maps to exactly one HTTP status: 400/404/409/422/403/401.
- **`internal/domain/shared`**: `Actor`, `Role`, `PageRequest`/`Page[T]`, `Nullable[T]` (PATCH tri-state), `AcademicYear` (Aug 1 – Jul 31, with `Contains`), `Semester`, and the date helpers (`DateOf`, `NewDate`). Dates are `time.Time` at midnight UTC.
- Value types (enums, ranges) expose `Validate() error`, not `Valid() bool`. The error message describes the allowed values and leaves out the field name, e.g. `"must be one of: male, female, other"`, so `FieldErrors.AddErr` can attach the field. `shared.ValidateText(s, maxLen)` does the same for required free text. Never return a bare `Validate()` error from a service: wrap it with `derr.InvalidField`, otherwise it renders as a 500.
- **`internal/domain/port`**: `TxManager`, `PasswordHasher`, `TokenIssuer`, `SecretGenerator`, `Clock`.
- **`internal/domain/<aggregate>/`** (currently `identity`, `subject`, `course`) holds:
  - the entity and its constructor, which validates input and aggregates every field error;
  - `input.go`, `repository.go` (the port interfaces) and `errors.go` (codes plus predefined `Err*` values);
  - services taking `(ctx, actor, ...)`. Services also enforce admin-only rules (`actor.RequireAdmin()`), not just the HTTP layer.
  - Unit tests use in-memory fakes of the ports (`fakes_test.go` or at the top of `service_test.go`).
  - Reads that need another aggregate's data return a `View` (e.g. `course.View` adds `SubjectName`). A service that needs another aggregate declares a narrow reader interface (`course.SubjectReader`) instead of depending on that aggregate's service.
  - Other aggregates that create users use `identity.NewUser` and `identity.EnsureUnique` (in `identity/uniqueness.go`), never `UserService`.
- **`internal/adapter/postgres`**: bun repositories, each with an unexported row struct.
  - Embed `repo` and use `r.idb(ctx)` so that queries join the transaction from `TxManager.WithinTx`.
  - Embed `timestamps` in rows (it bumps `updated_at` on update).
  - Wrap write errors with `translate(err, constraintErrors{...})` so that constraint names map to domain errors. The same FK means different things on insert and on delete (`fk_courses_subject_id` is "subject not found" when creating a course, "subject in use" when deleting a subject), so use a separate map per operation.
  - Views that join other tables use a row struct embedding the base row with `bun:",extend"` plus `scanonly` columns (see `courseViewRow`).
  - Return the aggregate's `ErrXNotFound` on `sql.ErrNoRows`.
  - bun 1.3 uses `int64` for `Limit`/`Offset`/`ScanAndCount`.
- **`internal/adapter/postgres/pgtest`**: `pgtest.DB(t)` returns a migrated, truncated Postgres from testcontainers, one container per test package. Tests that use it must not call `t.Parallel()`. Repo tests are in `package postgres_test`.
- **`internal/adapter/security`**: bcrypt hasher, HS256 JWT issuer and verifier (claims `sub`, `role`), and a random refresh-token secret generator.
- **`internal/adapter/httpapi`**: the HTTP layer. The directory isn't named `http`, so the package doesn't shadow net/http.
  - `router.go`: `NewRouter` sets up the middleware chain and mounts each handler under `/api/v1`.
  - `handler/`: each handler declares a small service interface and implements `MountRoutes(public, protected chi.Router)`. `protected` is already behind `middleware.Auth`. Don't add role middleware: the service checks permissions (see `docs/architecture.md`, Authorization). Shared helpers such as `actorAndID(r)` live in `handler/handler.go`.
  - `dto/`: request and response structs. `dto.Date` handles `YYYY-MM-DD`, `dto.Nullable[T]` handles PATCH fields, and `dto.NewPageResponse` builds paginated responses.
  - `httpx/`: `DecodeBody[T](w, r)` (JSON + shape-only `validate` tags, errors as `fields[]`), `GetParamID`, the query helpers, `GetPageRequest`, and `httpx.Actor(r)`.
  - `respond/`: `respond.Error(w, r, err)` is the single place where errors become HTTP responses. It handles `*respond.APIError` and `*derr.Error`; anything else becomes a logged 500.
  - `middleware/`: RequestID (attaches a request logger, read it with `zerolog.Ctx`), Logging, Recoverer, CORS, JSON 404/405, `Auth(verifier)` (reads the `access_token` cookie), `RequireRole`.
- **`internal/app`**: the composition root. `app.New(cfg, db, Overrides{})` wires everything. `cmd/goschool` and the HTTP tests in `internal/app/app_test.go` both use it. Add new repos, services and handlers here.

Auth: login sets `access_token` (Path=/, SameSite=Lax) and `refresh_token` (Path=/api/v1/auth, SameSite=Strict). Both cookies are HttpOnly, and `Secure` only when `ENVIRONMENT=prod`. Refresh tokens are stored as SHA-256 hashes in `tokens.body`. They are rotated on each refresh, and reusing a used token revokes all of that user's tokens.

## Database

- The schema lives in `internal/db/migration/*.sql` and is embedded with `//go:embed`. Each file uses goose `-- +goose Up` / `-- +goose Down` sections. Right now everything is in `0001_init.sql`.
- Convention in that file: create the tables first, then the indexes, then every foreign key as a named `ALTER TABLE ... ADD CONSTRAINT fk_<table>_<cols>` at the end. The Down section drops constraints, then indexes, then tables, in reverse order. When you add a table or FK, update both sections.
- Composite foreign keys enforce business rules:
  - `teacher_courses(teacher_id, subject_id)` → `teacher_subjects`: a teacher can only teach courses in subjects they are qualified for.
  - `teaching_assignments` → both `class_courses(class_id, course_id, academic_year)` and `teacher_courses(teacher_id, course_id)`.
  - `courses` has an extra unique index on `(id, subject_id)` only so that it can be an FK target.
- Partial unique indexes enforce one open (`end_date IS NULL`) enrollment per student and year (`idx_student_class_history_open`) and one open homeroom assignment per class and year (`idx_class_homeroom_assignments_open`). Other rules (score-column name uniqueness, the semester range) are deliberately left to the domain.
- `academic_year` is an INT holding the starting year (2025 means 2025–2026). Connections set `TimeZone=UTC`. Scores range from 0 to 10, and `score_columns.ratio_percent` sets each column's weight.
