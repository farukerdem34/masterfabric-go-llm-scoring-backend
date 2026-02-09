package gateway

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/masterfabric-go/masterfabric/internal/domain/apimanagement/repository"
	iamService "github.com/masterfabric-go/masterfabric/internal/domain/iam/service"
	"github.com/masterfabric-go/masterfabric/internal/shared/middleware"
	"github.com/masterfabric-go/masterfabric/internal/shared/response"
	"github.com/redis/go-redis/v9"
)

// Pipeline is the gateway policy pipeline that enforces policies on managed endpoints.
type Pipeline struct {
	endpointRepo repository.EndpointRepository
	policyRepo   repository.PolicyRepository
	rbac         iamService.RBACService
	redis        *redis.Client
	logger       *slog.Logger
}

// NewPipeline creates a new gateway Pipeline.
func NewPipeline(
	endpointRepo repository.EndpointRepository,
	policyRepo repository.PolicyRepository,
	rbac iamService.RBACService,
	redisClient *redis.Client,
	logger *slog.Logger,
) *Pipeline {
	return &Pipeline{
		endpointRepo: endpointRepo,
		policyRepo:   policyRepo,
		rbac:         rbac,
		redis:        redisClient,
		logger:       logger,
	}
}

// Enforce is middleware that runs the full policy pipeline for managed endpoints.
// It performs: endpoint lookup -> auth policy check -> permission enforcement -> rate limiting.
func (p *Pipeline) Enforce(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Skip for non-managed paths (health, auth, admin)
		if shouldSkipPipeline(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		// 1. Resolve app from context
		appIDStr := r.Header.Get("X-App-ID")
		if appIDStr == "" {
			next.ServeHTTP(w, r) // No app context, skip pipeline
			return
		}

		appID, err := uuid.Parse(appIDStr)
		if err != nil {
			response.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid X-App-ID"})
			return
		}

		// 2. Endpoint lookup
		endpoint, err := p.endpointRepo.GetByMethodPath(ctx, appID, r.Method, r.URL.Path, "v1")
		if err != nil {
			// No managed endpoint found, pass through
			next.ServeHTTP(w, r)
			return
		}

		if !endpoint.IsActive() {
			response.JSON(w, http.StatusGone, map[string]string{"error": "endpoint is not active"})
			return
		}

		// 3. Fetch endpoint policy
		policy, _ := p.policyRepo.GetByEndpointID(ctx, endpoint.ID)

		// 4. Permission enforcement
		if policy != nil && policy.RequiredPermission != "" {
			userID, ok := middleware.UserIDFromContext(ctx)
			if !ok {
				response.JSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
				return
			}

			orgID, _ := middleware.OrgIDFromContext(ctx)
			has, err := p.rbac.HasPermission(ctx, userID, orgID, policy.RequiredPermission)
			if err != nil {
				p.logger.Error("permission check failed", "error", err)
				response.JSON(w, http.StatusInternalServerError, map[string]string{"error": "permission check failed"})
				return
			}
			if !has {
				response.JSON(w, http.StatusForbidden, map[string]string{"error": "insufficient permissions for this endpoint"})
				return
			}
		}

		// 5. Rate limiting
		if policy != nil && policy.RateLimit > 0 && p.redis != nil {
			if err := p.checkRateLimit(r, appID, endpoint.ID, policy.RateLimit); err != nil {
				response.JSON(w, http.StatusTooManyRequests, map[string]string{"error": "rate limit exceeded"})
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

// checkRateLimit uses a Redis sliding window counter.
func (p *Pipeline) checkRateLimit(r *http.Request, appID, endpointID uuid.UUID, limit int) error {
	ctx := r.Context()
	key := fmt.Sprintf("rate:%s:%s", appID, endpointID)

	count, err := p.redis.Incr(ctx, key).Result()
	if err != nil {
		return err
	}

	if count == 1 {
		// Set TTL on first request in the window (1 minute window)
		p.redis.Expire(ctx, key, 60_000_000_000) // 60 seconds in nanoseconds
	}

	if count > int64(limit) {
		return fmt.Errorf("rate limit exceeded")
	}

	return nil
}

// shouldSkipPipeline returns true for paths that bypass the gateway policy pipeline.
func shouldSkipPipeline(path string) bool {
	skipPrefixes := []string{"/health", "/api/v1/auth", "/api/v1/admin"}
	for _, prefix := range skipPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}
