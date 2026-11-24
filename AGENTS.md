# Agent Guidelines for CAATSM

## Commands
- **Build**: `make build` (bin/receiver)
- **Run**: `make run-dev` (dev mode), `make run-prod` (prod mode)
- **Lint**: `make lint` (golangci-lint)
- **Test**: `make test` (Unit/Ginkgo), `make test-int` (Integration/Docker)
- **Single Test**: `ginkgo -r -v --focus "Test Description" ./path/to/package`
- **Coverage**: `make coverage` (>80% target)

## Code Style & Architecture
- **Blueprint**: Follow [`docs/architecture.md`](docs/architecture.md) — 3-layer high-cohesion Go layout: `internal/telegram/{aviation,weather}`, `internal/pipeline`, `internal/{nats,postgres,config,observability,buildinfo}`, `cmd/receiver`. Enforce layer boundaries via `.golangci.yml` (`depguard`).
- **Parsers**: Composite routing (`telegram.Router`) over aviation (ARR/DEP/CNL/DLA/FPL) and weather (METAR/SPECI/TAF); keep `internal/telegram/...` pure Go (stdlib only) with zero external IO/logging.
- **Interfaces**: Consumer-side small interfaces (*Accept interfaces, return structs*) in `internal/pipeline/ports.go` and `internal/nats/consumer.go`; avoid `any`.
- **Formatting**: Run `go fmt ./...` and `goimports` before committing.
- **Naming**: `CamelCase` (exported), `camelCase` (private). Package names match dirs. Avoid package/type name stuttering.
- **Errors**: Wrap with context (`fmt.Errorf("...: %w", err)`). Use `errors.Is` / `errors.As`.
- **Testing**: Ginkgo BDD style (`Describe`, `It`) for unit tests; standard `testing` with `-tags=integration` in `test/integration/`. Mock consumer interfaces.
- **Observability**: Propagate `context.Context`. Use OpenTelemetry (traces/metrics/TraceContext propagation) and Zap structured logging.

## Cursor Rules (.cursor/rules/do.mdc)
- **Expertise**: Go, Microservices, Pragmatic Layered Modularity, TDD.
- **Security**: Input validation, secure defaults, retries/backoff.
- **Perf**: Benchmarks, minimize allocations.
