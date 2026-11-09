---
name: developer
description: Implementation guidelines for building features in this repo (Go todo API on gin) — architecture layers, storage interface, config, error handling, testing, and required commands. Use when writing or modifying code in this repository.
---

# Developer guidelines — project-swe (todo API)

## Architecture — separation of concerns (strict)

Requests flow through exactly these layers; never skip one or reach across:

```
internal/routes  (gin handlers: bind/validate, auth, map errors → HTTP, JSON)
       ↓
internal/task/service.go  (business rules: ownership, status transitions, pagination math)
       ↓
internal/task/repository.go  (Repository interface — the persistence contract)
       ↓
internal/storage/postgres.go  (the only place pgx/SQL lives)
```

- **Handlers** (`routes/get.go`, `post.go`, `patch.go`, `delete.go`, `auth.go`): no SQL, no business logic, no direct DB access. Bind and validate input, call the service, map domain errors to status codes, write JSON.
- **Service** (`task/service.go`): no gin import, no SQL. Owns authorization scoping (non-admin callers are pinned to their own user ID server-side), status transition rules (`task/status.go`), and pagination math.
- **Repository interface** (`task/repository.go`): the DB-agnostic contract. Swapping databases must only require a new implementation in `internal/storage/` — never a change in routes or service.
- **Storage** (`storage/postgres.go`): implements the interface; the only package that may import the driver. Map driver errors to domain errors (`internal/errors`) before returning them.
- Cross-cutting code lives in its own package (`internal/auth`, `internal/errors`, `internal/events`, `internal/config`) — don't inline it into handlers.

## Ownership rule (hard requirement)

**Every task must have a user id.**

- The service refuses to create a task without an owner (fail with a domain error, don't persist).
- The `tasks.user_id` column is `NOT NULL` with **no database default** — an insert that omits `user_id` must fail at the database, never silently create an ownerless row.
- Keep `internal/storage/postgres.go` and `migrations/init.sql` in sync on this.

## Config rules (`internal/config` + `config.yaml` / `config.docker.yaml`)

Everything environment-specific comes from config — nothing hardcoded:

- `server.port`, `server.metrics_port` (Prometheus), `server.monitoring`
- `database.url`, pool/connection settings: `max_conns`, `min_conns`, `max_conn_lifetime`, `max_conn_idle_time`
- `auth.admin_username` / `auth.admin_password` (admin login comes from config, never a DB row)
- `log.level` (debug|info|warn|error)
- `pagination.default_limit` / `pagination.max_limit` — the limits are **defined in config** and flexible from the API: when the request omits `?limit=`, the config default applies; a request `?limit=` above `max_limit` is capped at `max_limit`; `page` ≥ 1, offset = (page−1)·limit
- `seeder.count`

When adding a setting: add it to the config struct, both YAML files, and a test in `internal/config/config_test.go`.

## HTTP conventions (gin)

- Register all routes in `internal/routes/register.go`; group `/tasks` under `auth.RequireAuth()`.
- New endpoints get swagger annotations; regenerate and commit docs with `make swag`.
- Error mapping: validation → 400, not found → 404, conflict/duplicate → 409, unauthorized → 401, forbidden → 403, everything else → 500 with a generic body (never leak SQL errors or stack traces).
- Soft delete: `DELETE /tasks/:id` closes a task (status → close), it is not removed from the DB; `List` excludes closed tasks.

## Testing requirements (DoD: ≥70% coverage, `make test-cov` stores + shows the report)

- Unit tests use the in-package mocks (`internal/task/mock.go`, `internal/user/mock.go`) — **no DB required** for `make test`.
- Table-driven tests; cover success paths *and* error paths (a service method that only tests the happy path is not done).
- **Auth tokens in tests come from the API**: exercise signup/login (or the token generator through the real flow) — never hardcode tokens.
- Storage tests go behind the `integration` build tag and run via `make test-integration`; API smoke tests live in `qa/` behind the `qa` tag.
- `make test-cov` must exist and: run unit tests with `-coverprofile=coverage.out`, render `coverage.html`, and print a per-package summary. Keep it green and ≥70% — add or extend tests for any new code path before considering a feature done.
- Run the gates before declaring done: `make lint-test`, `make test-cov`, `make test-integration` (storage changes), `make test-qa` (API changes).

## Git conventions

- Branch from `main` and name branches by type: `feat/<short-description>` for new features, `fix/<short-description>` for bug fixes, `refactor/...` for restructures, `test/...` for test-only work, `chore/...` for tooling, `docs/...` for documentation. Never commit straight to `main`.
- Commit messages are conventional: `feat:`, `fix:`, `refactor:`, `test:`, `chore:`, `docs:` with a short imperative subject line.

## Definition of done for a feature

A feature is done only when all of these hold:

1. Signup → login works and issues a token; users see only their own tasks (list/get/update/delete), paginated and filterable.
2. Admin (config credentials) can list all users' tasks and filter by `user_id` and the other filters.
3. Every task is created with a user id; there is no code path that can persist an ownerless task.
4. Layers respected, persistence only via the `Repository` interface.
5. New settings in config (both YAML files) with tests.
6. `make lint-test` clean, coverage ≥70% with the report stored, integration and QA suites pass.
7. Swagger docs regenerated and in sync with the routes.
