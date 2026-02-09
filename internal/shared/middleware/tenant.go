package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/masterfabric-go/masterfabric/internal/domain/tenant/repository"
	"github.com/masterfabric-go/masterfabric/internal/shared/logger"
	"github.com/masterfabric-go/masterfabric/internal/shared/response"
)

const (
	ContextKeyTenantID contextKey = "tenant_id"
	ContextKeyAppID    contextKey = "tenant_app_id"
)

// TenantResolver resolves the tenant (organization) from the request.
// Resolution order: X-Organization-ID header > JWT claims > subdomain.
func TenantResolver(orgRepo repository.OrgRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			var orgID uuid.UUID

			// 1. Check explicit header
			if header := r.Header.Get("X-Organization-ID"); header != "" {
				parsed, err := uuid.Parse(header)
				if err != nil {
					response.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid X-Organization-ID"})
					return
				}
				orgID = parsed
			}

			// 2. Fall back to JWT claims (if auth middleware already ran)
			if orgID == uuid.Nil {
				if claimOrgID, ok := ctx.Value(ContextKeyOrganizationID).(uuid.UUID); ok && claimOrgID != uuid.Nil {
					orgID = claimOrgID
				}
			}

			// 3. Fall back to subdomain
			if orgID == uuid.Nil {
				host := r.Host
				parts := strings.Split(host, ".")
				if len(parts) > 2 {
					slug := parts[0]
					if orgRepo != nil {
						org, err := orgRepo.GetBySlug(ctx, slug)
						if err == nil && org != nil {
							orgID = org.ID
						}
					}
				}
			}

			if orgID != uuid.Nil {
				ctx = context.WithValue(ctx, ContextKeyTenantID, orgID)
				ctx = logger.ContextWithOrganizationID(ctx, orgID.String())
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireTenant ensures a tenant ID is present in the context.
func RequireTenant(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := TenantIDFromContext(r.Context()); !ok {
			response.JSON(w, http.StatusBadRequest, map[string]string{"error": "tenant (organization) not resolved"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

// TenantIDFromContext extracts the tenant ID from context.
func TenantIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(ContextKeyTenantID).(uuid.UUID)
	return id, ok && id != uuid.Nil
}
