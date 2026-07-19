# Refresh Token Flow — Design Spec

## Overview

Add a refresh token flow to the existing IAM auth system. Short-lived access tokens (15min JWT) paired with long-lived refresh tokens (7d opaque) stored server-side in PostgreSQL. Refresh tokens delivered via HttpOnly Secure cookies. Deployed to Render with managed PostgreSQL.

## Database Schema

**New migration:** `00013_create_refresh_tokens.sql`

```sql
CREATE TABLE refresh_tokens (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash    VARCHAR(64) NOT NULL UNIQUE,
    family_id     UUID NOT NULL,
    expires_at    TIMESTAMPTZ NOT NULL,
    revoked_at    TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ip_address    INET,
    user_agent    TEXT
);

CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_token_hash ON refresh_tokens(token_hash);
CREATE INDEX idx_refresh_tokens_family_id ON refresh_tokens(family_id);
```

- `token_hash`: SHA-256 of the opaque token (never store plaintext)
- `family_id`: links rotation chains for replay detection
- `revoked_at`: NULL = active, non-NULL = revoked
- `ip_address` / `user_agent`: audit trail

## Token Design

| Property | Access Token | Refresh Token |
|----------|-------------|---------------|
| Format | JWT | Opaque (32 bytes, base64url) |
| Lifetime | 15 minutes | 7 days |
| Storage (client) | Memory / variable | HttpOnly cookie |
| Storage (server) | Stateless validation | SHA-256 hash in `refresh_tokens` table |
| Cookie settings | N/A | `Secure; SameSite=Strict; Path=/auth/refresh; Max-Age=604800` |

## API Endpoints

### Modified

| Method | Path | Auth | Behavior |
|--------|------|------|----------|
| `POST` | `/api/v1/auth/login` | Public | Returns access token + sets refresh token cookie |
| `POST` | `/api/v1/auth/register` | Public | Same as before + creates initial refresh token |

### New

| Method | Path | Auth | Behavior |
|--------|------|------|----------|
| `POST` | `/api/v1/auth/refresh` | Cookie | Rotates refresh token, returns new access token |
| `POST` | `/api/v1/auth/logout` | Cookie | Revokes current refresh token, clears cookie |
| `POST` | `/api/v1/auth/logout-all` | JWT | Revokes ALL refresh tokens for the user |
| `GET` | `/api/v1/auth/sessions` | JWT | Lists active sessions (devices) |

### Response Shapes

```json
// POST /auth/login  →  200
{
  "access_token": "eyJhbG...",
  "token_type": "Bearer",
  "expires_in": 900,
  "user": { "id": "...", "email": "...", "first_name": "...", "last_name": "..." }
}

// POST /auth/refresh  →  200
{
  "access_token": "eyJhbG...",
  "token_type": "Bearer",
  "expires_in": 900
}

// POST /auth/refresh  →  401
{ "error": "invalid or expired refresh token" }

// GET /auth/sessions  →  200
{
  "sessions": [
    { "id": "...", "created_at": "...", "ip_address": "...", "user_agent": "...", "is_current": false }
  ]
}
```

## Token Flow

```
LOGIN:
  1. Validate email/password
  2. Generate access JWT (15min)
  3. Generate opaque refresh token (32 bytes)
  4. Store SHA-256(refresh_token) + family_id in DB
  5. Set-Cookie: refresh_token=...; HttpOnly; Secure; SameSite=Strict
  6. Return { access_token, user } in JSON

REFRESH:
  1. Read refresh_token from cookie
  2. Hash it, look up in DB
  3. If not found OR revoked OR expired → 401
  4. If family_id has a revoked sibling → replay detected, revoke entire family → 401
  5. Revoke current token (set revoked_at)
  6. Generate new access JWT (15min)
  7. Generate new refresh token (7d), SAME family_id
  8. Store new token_hash in DB
  9. Set-Cookie: new refresh_token
  10. Return { access_token } in JSON

LOGOUT:
  1. Read refresh_token from cookie
  2. Revoke that token in DB (set revoked_at)
  3. Clear cookie
  4. Return 204

LOGOUT-ALL:
  1. From JWT context: revoke ALL tokens for this user_id
  2. Clear cookie
  3. Return 204
```

## Implementation Structure (Clean Architecture)

### Domain Layer

```
internal/domain/iam/model/refresh_token.go
  → RefreshToken struct

internal/domain/iam/repository/refresh_token_repository.go
  → RefreshTokenRepository interface
    - Create(ctx, token) error
    - GetByTokenHash(ctx, hash) (*RefreshToken, error)
    - Revoke(ctx, id) error
    - RevokeAllByUserID(ctx, userID) error
    - RevokeFamily(ctx, familyID) error
    - ListActiveByUserID(ctx, userID) ([]RefreshToken, error)
    - DeleteExpired(ctx) error
```

### Application Layer

```
internal/application/iam/usecase/refresh_token_usecase.go
  → RefreshTokenUseCase
    - Login (generates both tokens, sets cookie)
    - Refresh (validates, rotates, returns new access token)
    - Logout (revokes single token)
    - LogoutAll (revokes all user tokens)
    - ListSessions (returns active tokens)
```

### Infrastructure Layer

```
internal/infrastructure/postgres/iam/refresh_token_repo.go
  → PostgresRefreshTokenRepository (implements RefreshTokenRepository)

internal/infrastructure/http/handler/iam/handler.go
  → Add methods: Refresh, Logout, LogoutAll, Sessions
```

### Shared Layer (modified)

```
internal/shared/config/config.go
  → Add RefreshTokenConfig struct (Duration, CookieName, CookieDomain, Secure)
  → Add DATABASE_URL parsing support
  → Add PORT env var fallback
```

### Router Changes

```go
// Public group:
r.Post("/auth/refresh", iamHandler.Refresh)

// JWT-protected group:
r.Post("/auth/logout", iamHandler.Logout)
r.Post("/auth/logout-all", iamHandler.LogoutAll)
r.Get("/auth/sessions", iamHandler.Sessions)
```

## Render Deployment

### `render.yaml`

```yaml
services:
  - type: web
    name: masterfabric-go
    runtime: go
    buildCommand: go build -o server ./cmd/server
    startCommand: ./server
    envVars:
      - key: SERVER_HOST
        value: "0.0.0.0"
      - key: SERVER_PORT
        value: "10000"
      - key: DATABASE_URL
        fromDatabase:
          name: masterfabric-db
          property: connectionString
      - key: JWT_SECRET
        generateValue: true
      - key: JWT_ISSUER
        value: "masterfabric"
      - key: LOG_LEVEL
        value: "info"
      - key: LOG_FORMAT
        value: "json"
    healthCheckPath: /health/live

databases:
  - name: masterfabric-db
    plan: free
    databaseName: masterfabric
```

### Config Adjustments

1. `DATABASE_URL` support — Render provides a single connection string; `Load()` parses it instead of individual `DB_HOST`/`DB_PORT`/etc.
2. `PORT` fallback — Render sets `PORT`; config checks `PORT` → `SERVER_PORT` → `8080`.

## Files to Create/Modify

| Action | File |
|--------|------|
| **Create** | `internal/domain/iam/model/refresh_token.go` |
| **Create** | `internal/domain/iam/repository/refresh_token_repository.go` |
| **Create** | `internal/application/iam/usecase/refresh_token_usecase.go` |
| **Create** | `internal/infrastructure/postgres/iam/refresh_token_repo.go` |
| **Create** | `internal/infrastructure/postgres/migrations/00013_create_refresh_tokens.sql` |
| **Create** | `render.yaml` |
| **Modify** | `internal/infrastructure/http/handler/iam/handler.go` — add Refresh, Logout, LogoutAll, Sessions |
| **Modify** | `internal/infrastructure/http/router/router.go` — add new routes |
| **Modify** | `internal/shared/config/config.go` — add DATABASE_URL parsing, PORT fallback, RefreshTokenConfig |
| **Modify** | `cmd/server/main.go` — wire RefreshTokenRepository, RefreshTokenUseCase |
