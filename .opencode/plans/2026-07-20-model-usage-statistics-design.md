# Model Usage Statistics — Design Spec

**Date:** 2026-07-20
**Status:** Approved
**Scope:** Backend API endpoints + database schema + frontend statistics tab

---

## 1. Overview

The LLM Playground frontend runs models in-browser via WebLLM and already tracks per-generation metrics (token count, inference time, tokens/sec, first-token latency). This feature adds:

1. A backend endpoint to **record** these metrics after each generation
2. A backend endpoint to **query** a user's own usage statistics with date range filtering
3. A frontend **Statistics tab** showing per-model breakdowns and aggregate stats

**Key constraint:** Each user can only see their own statistics. No cross-user or cross-organization visibility.

---

## 2. Data Model

### Migration: `00014_create_model_usage_stats.sql`

```sql
-- +goose Up
CREATE TABLE model_usage_stats (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    model_id VARCHAR(255) NOT NULL,
    token_count INTEGER NOT NULL DEFAULT 0,
    first_token_time_ms INTEGER,
    inference_time_ms INTEGER NOT NULL DEFAULT 0,
    tokens_per_second NUMERIC(10,2),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_model_usage_stats_user_id ON model_usage_stats(user_id);
CREATE INDEX idx_model_usage_stats_user_created ON model_usage_stats(user_id, created_at);
CREATE INDEX idx_model_usage_stats_user_model ON model_usage_stats(user_id, model_id);
CREATE INDEX idx_model_usage_stats_model_id ON model_usage_stats(model_id);

-- +goose Down
DROP TABLE IF EXISTS model_usage_stats;
```

### Entity

```go
// internal/domain/statistics/model/usage_stat.go
type UsageStat struct {
    ID               uuid.UUID
    UserID           uuid.UUID
    ModelID          string
    TokenCount       int
    FirstTokenTimeMs *int       // nil if first token never captured
    InferenceTimeMs  int
    TokensPerSecond  *float64   // nil if zero tokens
    CreatedAt        time.Time
}
```

---

## 3. API Endpoints

### POST /api/v1/statistics/usage

Records a single generation's metrics. User ID extracted from JWT context.

**Request:**
```json
{
  "model_id": "TinyLlama-1.1B-Chat-v1.0-q4f16_1-MLC",
  "token_count": 150,
  "first_token_time_ms": 120,
  "inference_time_ms": 3400,
  "tokens_per_second": 44.12
}
```

**Response 201:**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "user_id": "...",
  "model_id": "TinyLlama-1.1B-Chat-v1.0-q4f16_1-MLC",
  "token_count": 150,
  "first_token_time_ms": 120,
  "inference_time_ms": 3400,
  "tokens_per_second": 44.12,
  "created_at": "2026-07-20T12:00:00Z"
}
```

**Validation:**
- `model_id` — required, non-empty string
- `token_count` — required, >= 0
- `inference_time_ms` — required, > 0
- `first_token_time_ms` — optional, >= 0
- `tokens_per_second` — optional, >= 0

### GET /api/v1/statistics/usage

Queries the authenticated user's own usage stats.

**Query Parameters:**
| Param | Type | Required | Default | Description |
|---|---|---|---|---|
| `model_id` | string | no | — | Filter by model |
| `start_date` | ISO 8601 | no | — | Start of range (inclusive) |
| `end_date` | ISO 8601 | no | — | End of range (inclusive) |
| `page` | int | no | 1 | Page number |
| `per_page` | int | no | 50 | Results per page (max 100) |

**Response 200:**
```json
{
  "data": [
    {
      "id": "...",
      "user_id": "...",
      "model_id": "TinyLlama-1.1B-Chat-v1.0-q4f16_1-MLC",
      "token_count": 150,
      "first_token_time_ms": 120,
      "inference_time_ms": 3400,
      "tokens_per_second": 44.12,
      "created_at": "2026-07-20T12:00:00Z"
    }
  ],
  "total": 342,
  "page": 1,
  "per_page": 50,
  "summary": {
    "total_generations": 342,
    "total_tokens": 51200,
    "avg_tokens_per_second": 42.5,
    "models_used": ["TinyLlama-1.1B-Chat-v1.0-q4f16_1-MLC", "Llama-3.2-1B-Instruct-q4f16_1-MLC"],
    "model_breakdown": [
      { "model_id": "TinyLlama-1.1B-Chat-v1.0-q4f16_1-MLC", "count": 200, "avg_tokens_per_second": 45.2 },
      { "model_id": "Llama-3.2-1B-Instruct-q4f16_1-MLC", "count": 142, "avg_tokens_per_second": 39.1 }
    ]
  }
}
```

---

## 4. Domain Layer

### Repository Interface

```go
// internal/domain/statistics/repository/usage_stat_repository.go
type UsageStatRepository interface {
    Create(ctx context.Context, stat *model.UsageStat) error
    ListByUser(ctx context.Context, userID uuid.UUID, filter UsageStatFilter) ([]model.UsageStat, int, error)
    GetSummaryByUser(ctx context.Context, userID uuid.UUID, filter UsageStatFilter) (*UsageSummary, error)
}

type UsageStatFilter struct {
    ModelID   *string
    StartDate *time.Time
    EndDate   *time.Time
    Page      int
    PerPage   int
}

type UsageSummary struct {
    TotalGenerations int
    TotalTokens      int64
    AvgTokensPerSec  float64
    ModelsUsed       []string
    ModelBreakdown   []ModelBreakdown
}

type ModelBreakdown struct {
    ModelID         string
    Count           int
    AvgTokensPerSec float64
}
```

### Use Cases

- **RecordUsageUseCase** — Validates input, creates `UsageStat` entity, calls `repository.Create()`
- **GetUserStatsUseCase** — Calls `repository.ListByUser()` and `repository.GetSummaryByUser()`, returns combined response

### DTOs

```go
// internal/application/statistics/dto/usage_stat_dto.go
type RecordUsageRequest struct {
    ModelID          string   `json:"model_id" validate:"required"`
    TokenCount       int      `json:"token_count" validate:"required,gte=0"`
    FirstTokenTimeMs *int     `json:"first_token_time_ms,omitempty" validate:"omitempty,gte=0"`
    InferenceTimeMs  int      `json:"inference_time_ms" validate:"required,gt=0"`
    TokensPerSecond  *float64 `json:"tokens_per_second,omitempty" validate:"omitempty,gte=0"`
}

type UsageStatResponse struct { ... }  // full record
type UsageStatsResponse struct { ... } // paginated data + summary
type UsageStatsQuery struct { ... }    // query params
```

---

## 5. Infrastructure Layer

### PostgreSQL Repository

`internal/infrastructure/postgres/statistics/usage_stat_repository.go`:
- `Create`: `INSERT INTO model_usage_stats (...) VALUES (...)` with parameterized args
- `ListByUser`: Dynamic WHERE (model_id, date range), `ORDER BY created_at DESC`, `LIMIT/OFFSET`
- `GetSummaryByUser`: SQL aggregates (`COUNT`, `SUM`, `AVG`) with `GROUP BY model_id`

### HTTP Handler

`internal/infrastructure/http/handler/statistics/usage_handler.go`:
- `RecordUsage`: Parse body → call use case → return 201
- `GetUserStats`: Parse query params → call use case → return 200

### Router

Add to protected route group in `router.go`:
```go
r.Route("/statistics", func(r chi.Router) {
    r.Post("/usage", statsHandler.RecordUsage)
    r.Get("/usage", statsHandler.GetUserStats)
})
```

### Dependency Injection

Add to `buildDependencies()` in `cmd/server/main.go`:
1. `statsRepo := postgresStats.NewUsageStatRepo(db)`
2. `recordUsageUC := statsUC.NewRecordUsageUseCase(statsRepo)`
3. `getUserStatsUC := statsUC.NewGetUserStatsUseCase(statsRepo)`
4. `statsHandler := statsHandler.NewUsageHandler(recordUsageUC, getUserStatsUC)`
5. Pass `statsHandler` to `router.New(deps)`

---

## 6. Frontend Integration

### Stats Submission

After generation completes in `useWebLLM.ts`, iterate ready models and POST metrics:
```typescript
for (const modelId of readyModels) {
  const result = results[modelId];
  if (result.tokenCount && result.inferenceTime) {
    await fetch(`${API_BASE}/api/v1/statistics/usage`, {
      method: "POST",
      headers: { "Authorization": `Bearer ${token}` },
      body: JSON.stringify({
        model_id: modelId,
        token_count: result.tokenCount,
        first_token_time_ms: result.firstTokenTime,
        inference_time_ms: result.inferenceTime,
        tokens_per_second: result.tokensPerSecond,
      }),
    });
  }
}
```

### Statistics Tab

New route/page with:
- Date range picker (start/end date inputs)
- Summary card (total generations, total tokens, avg tokens/sec)
- Per-model breakdown table (model name, count, avg speed)
- Paginated raw records list

### New Frontend Files

- `app/components/StatsTab.tsx` — main statistics view
- `app/components/ModelBreakdownCard.tsx` — per-model stats table
- `app/components/StatsSummary.tsx` — aggregate summary card
- `app/hooks/useStats.ts` — data fetching hook

---

## 7. New Files Summary

### Backend (8 new + 2 modify)
1. `internal/infrastructure/postgres/migrations/00014_create_model_usage_stats.sql`
2. `internal/domain/statistics/model/usage_stat.go`
3. `internal/domain/statistics/repository/usage_stat_repository.go`
4. `internal/application/statistics/dto/usage_stat_dto.go`
5. `internal/application/statistics/usecase/record_usage.go`
6. `internal/application/statistics/usecase/get_user_stats.go`
7. `internal/infrastructure/postgres/statistics/usage_stat_repository.go`
8. `internal/infrastructure/http/handler/statistics/usage_handler.go`
9. `internal/infrastructure/http/router/router.go` (modify)
10. `cmd/server/main.go` (modify)

### Frontend (4 new + 1 modify)
11. `app/hooks/useStats.ts`
12. `app/components/StatsTab.tsx`
13. `app/components/ModelBreakdownCard.tsx`
14. `app/components/StatsSummary.tsx`
15. `app/page.tsx` (modify — add stats tab)
