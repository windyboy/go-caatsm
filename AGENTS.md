# Agent Guidelines for CAATSM

## Commands
- **Build**: `make build` (bin/receiver)
- **Run**: `make run-dev` (dev mode), `make run-prod` (prod mode)
- **Lint**: `make lint` (golangci-lint)
- **Test**: `make test` (Unit/Ginkgo), `make test-int` (Integration/Docker)
- **Single Test**: `ginkgo -r -v --focus "Test Description" ./path/to/package`
- **Coverage**: `make coverage` (>80% target)

## Code Style & Architecture
- **Structure**: Clean Architecture (`cmd/`, `internal/{domain,app,adapter,infra}`, `pkg/`).
- **Parsers**: Composite parser pattern with specialized sub-parsers (aviation, weather).
- **Domain**: Core domain types include aviation telegrams and weather reports.
- **Formatting**: Run `go fmt ./...` and `goimports` before committing.
- **Naming**: `CamelCase` (exported), `camelCase` (private). Package names match dirs.
- **Errors**: Wrap with context (`fmt.Errorf("...: %w", err)`). Use `errors.Is`.
- **Types**: Interface-driven development. Avoid `any`.
- **Testing**: Ginkgo BDD style (`Describe`, `It`). Table-driven. Mock interfaces.
- **Observability**: Propagate `context.Context`. Use OpenTelemetry (traces/metrics).
- **Generated**: NEVER edit `wire_gen.go` or `*_gen.go`.

## Cursor Rules (.cursor/rules/do.mdc)
- **Expertise**: Go, Microservices, Clean Arch, TDD.
- **Security**: Input validation, secure defaults, retries/backoff.
- **Perf**: Benchmarks, minimize allocations.
