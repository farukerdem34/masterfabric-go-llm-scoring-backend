# Plan: Fix CORS + Cross-Origin Cookie for Render Deployment

## Problem

Two issues prevent cross-origin cookie-based auth from working:

1. **CORS**: Server returns `Access-Control-Allow-Origin: *` → browser rejects when `credentials: 'include'` is used (CORS spec: wildcard forbidden with credentials)
2. **Cookie SameSite**: `SameSite=Strict` → browser never sends cookies on cross-origin fetch/XHR requests

## Root Cause Analysis

### CORS
- `CORS_ALLOWED_ORIGINS` env defaults to `nil` (empty) on Render
- `CORSOptions([])` sets `AllowedOrigins: []`, `AllowCredentials: false`
- `go-chi/cors` with empty origins still sends `Access-Control-Allow-Origin: *`
- Frontend uses `credentials: 'include'` (required for cookies) → browser rejects `*`

### Cookie SameSite
- All refresh token cookies use `SameSite: http.SameSiteStrictMode` (handler.go:95, 228)
- Cross-origin requests (localhost:3000 → Render) → `Strict` blocks cookie entirely
- Only `SameSite=None` works cross-origin (requires `Secure=true`)

## Fix 1: `internal/shared/middleware/cors.go`

Replace entire file. Use `AllowOriginFunc` to reflect the exact requesting origin:

- `*` in `CORS_ALLOWED_ORIGINS` → accept any origin, reflect it back
- Specific origins → match against set, reflect matching origin
- Empty → deny all cross-origin
- `AllowCredentials` always `true`

Key change: never return `*` in `Access-Control-Allow-Origin` header — always reflect the exact origin.

## Fix 2: `internal/infrastructure/http/handler/iam/handler.go`

Change all 4 occurrences of `SameSite: http.SameSiteStrictMode`:

```go
sameSite := http.SameSiteStrictMode
if h.refreshTokenCfg.Secure {
    sameSite = http.SameSiteNoneMode
}
```

- `Secure=true` (production) → `SameSite=None` (cross-origin cookie support)
- `Secure=false` (development) → `SameSite=Strict` (same-origin, more secure)

Affected locations:
- Line 95: Login handler cookie set
- Line 228: Refresh handler cookie set
- (Logout and LogoutAll clear cookies with MaxAge=-1, no SameSite needed)

## Fix 3: `render.yaml`

Add `CORS_ALLOWED_ORIGINS` env var to web service envVars.

## Verification

1. `go test -short -race ./...`
2. `go vet ./...`
3. `go build ./cmd/server`
