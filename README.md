# goschool

A school management REST API in Go: users (student / teacher / admin), subjects, courses, classes, teaching assignments, timetables and scores.

- API contract: [`api/openapi.yaml`](api/openapi.yaml)
- Architecture, business rules and build status: [`docs/architecture.md`](docs/architecture.md)
- Postman collection: [`api/postman/goschool.postman_collection.json`](api/postman/goschool.postman_collection.json)

> **Status:** health, authentication, user administration, subjects and courses are implemented so far (phases 0–2). The other endpoints in the OpenAPI file return 404 until their phase lands.

## Prerequisites

- **Go 1.25+.** With an older Go, the `go` command downloads 1.25 automatically.
- **Docker** (with Docker Compose), for the local database and the integration tests.

## Run it locally

```bash
# 1. Configuration: copy the template and adjust it if needed
cp .env.example .env

# 2. Start Postgres 16 (data is kept in the `pgdata` volume)
docker compose up -d --wait

# 3. Create the schema
go run ./cmd/goschool migrate

# 4. Create the first admin (ADMIN_USERNAME / ADMIN_PASSWORD from .env; does nothing if it exists)
go run ./cmd/goschool seed-admin

# 5. Start the API on http://localhost:8080 (Ctrl-C to stop)
go run ./cmd/goschool serve
```

Check it:

```bash
curl http://localhost:8080/api/v1/healthz
# {"status":"ok"}
```

### Try it with curl

Authentication uses HttpOnly cookies, so keep a cookie jar:

```bash
U=http://localhost:8080/api/v1
curl -c jar.txt -X POST $U/auth/login -d '{"username":"admin","password":"change-me-please"}'
curl -b jar.txt $U/auth/me
curl -b jar.txt "$U/users?page_size=5"
curl -b jar.txt -c jar.txt -X POST $U/auth/refresh      # rotates both cookies
curl -b jar.txt -c jar.txt -X POST $U/auth/logout
```

### Try it with Postman

1. **Import** → select `api/postman/goschool.postman_collection.json`.
2. Open the collection's **Variables** tab. Check that `adminUsername` and `adminPassword` match your `.env`, and that `baseUrl` matches `LISTEN_PORT`.
3. Run **Auth → Login (admin)** first. Postman keeps the cookies, so every later request is authenticated.
4. When you get `401 unauthorized` after about 15 minutes, run **Auth → Refresh tokens** or log in again.

Every request has a test script that checks the expected status, so you can also run the whole collection with the **Collection Runner**. The folders are already in the right order, and the last one, *Session (run last)*, changes the admin password and logs out. The *Change my password* request updates the `adminPassword` variable for you. Run it a second time to switch back to the original password, which is still the one in `.env`.

## Commands

| Command | What it does |
|---|---|
| `go run ./cmd/goschool migrate` | Apply pending migrations (embedded goose SQL) |
| `go run ./cmd/goschool rollback` | Roll back the latest migration. Today that drops **all** tables. |
| `go run ./cmd/goschool seed-admin` | Create the admin from `ADMIN_USERNAME` / `ADMIN_PASSWORD` if it doesn't exist |
| `go run ./cmd/goschool serve` | Run the HTTP API |
| `go test ./...` | All tests. The integration tests start their own throwaway Postgres in Docker. |
| `go test -short ./...` | Unit tests only, no Docker needed |
| `BUNDEBUG=2 go run ./cmd/goschool serve` | Log every SQL query (`1` = failed queries only) |

## Configuration

All settings come from environment variables. A `.env` file in the working directory is loaded automatically. See [`.env.example`](.env.example) for the full list with defaults.

| Variable | Notes |
|---|---|
| `ENVIRONMENT` | `dev`, `test` or `prod`. In `prod` logs are JSON and cookies get the `Secure` flag, which requires HTTPS. |
| `PG_HOST`, `PG_PORT`, `PG_USER`, `PG_PASSWORD`, `PG_DB_NAME`, `PG_SSL_MODE` | Postgres connection. docker-compose uses the same values. |
| `LISTEN_IP`, `LISTEN_PORT` | Server address (default `0.0.0.0:8080`) |
| `CORS_ALLOWED_ORIGINS` | Comma-separated origins allowed to call the API from a browser |
| `JWT_SECRET` | At least 32 characters. **Change it** for anything beyond local use. |
| `JWT_ACCESS_EXPIRES`, `JWT_REFRESH_EXPIRES` | Go durations, e.g. `15m`, `168h` |
| `ADMIN_USERNAME`, `ADMIN_PASSWORD` | Only used by `seed-admin` |

## Troubleshooting

- **`ping postgres: ... connection refused`**: the database isn't running. Start it with `docker compose up -d --wait`, and check that `PG_PORT` matches the port mapped in `docker-compose.yml`.
- **Port 5432 already in use**: another Postgres is running. Set `PG_PORT=5433` in `.env` (docker-compose maps the same variable), then run `docker compose up -d --wait` again.
- **`invalid jwt (JWT_*) config`**: `JWT_SECRET` is missing or shorter than 32 characters.
- **Login works but the next request returns 401 in Postman or a browser**: with `ENVIRONMENT=prod`, cookies are `Secure` and are only sent over HTTPS. Use `dev` locally.
- **Start from a clean database**: `docker compose down -v` deletes the volume. Then run `migrate` and `seed-admin` again.
