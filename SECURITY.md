# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 0.0.1   | :white_check_mark: |

## Reporting a Vulnerability

**Please do not report security vulnerabilities through public GitHub issues.**

Instead, please report them via one of the following methods:

- **Email**: security@masterfabric.co
- **GitHub Security Advisory**: Use the [Security tab](https://github.com/masterfabric-go/masterfabric-go/security/advisories/new) to create a private security advisory

### What to Include

When reporting a security vulnerability, please include:

- Description of the vulnerability
- Steps to reproduce
- Potential impact
- Suggested fix (if any)

### Response Timeline

- **Initial Response**: Within 48 hours
- **Status Update**: Within 7 days
- **Fix Timeline**: Depends on severity (typically 30-90 days)

## Trust Model

masterfabric-go is a multi-tenant API management platform. The server is trusted to enforce authentication, authorization, tenant isolation, and policy rules. Clients are untrusted. Administrative operators are trusted to configure secrets, CORS origins, and infrastructure bindings correctly.

### Security Controls (current)

| Area | Control |
| ---- | ------- |
| Authentication | JWT bearer tokens on `/api/v1` administrative routes |
| Authorization | RBAC with wildcard-aware permission checks on state-changing admin routes |
| Multi-tenancy | `organization_id` and `app_id` enforced in repositories and gateway handlers |
| Error handling | Generic 5xx client messages; detailed causes logged server-side |
| Request bodies | Global `MAX_BODY_BYTES` limit (default 1 MiB) via `MaxBytesReader` |
| CORS | `CORS_ALLOWED_ORIGINS` allow-list; credentials disabled for `*` or empty list |
| Database DSN | Credentials escaped through `net/url` |
| Pagination | `page` clamped to `MaxPage` to prevent negative SQL offsets |
| Outbound proxy | No redirect following; bounded response reads for gateway HTTP proxy |
| Health probes | Readiness returns generic unhealthy markers only |
| Containers | Non-root runtime user; Go 1.26.4 builder; alpine 3.24 runtime |
| Local compose | Database/cache/Kafka ports bind to loopback by default |

### Environment Variables

| Variable | Purpose | Production guidance |
| -------- | ------- | ------------------- |
| `JWT_SECRET` | HS256 signing key | Required; never use the default value |
| `CORS_ALLOWED_ORIGINS` | Comma-separated browser origins | Set explicit origins; avoid `*` |
| `MAX_BODY_BYTES` | Request body cap | Keep at or below gateway policy limits |
| `DB_SSLMODE` | PostgreSQL TLS mode | Use `require` or stricter |
| `DB_HOST_BIND` | Compose host bind for Postgres | Keep `127.0.0.1` outside isolated dev machines |

### Security Best Practices

When using masterfabric-go in production:

- Change default `JWT_SECRET` to a strong, random value
- Use SSL/TLS for database connections (`DB_SSLMODE=require`)
- Set `CORS_ALLOWED_ORIGINS` to explicit trusted origins
- Enable rate limiting for production workloads via endpoint policies
- Regularly update dependencies (`go get -u ./...`) and run `govulncheck`
- Review and rotate API keys regularly
- Monitor audit logs for suspicious activity
- Use environment variables for sensitive configuration
- Keep Docker images updated
- Restrict `/metrics` and health endpoints at the network edge

### Accepted Risks

| Risk | Rationale | Mitigation |
| ---- | --------- | ---------- |
| Unauthenticated `/metrics` and `/health/*` | Required for orchestrator probes and Prometheus scraping | Restrict by network policy or reverse-proxy auth |
| HS256 JWT | Simplicity for single-tenant deployments | Rotate secrets; prefer external identity for large fleets |
| Dynamic SQL gateway handler | Admin-defined table names via endpoint configuration | RBAC on endpoint creation; audit endpoint changes |
| Gateway HTTP proxy (gosec G704) | Managed endpoints may proxy to operator-configured backends | RBAC on endpoint creation; redirect refusal; response size cap |
| Development compose credentials | Convenience for local bootstrap | Loopback bind + documented dev-only posture |

### Verification Commands

```bash
go build ./... && go vet ./... && go test ./...
go run golang.org/x/vuln/cmd/govulncheck@latest ./...
go run github.com/securego/gosec/v2/cmd/gosec@latest -quiet ./...
```

### Security Updates

Security updates will be:

- Documented in CHANGELOG.md
- Tagged with security labels
- Released as patch versions

Thank you for helping keep masterfabric-go secure!
