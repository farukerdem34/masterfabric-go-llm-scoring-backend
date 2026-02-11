package router

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"

	// Handlers
	apimgmtHandler "github.com/masterfabric-go/masterfabric/internal/infrastructure/http/handler/apimanagement"
	auditHandler "github.com/masterfabric-go/masterfabric/internal/infrastructure/http/handler/audit"
	"github.com/masterfabric-go/masterfabric/internal/infrastructure/http/handler/health"
	iamHandler "github.com/masterfabric-go/masterfabric/internal/infrastructure/http/handler/iam"
	tenantHandler "github.com/masterfabric-go/masterfabric/internal/infrastructure/http/handler/tenant"

	// Services & middleware
	iamService "github.com/masterfabric-go/masterfabric/internal/domain/iam/service"
	"github.com/masterfabric-go/masterfabric/internal/gateway"
	"github.com/masterfabric-go/masterfabric/internal/shared/middleware"

	// Repositories (for tenant resolver middleware)
	tenantRepo "github.com/masterfabric-go/masterfabric/internal/domain/tenant/repository"
)

// Dependencies holds all injected dependencies for the router.
type Dependencies struct {
	Logger *slog.Logger
	DB     *pgxpool.Pool
	Redis  *redis.Client

	// Services
	AuthService iamService.AuthService
	RBACService iamService.RBACService

	// Handlers
	IAMHandler    *iamHandler.Handler
	TenantHandler *tenantHandler.Handler
	APIMgmtHandler *apimgmtHandler.Handler
	AuditHandler  *auditHandler.Handler

	// Gateway
	GatewayPipeline *gateway.Pipeline

	// Repos needed for middleware
	OrgRepo        tenantRepo.OrgRepository
	WorkspaceRepo  tenantRepo.WorkspaceRepository
}

// New creates the root Chi router with all middleware and routes.
func New(deps Dependencies) *chi.Mux {
	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.Logging(deps.Logger))
	r.Use(middleware.Recoverer(deps.Logger))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID", "X-Organization-ID", "X-App-ID"},
		ExposedHeaders:   []string{"X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Health endpoints
	healthHandler := health.NewHandler(deps.DB, deps.Redis)
	r.Get("/health/live", healthHandler.Liveness)
	r.Get("/health/ready", healthHandler.Readiness)

	// Prometheus metrics
	r.Handle("/metrics", promhttp.Handler())

	// API v1 routes
	r.Route("/api/v1", func(r chi.Router) {
		// Public auth routes (no JWT required)
		r.Route("/auth", func(r chi.Router) {
			if deps.IAMHandler != nil {
				r.Post("/register", deps.IAMHandler.Register)
				r.Post("/login", deps.IAMHandler.Login)
			}
		})

		// Protected routes (require JWT)
		r.Group(func(r chi.Router) {
			if deps.AuthService != nil {
				r.Use(middleware.JWTAuth(deps.AuthService))
			}

			// Tenant resolution middleware (with workspace support)
			if deps.OrgRepo != nil {
				// Note: WorkspaceRepo can be nil - workspace resolution is optional
				r.Use(middleware.TenantResolverWithWorkspace(deps.OrgRepo, deps.WorkspaceRepo))
			}

			// Gateway pipeline (rate limiting, permission enforcement for managed endpoints)
			// Must be applied before specific routes so it can handle dynamic endpoints
			if deps.GatewayPipeline != nil {
				r.Use(deps.GatewayPipeline.Enforce)
			}

			// User routes
			if deps.IAMHandler != nil {
				r.Get("/me", deps.IAMHandler.GetMe)
				r.Route("/users", func(r chi.Router) {
					r.Get("/", deps.IAMHandler.ListUsers)
					r.Get("/{id}", deps.IAMHandler.GetUser)
				})
				r.Post("/roles/assign", deps.IAMHandler.AssignRole)
			}

			// Organization routes
			if deps.TenantHandler != nil {
				r.Route("/organizations", func(r chi.Router) {
					r.Post("/", deps.TenantHandler.CreateOrg)
					r.Get("/", deps.TenantHandler.ListOrgs)
					r.Route("/{orgId}", func(r chi.Router) {
						r.Get("/", deps.TenantHandler.GetOrg)

						// Apps under organization
						r.Route("/apps", func(r chi.Router) {
							r.Post("/", deps.TenantHandler.CreateApp)
							r.Get("/", deps.TenantHandler.ListApps)
							r.Route("/{appId}", func(r chi.Router) {
								r.Get("/", deps.TenantHandler.GetApp)

								// API keys under app
								r.Route("/keys", func(r chi.Router) {
									r.Post("/", deps.TenantHandler.CreateAPIKey)
									r.Get("/", deps.TenantHandler.ListAPIKeys)
									r.Delete("/{keyId}", deps.TenantHandler.RevokeAPIKey)
								})

								// Endpoints under app
								if deps.APIMgmtHandler != nil {
									r.Route("/endpoints", func(r chi.Router) {
										r.Post("/", deps.APIMgmtHandler.DefineEndpoint)
										r.Get("/", deps.APIMgmtHandler.ListEndpoints)
										r.Route("/{endpointId}", func(r chi.Router) {
											r.Get("/", deps.APIMgmtHandler.GetEndpoint)
											r.Post("/retire", deps.APIMgmtHandler.RetireEndpoint)
											r.Post("/activate", deps.APIMgmtHandler.ActivateEndpoint)
											r.Put("/policy", deps.APIMgmtHandler.UpdatePolicy)
											r.Get("/policy", deps.APIMgmtHandler.GetPolicy)
										})
									})
								}
							})
						})

						// Workspaces under organization
						r.Route("/workspaces", func(r chi.Router) {
							r.Post("/", deps.TenantHandler.CreateWorkspace)
							r.Get("/", deps.TenantHandler.ListWorkspaces)
							r.Route("/{workspaceId}", func(r chi.Router) {
								r.Put("/", deps.TenantHandler.UpdateWorkspace)
							})
						})

						// Audit logs under organization
						if deps.AuditHandler != nil {
							r.Get("/audit-logs", deps.AuditHandler.ListByOrg)
						}
					})
				})
			}

			// Audit logs by user
			if deps.AuditHandler != nil {
				r.Get("/users/{userId}/audit-logs", deps.AuditHandler.ListByUser)
			}

			// Catch-all handler for managed endpoints (must be last in the group)
			// This allows the gateway pipeline to handle dynamic endpoints like /api/v1/products
			// The gateway middleware will validate and return responses for managed endpoints
			r.HandleFunc("/*", func(w http.ResponseWriter, r *http.Request) {
				// Gateway middleware should have already handled this if it's a managed endpoint
				// If we reach here, it means no endpoint was found, return 404
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"error":"endpoint not found","code":404,"message":"No endpoint registered for this path. Define the endpoint first using POST /api/v1/organizations/{orgId}/apps/{appId}/endpoints"}`))
			})
		})
	})

	// Catch-all handler for managed endpoints (must be after all specific routes)
	// This allows the gateway pipeline to handle dynamic endpoints like /api/v1/products
	// The gateway middleware will validate and return responses for managed endpoints
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		// If this is an API v1 path, let the gateway handle it (if it hasn't already)
		// Otherwise return 404
		if !strings.HasPrefix(r.URL.Path, "/api/v1") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":"not found","code":404}`))
			return
		}
		
		// For /api/v1 paths, check if gateway pipeline already handled it
		// If not, return 404 (gateway would have returned response if endpoint existed)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"endpoint not found","code":404,"message":"No endpoint registered for this path. Define the endpoint first."}`))
	})

	return r
}
