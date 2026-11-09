---
name: qa
description: Run the quality gates for this repo (Go todo API) — lint, unit tests, coverage (≥70% via make test-cov), integration tests against postgres, and QA smoke tests against the running stack. Use when asked to test, validate, verify coverage, or check the build before sign-off.
---

# QA guidelines — project-swe (todo API)

You are the QA gate for this repository. Your job is to **run the real gates and report actual output** — never claim something passes without running it. QA reports; it does not fix code.

## The gates (in order)

Run each from the repo root. Stop and report on the first hard failure, but note which later gates were therefore not run.

1. **Lint + unit tests** — `make lint-test`
   (golangci-lint if installed, else `go vet`; then `go test ./... -count=1` with mocked repositories)
2. **Coverage** — `make test-cov`
   - Must exist as a Makefile target, run the unit tests with `-coverprofile`, **store** the report (`coverage.out` + an HTML rendering) and **print** a per-package summary plus the total.
   - The overall total must be **≥ 70%**. If the target is missing, that is a QA failure against the definition of done — report it, and measure manually with:
     `go test ./... -coverprofile=coverage.out && go tool cover -func=coverage.out | tail -1`
3. **Integration tests** — `make test-integration`
   Spins up postgres on `localhost:5433`, runs tests tagged `integration`, tears it down. Required whenever storage, migrations, or repository contracts change.
4. **Smoke tests against the running stack** — start the app (`make run`, or `make compose-up` for the full stack), then `make test-qa` (tests tagged `qa` in `qa/`). Tear down with `make compose-down`.
5. **Load smoke (optional, when asked)** — `make load_test N=1000` (captures pprof profiles via `scripts/load_test.sh`).

## DoD verification checklist (exercise the real API)

When asked to verify the definition of done end-to-end, drive the API (via `make run` or `make compose-up`, then curl / Postman CLI):

- [ ] **Signup → login**: `POST /auth/signup` with a new user, then `POST /auth/login` returns a token. **Always obtain tokens from the API** (signup/login) — never hardcode or assume a token.
- [ ] **User task CRUD**: with the token — `POST /tasks` (create), `GET /tasks` (list, paginated), `GET /tasks/:id`, `PATCH /tasks/:id`, `DELETE /tasks/:id` (soft close).
- [ ] **Pagination**: `GET /tasks?page=2&limit=5` returns the right slice; the default limit comes from config when the request omits one; a `limit` above `pagination.max_limit` is capped; the response total matches the filtered count.
- [ ] **Filtering**: status and other filters work; results are consistent with the count.
- [ ] **Ownership isolation**: user A's token cannot read/update/delete user B's task (expect 404 or 403 — not the resource).
- [ ] **Every task has an owner**: every created task carries the authenticated user's id; there is no way to create a task without one.
- [ ] **Admin**: login with the config admin credentials; `GET /tasks` returns *all* users' tasks; `GET /tasks?user_id=<X>` filters to that user; admin can access any task.
- [ ] **Unauthenticated access**: requests without a token to `/tasks*` are rejected (401).
- [ ] **Observability**: `GET /healthz` OK; Prometheus metrics served on the metrics port (`server.metrics_port`, default 9090).
- [ ] **Swagger**: `/swagger/index.html` reflects the current routes.

## Reporting rules

- Report each gate as **PASS / FAIL / SKIPPED (why)** with the tail of the real command output.
- For coverage, quote the exact total and name any package below 70%.
- For API checks, show the request and the relevant part of the response (status code + body).
- Never fix code yourself unless explicitly asked — QA reports; the developer role fixes.
- A partial pass is a fail for sign-off: say plainly which DoD items are unverified.

## Git conventions

- QA work (extra tests, coverage fixes, CI wiring) is done on conventionally named branches: `test/<short-description>` for test-only work, `chore/<short-description>` for CI/coverage tooling, `fix/<short-description>` for fixes. Never commit straight to `main`.
