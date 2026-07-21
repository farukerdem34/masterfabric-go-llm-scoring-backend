# MasterFabric Go - Agent Instructions

## Quick Commands

```bash
# Development (hot-reload, full infra)
./dev.sh

# Single test
go test -run TestName ./path/to/package/...

# Full lint pipeline (format + vet + golangci-lint)
./scripts/lint.sh
gofmt -w .                     # Format all Go files

# Security scans
go run golang.org/x/vuln/cmd/govulncheck@latest ./...
go run github.com/securego/gosec/v2/cmd/gosec@latest -quiet ./...

# Build
make build                      # Output: bin/masterfabric
make docker-build               # Docker image via deployments/Dockerfile
```

## Architecture Gotchas

- **Clean Architecture enforced**: Domain → Application → Infrastructure. Never import infrastructure from domain.
- **Multi-tenancy**: Every query needs `organization_id` + `app_id` from context via `middleware.OrgIDFromContext(ctx)`.
- **Dynamic handlers**: Gateway routes to CRUD handlers based on `backend_service` + `backend_action` in endpoint definitions.
- **Event bus**: `KAFKA_ENABLED=true` uses Kafka; otherwise in-process channel bus.
- **Migrations run automatically** on server startup via `infraPostgres.RunMigrations`.

## Project Structure

```
cmd/server/main.go          # DI wiring, startup
internal/domain/             # Pure Go entities, interfaces
internal/application/        # Use cases, DTOs
internal/infrastructure/     # HTTP, Postgres, Redis, Kafka implementations
internal/gateway/            # API gateway pipeline
internal/shared/             # Cross-cutting: config, middleware, errors, events
scripts/                     # test.sh, lint.sh, seed.go, migrate.sh
```

## Testing

- Tests use `package xxx_test` (black-box) pattern.
- Table-driven tests with `testify/assert`.
- Mock interfaces, not implementations.
- Run single test: `go test -run TestName ./path/...`
- Coverage: `./scripts/test.sh -cover`

## Code Style

- Files: `snake_case.go`
- Interfaces: descriptive names (`EndpointRepository`, not `IEndpointRepository`)
- Constructors: `NewXxx()` pattern
- Errors: `ErrXxx` variables or `domainErr.New(code, message, cause)`
- Always wrap errors: `fmt.Errorf("context: %w", err)`

## Development

- Hot-reload uses `air` (installed automatically by `dev.sh`).
- `dev.sh` starts Docker services, waits for health, runs migrations, starts hot-reload server.
- `dev.sh` explicitly sets `KAFKA_ENABLED=true` (default is `false`; production uses in-process event bus).
- Server runs on `:8080`, Kafka UI on `:8090`.
- Environment variables control all config (see README for full list).

## Security Notes

- `JWT_SECRET` defaults to `change-me-in-production` with startup warning.
- CORS origins configurable; empty or `*` disables credentials.
- Request body limited to 1 MiB by default.
- 5xx errors return generic messages; details logged server-side.
