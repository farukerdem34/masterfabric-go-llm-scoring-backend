package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/masterfabric-go/masterfabric/internal/domain/audit/model"
	"github.com/masterfabric-go/masterfabric/internal/domain/audit/repository"
)

// AuditLog is middleware that records audit log entries for each request.
func AuditLog(auditRepo repository.AuditRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Serve the request first
			wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(wrapped, r)

			// Record audit log asynchronously (best-effort)
			go func() {
				// Create a detached context so audit write survives client disconnect
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()

				// Copy relevant values from request context
				reqCtx := r.Context()
				if orgID, ok := reqCtx.Value(ContextKeyOrganizationID).(uuid.UUID); ok {
					ctx = context.WithValue(ctx, ContextKeyOrganizationID, orgID)
				}
				if userID, ok := reqCtx.Value(ContextKeyUserID).(uuid.UUID); ok {
					ctx = context.WithValue(ctx, ContextKeyUserID, userID)
				}

				// Get request ID from response header
				requestID := w.Header().Get(RequestIDHeader)

				var userIDPtr *uuid.UUID
				if userID, ok := reqCtx.Value(ContextKeyUserID).(uuid.UUID); ok && userID != uuid.Nil {
					userIDPtr = &userID
				}

				var orgID uuid.UUID
				if v, ok := reqCtx.Value(ContextKeyOrganizationID).(uuid.UUID); ok {
					orgID = v
				}

				entry := &model.AuditLog{
					OrganizationID: orgID,
					UserID:         userIDPtr,
					RequestID:      requestID,
					Action:         r.Method + " " + r.URL.Path,
					ResourceType:   "http_request",
					ResourceID:     r.URL.Path,
					IPAddress:      r.RemoteAddr,
					UserAgent:      r.UserAgent(),
				}

				_ = auditRepo.Create(ctx, entry)
			}()
		})
	}
}
