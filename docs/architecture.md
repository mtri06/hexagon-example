# goschool architecture

goschool is a school management HTTP service built on **hexagonal architecture** (ports and adapters).
The HTTP contract lives in [`api/openapi.yaml`](../api/openapi.yaml). The database schema lives in
[`internal/db/migration`](../internal/db/migration).

```
            driving adapter                core                     driven adapters
   ┌────────────────────────┐   ┌──────────────────────────┐   ┌──────────────────────────┐
   │ adapter/httpapi        │──▶│ domain/<aggregate>       │◀──│ adapter/postgres (bun)   │
   │  handler, dto, middle- │   │  entity + Service        │   │ adapter/security         │
   │  ware, respond         │   │  repository.go (ports)   │   │  (bcrypt, JWT)           │
   └────────────────────────┘   │ domain/port, derr, shared│   └──────────────────────────┘
                                └──────────────────────────┘
              internal/app wires everything (composition root); cmd/goschool runs it
```

## Project layout

```
cmd/goschool/main.go            CLI: migrate | rollback | seed-admin | serve (graceful shutdown)
api/openapi.yaml                HTTP contract, the source of truth for the transport layer
docs/architecture.md            This file
internal/
  app/                          Composition root: builds repos, services, handlers and the router (used by cmd and HTTP tests)
  config/                       Loads .env plus the environment into one Config; each command validates the sections it needs
  logger/                       zerolog setup
  db/migration/                 goose SQL, embedded with //go:embed
  domain/                       CORE. Must not import bun, net/http, chi, jwt, bcrypt or pgx
    derr/                       Domain error type (see "Errors")
    shared/                     Value objects: AcademicYear, Semester, DateRange, Page/PageRequest, Actor{UserID, Role}
    port/                       Cross-cutting driven ports: TxManager, PasswordHasher, TokenIssuer, SecretGenerator, Clock
    identity/                   User entity, Role, login / refresh / logout / change and reset password, Token entity
    student/                    Student = User + StudentProfile
    teacher/                    Teacher = User + TeacherProfile; subject and course qualifications
    subject/                    Subject
    course/                     Course (belongs to a subject, active or inactive)
    class/                      Class, ClassCourse (curriculum per year), HomeroomAssignment, Enrollment
    assignment/                 TeachingAssignment
    timetable/                  TimetableEntry; slot and teacher-clash rules
    grading/                    ScoreColumn, Score, Gradebook, weighted-average computation
  adapter/
    postgres/                   bun implementation of every repository port
      db.go                     Opens *bun.DB (pgdriver), sets TimeZone=UTC, BUNDEBUG query logging
      tx.go                     TxManager; stores bun.Tx in ctx; repo.idb(ctx) returns bun.IDB
      errors.go                 pg 23505/23503/23514 → derr, keyed by constraint name; `timestamps` row mixin bumps updated_at
      <aggregate>_repo.go       Unexported bun row struct + mapping + one repository implementation
      pgtest/                   testcontainers Postgres for integration tests (migrated, truncated per test)
    security/                   bcrypt PasswordHasher, HS256 JWT TokenIssuer, random refresh-token generator
    httpapi/                    (not `http`, to avoid shadowing net/http)
      router.go                 chi router, /api/v1 mount, middleware chain, handler registration
      handler/                  One file per resource. Each handler declares the narrow service interface it uses
      dto/                      Request and response structs that mirror openapi.yaml, plus mapping to and from domain types
      httpx/                    DecodeBody[T], path, query and pagination helpers
      middleware/               Auth (cookie JWT → shared.Actor in ctx), RequireRole, CORS, request logging, JSON 404
      respond/                  JSON(w, status, v), Error(w, r, err): the ONLY place errors become HTTP
```

### Per-aggregate package convention (`internal/domain/<aggregate>/`)

| File | Content |
|---|---|
| `<aggregate>.go` | Entity struct (no tags), `New<Entity>(input) (*Entity, error)` constructor, invariant-preserving methods (`End(date)`, `Deactivate()`, …), `validate()` |
| `input.go` | `Create<X>Input`, `Update<X>Input` (pointer fields for PATCH semantics), list filters |
| `repository.go` | Repository port(s) the service needs. Methods take `context.Context` and return domain types or `derr` |
| `service.go` | `Service` struct + `NewService(deps...)`; one exported method per use case; every method takes `ctx` and, when authorization depends on data, `shared.Actor` |
| `errors.go` | Error-code constants for this aggregate (`CodeCourseNotInCurriculum = "course_not_in_curriculum"`) |
| `*_test.go` | Unit tests against in-memory fakes of the ports |

A service may depend on **another aggregate's repository port**, never on another service's
implementation. Example: `assignment.Service` needs `class.CurriculumReader` and
`teacher.QualificationReader`. Keep those cross-aggregate ports small: declare a read-only
interface in the consuming package, or reuse a narrow one exported by the owning package.

## Responsibilities and dependency rules

| Layer | Owns | May import |
|---|---|---|
| `domain/*` | Entities, invariants, **all business validation**, use cases, data-dependent authorization (e.g. "a teacher may only grade class-courses they are assigned to"), ports | stdlib, `domain/derr`, `domain/shared`, `domain/port`, other domain packages |
| `adapter/postgres` | SQL through bun, row ↔ entity mapping, transactions, translating DB errors to `derr` as a safety net | `domain/...`, bun, pgdriver |
| `adapter/security` | Password hashing, token signing and verification | `domain/port` |
| `adapter/httpapi` | Wire format, shape validation, coarse role gating, status mapping. **No business rules** | `domain/...`, chi, validator |
| `config`, `logger` | Infrastructure setup | anything |
| `app`, `cmd/goschool` | Wiring only | everything |

Arrows only point inward. The domain never learns about HTTP or SQL, so you can test every
rule with in-memory fakes.

### Validation in two levels
1. **HTTP shape** (`dto` + `validate` tags via `httpx.DecodeBody`): JSON well-formed, required
   fields present, types, enum spelling, obvious bounds from the OpenAPI schema. Failure →
   `400 invalid_body` with `fields` named by JSON field.
2. **Domain** (constructors and services): every business rule, trimming and normalization,
   cross-field checks (`end_date ≥ start_date`), and checks that need the DB (uniqueness,
   qualification, enrollment). Failure → `derr` with the appropriate kind.

If the HTTP layer were removed, the domain would still reject every invalid state.

### Transactions
```go
// domain/port
type TxManager interface {
    WithinTx(ctx context.Context, fn func(ctx context.Context) error) error
}
```
The postgres adapter opens a `bun.Tx`, puts it in `ctx`, and every repo runs its queries on
`idb(ctx)`, which is the tx when present and `*bun.DB` otherwise. Services wrap multi-write use
cases (create teacher = users + teacher_profiles; transfer enrollment; gradebook bulk upsert) in
`WithinTx` without importing bun. Checks that guard against races (one open homeroom per class
and year, ratio total ≤ 100) run inside the same tx and use `SELECT … FOR UPDATE` on the scope row
through a dedicated repo method (`LockScope`).

### Authorization
- `middleware.Auth` verifies the JWT and puts a `shared.Actor{UserID, Role}` in the context.
  `middleware.RequireRole` blocks roles that can never access a route.
- Rules that depend on data ("student reads only self", "teacher grades only assigned
  class-courses", "teacher sees only students they teach") live in domain services. Services
  receive `actor` explicitly and return `derr.Forbidden`.

## Errors

`internal/domain/derr` replaces the old `apperr` package:

```go
type Kind uint8
const (
    KindInvalid Kind = iota + 1 // input breaks an invariant              → 400
    KindUnauthenticated         // bad credentials / token                → 401
    KindForbidden               // actor not allowed                      → 403
    KindNotFound                // entity or referenced entity missing    → 404
    KindConflict                // uniqueness, or still referenced        → 409
    KindRuleViolation           // valid input that breaks a business rule → 422
)

type FieldError struct{ Field, Message string }

type Error struct {
    Kind   Kind
    Code   string       // stable, machine-readable, e.g. "teacher_not_qualified_for_course"
    Msg    string       // human-readable
    Fields []FieldError // for KindInvalid
    Err    error        // wrapped cause, never serialized
}
func (e *Error) Error() string
func (e *Error) Unwrap() error

func Invalid(code, msg string, fields ...FieldError) *Error
func NotFound(code, msg string) *Error
func Conflict(code, msg string) *Error
func RuleViolation(code, msg string) *Error
func Forbidden(code, msg string) *Error
func Unauthenticated(code, msg string) *Error
func (e *Error) Wrap(err error) *Error
// FieldErrors collects field → message pairs in validate() and returns nil if empty:
type FieldErrors map[string]string
func (f FieldErrors) Err(code, msg string) error
```

- Error codes are constants in each aggregate's `errors.go` and are listed in the OpenAPI
  operation descriptions.
- `adapter/httpapi/respond.Error` is the single mapping point: `*derr.Error` → status by `Kind`,
  body `{code, message, fields}`. Anything else is logged with the request id and returned as
  `500 internal_error`.
- Repositories return `derr.NotFound` for missing rows. Unexpected DB errors are returned wrapped
  and left unclassified, so they end up as a 500.

## Persistence with bun

- Driver: `github.com/uptrace/bun` + `bun/dialect/pgdialect` + `bun/driver/pgdriver`. pgx and sqlx
  are removed. goose keeps running migrations through `database/sql` on the same `*sql.DB`
  (`bun.DB.DB`).
- Row structs are unexported types in each `<aggregate>_repo.go` and are **not** domain entities. Repos map between
  them, which keeps bun tags out of the domain.
- `updated_at` is set in a `BeforeAppendModel` hook because the schema has no trigger.
- List queries return `(items, total)` using `ScanAndCount`.

## Business rules (summary)

| Area | Rule | Error |
|---|---|---|
| Users | username unique; email unique when set; DOB in the past; inactive users cannot log in; an admin cannot deactivate themselves | 409 / 400 / 401 / 422 |
| Auth | refresh tokens are rotated; reusing a used token revokes all of the user's tokens; password change or reset and deactivation revoke all tokens; tokens are stored as SHA-256 hashes | 401 |
| Students | `admission_date ≥ date_of_birth`; `graduated_date > admission_date`; graduated students cannot be enrolled | 400 / 422 |
| Teachers | course qualification requires subject qualification; removing a qualification that is still used is blocked | 422 / 409 |
| Courses | `subject_id` immutable; inactive courses cannot be added to a curriculum or qualification | 422 |
| Curriculum | (class, course, year) unique | 409 |
| Homeroom | one open assignment per class and year; one homeroom class per teacher and year | 422 |
| Enrollment | one open enrollment per student and year; transfers are atomic end-then-start | 422 |
| Dates | `end_date ≥ start_date`; dates inside the academic year (Aug 1 of `Y` to Jul 31 of `Y+1`) | 400 |
| Teaching assignment | class-course in curriculum; teacher active and qualified; one open assignment per class-course-year; teacher swap = end + create | 422 |
| Timetable | slot free for the class; course in curriculum; assigned teacher not booked elsewhere in the same slot; semester ∈ {1, 2} | 409 / 422 |
| Score columns | name unique per (course, year, semester); Σ ratio ≤ 100; average only when Σ = 100 | 409 / 422 |
| Scores | 0–10, 2 decimals; student enrolled in class that year; column matches course, year and semester; teacher holds an active assignment; `recorded_by = actor` | 400 / 422 / 403 |

## Adding a new aggregate (checklist)
1. Migration: table, then indexes, then named FKs, and mirror them in the Down section (see
   CLAUDE.md).
2. `domain/<agg>/`: entity + constructor + validate, inputs, repository port, service, error codes, tests.
3. `adapter/postgres/<agg>_repo.go` (row struct + repo) and a `pgtest`-based `<agg>_repo_test.go`.
4. `api/openapi.yaml`: paths + schemas + error codes in the descriptions.
5. `adapter/httpapi/dto/<agg>.go` + `handler/<agg>_handler.go` with `MountRoutes(public, protected chi.Router)`.
6. Wire everything in `internal/app/app.go` and add HTTP tests in `internal/app`.

## Schema notes
Enforced in the DB:
- `idx_student_class_history_open`: at most one open enrollment (`end_date IS NULL`) per student
  and academic year.
- `idx_class_homeroom_assignments_open`: at most one open homeroom assignment per class and
  academic year.

The domain still checks both rules first so it can return a precise `422`. The postgres adapter
maps a violation of either index (a race) to the same error code.

Deliberately left to the domain, not the DB (constraining them in SQL would make legitimate
corrections harder):
- Score column names unique per (course, year, semester), and `semester` ∈ {1, 2}.
- `teaching_assignments.teacher_id` is nullable and means "vacant". With the default
  `MATCH SIMPLE`, the composite FK to `teacher_courses` is skipped when it is NULL.
- `updated_at` has no trigger; bun sets it in a hook.
- `tokens.body` stores `sha256(token)`, never the raw refresh token.
- `scores.score_column_id` is `ON DELETE CASCADE`; the API guards it with `force=true` on column
  deletion.
- A score is tied to a course, not a class, through its column. The gradebook therefore scopes
  by the student's enrollment in the class for that year.

## Build status
See the phased plan; each phase ships one vertical slice (domain → bun → HTTP → wiring → tests).

- [x] Phase 0: foundation (config, derr/shared/port, bun + tx, router, respond, CLI, pgtest, docker-compose)
- [x] Phase 1: identity and auth (`/auth/*`, `/users*`, `/admins`, `seed-admin`)
- [ ] Phase 2: subjects and courses
- [ ] Phase 3: teachers and students
- [ ] Phase 4: classes, curriculum, homeroom, enrollments
- [ ] Phase 5: teaching assignments
- [ ] Phase 6: timetable
- [ ] Phase 7: grading
- [ ] Phase 8: hardening (OpenAPI contract tests)
