# Agent Guidelines for CAATSM Repository

## Build/Test Commands
- **Build**: `make build` or `task build` (compiles to `bin/receiver`)
- **Run dev**: `make run-dev` or `task run-dev` (uses `configs/config.dev.toml`)
- **Lint**: `make lint` or `task lint` (golangci-lint required)
- **Unit tests**: `make test` (Ginkgo) or `ginkgo -r -v ./path/to/package` for single test
- **Integration tests**: `make test-int` (requires Docker)
- **All tests**: `make test-all`
- **Coverage**: `make coverage` (target: maintain >80% coverage)

## Code Style Guidelines
- **Formatting**: Use tabs, `go fmt ./...` or `goimports` before commits
- **Naming**: `camelCase` for locals/unexported, `CamelCase` for exported; package names match directories
- **Imports**: Standard library → third-party → internal (alphabetized within groups)
- **Types**: Use interfaces for ports, appropriate Go types; avoid `any` unless necessary
- **Error handling**: Wrap errors with context, use `errors.Is()` for checking
- **Generated code**: Never edit `/pkg/di/wire_gen.go` or other generated files
- **Linting**: `golangci-lint run ./...` required; fix all issues before PR

## Testing Guidelines
- **Unit tests**: Ginkgo BDD style next to implementation (`*_test.go`); declarative descriptions
- **Integration**: Testcontainers in `test/integration`; spin up NATS/TimescaleDB
- **Coverage**: Run `make coverage` before merging; address regressions

## Architecture
- **Structure**: Clean Architecture - `domain` (business logic), `app` (use cases), `adapter` (I/O), `infra` (framework deps)
- **Entry point**: `cmd/main`
- **Config**: `configs/config.<env>.toml`; secrets via `CAATSM_*` env vars
