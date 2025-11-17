# Repository Guidelines

## Project Structure & Module Organization
Application entry lives in `cmd/main`, while Clean Architecture layers live under `internal` (`domain`, `app`, `adapter`, and `infra`). Shared wiring and compiled providers sit in `pkg/di`, configs in `configs/config.<env>.toml`, docs in `docs`, and reusable test fixtures in `test`. Keep new assets near the layer they extend (e.g., new parsers in `internal/adapter/parser`).

## Build, Test, and Development Commands
- `make build` / `task build` — compile `./cmd/main` into `bin/receiver` with Wire-generated deps.
- `make run-dev` / `task run-dev` — run with `GO_ENV=dev`, respecting `configs/config.dev.toml`.
- `make lint` / `task lint` — execute `golangci-lint` with the repository config.
- `make test`, `make test-int`, `make test-all` — run Ginkgo unit suites, integration suites (`test/integration`), or both.
- `make coverage` — produce `coverage/coverage.html`; open it before merging substantial changes.

## Coding Style & Naming Conventions
Stick to idiomatic Go: tabs for indentation, `camelCase` for locals, `CamelCase` for exported APIs, and package names that match their directory. Always run `gofmt`/`goimports` (or rely on `go fmt ./...`) before opening a PR. Generated files belong under `/pkg/di` (Wire) or the directory they serve; never hand-edit `wire_gen.go`. Linting via `golangci-lint` is required before submission.

## Testing Guidelines
Unit specs live next to implementation files as `*_test.go` and rely on Ginkgo; keep descriptions declarative ("should parse DEP messages"). Integration suites in `test/integration` spin up NATS and TimescaleDB via Testcontainers; run them locally with Docker. Target coverage is whatever `make coverage` reports for the touched packages—raise regressions above 80% when practical.

## Commit & Pull Request Guidelines
Follow the existing history style: optional emoji prefix + imperative summary (e.g., `✨ Add telemetry recorder`). Reference tickets in the body (`Refs #123`) and explain config or schema migrations explicitly. Pull requests must describe the change, include relevant commands/logs, attach screenshots for dashboard updates, and call out any new flags or environment variables.

## Security & Configuration Tips
Store secrets in environment variables (`CAATSM_*`) rather than committing them. When introducing new configuration keys, update the matching `configs/config.<env>.toml` and document overrides in `README.md`. Review `docker-compose.dev.yml` before running integration tests to ensure local services are isolated from production infrastructure.
