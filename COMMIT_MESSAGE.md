feat: initial project setup - enterprise multi-tenant SaaS backend platform

Initial commit for masterfabric-go - a production-ready, enterprise-grade
multi-tenant SaaS backend platform built with Go, following Domain-Driven
Design and Clean Architecture principles.

## Architecture & Design

- Domain-Driven Design (DDD) with bounded contexts:
  - IAM (Identity & Access Management)
  - Tenant & App Management
  - API Management
  - Observability & Audit
- Clean/Hexagonal Architecture with strict dependency rule
- Phase 1: Modular monolith (single binary, ready for service extraction)
- Event-driven architecture with Kafka integration

## Core Features

### IAM (Identity & Access Management)
- User registration and authentication (JWT + bcrypt)
- Role-Based Access Control (RBAC) with permissions
- Organization user management
- Multi-tenant user isolation

### Tenant & App Management
- Organization creation and management
- Application lifecycle management
- API key generation and management
- Tenant-scoped resource isolation

### API Management
- Endpoint definition and management
- Policy-based access control (rate limiting, authentication requirements)
- Endpoint retirement and versioning

### Observability & Audit
- Comprehensive audit logging
- Structured logging (slog with JSON output)
- OpenTelemetry integration (Prometheus metrics, tracing)
- Request ID tracking

## Infrastructure

- PostgreSQL 16 with pgx driver
- Redis 7 for caching
- Apache Kafka (KRaft mode) for event streaming
- Kafka UI for event monitoring
- Docker Compose for local development

## HTTP Layer

- Chi router for HTTP routing
- Custom middlewares:
  - Request ID injection
  - Structured logging
  - Recovery (panic handling)
  - JWT authentication
  - RBAC authorization
  - Tenant resolution
  - Audit logging
- JSON response helpers
- Request validation (go-playground/validator)
- Pagination support

## Development Tools

- `dev.sh` - Complete development lifecycle script:
  - Infrastructure management (Docker services)
  - Database migrations
  - Hot-reload server (via air)
  - Log viewing and cleanup
- `air` configuration for hot-reload development
- Utility scripts (`scripts/`):
  - `migrate.sh` - Database migration helper
  - `test.sh` - Test runner with coverage
  - `lint.sh` - Code linting and formatting
  - `seed.go` - Database seeding script

## Testing & Quality

- Comprehensive Postman collection (37 requests)
- Postman environment configuration
- Auto-capturing scripts for tokens and IDs
- Test assertions and negative test cases
- Unit tests with coverage support

## Documentation

- Comprehensive README with:
  - Architecture overview
  - Quick start guide
  - API documentation
  - Development workflow
  - Scripts documentation
- CHANGELOG.md following Keep a Changelog format
- CONTRIBUTING.md with contribution guidelines
- CODE_OF_CONDUCT.md for community standards
- SECURITY.md with security policy
- LICENSE (AGPL v3.0)

## GitHub Templates

- Pull request template
- Bug report template
- Feature request template
- Issue template configuration

## Version

- Version: 0.0.1
- Centralized version management (`internal/shared/version`)

## Database Migrations

11 migration files covering:
- Organizations
- Users and authentication
- Organization users (multi-tenancy)
- Roles and permissions
- Applications
- API keys
- Endpoints and policies
- Audit logs

## Event System

- Generic EventBus interface
- In-process event bus (default)
- Kafka event bus (when KAFKA_ENABLED=true)
- Domain events:
  - user.registered
  - role.assigned
  - organization.created
  - app.created
  - endpoint.created
  - endpoint.retired

## Configuration

- Environment-based configuration
- Support for development, staging, production
- Docker Compose for local development
- Health check endpoints

This initial release provides a solid foundation for building scalable,
multi-tenant SaaS applications with enterprise-grade features.
