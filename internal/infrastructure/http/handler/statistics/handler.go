package statistics

import (
	"net/http"

	"github.com/masterfabric-go/masterfabric/internal/application/statistics/dto"
	"github.com/masterfabric-go/masterfabric/internal/application/statistics/usecase"
	"github.com/masterfabric-go/masterfabric/internal/shared/middleware"
	"github.com/masterfabric-go/masterfabric/internal/shared/pagination"
	"github.com/masterfabric-go/masterfabric/internal/shared/response"
	"github.com/masterfabric-go/masterfabric/internal/shared/validator"
)

// Handler provides Model Usage Statistics HTTP handlers.
type Handler struct {
	recordUsageUC  *usecase.RecordUsageUseCase
	getUserStatsUC *usecase.GetUserStatsUseCase
}

// NewHandler creates a new Statistics handler.
func NewHandler(recordUsageUC *usecase.RecordUsageUseCase, getUserStatsUC *usecase.GetUserStatsUseCase) *Handler {
	return &Handler{
		recordUsageUC:  recordUsageUC,
		getUserStatsUC: getUserStatsUC,
	}
}

// RecordUsage records a single generation's usage statistics.
func (h *Handler) RecordUsage(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.JSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var req dto.RecordUsageRequest
	if err := validator.DecodeAndValidate(r, &req); err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	stat, err := h.recordUsageUC.Execute(r.Context(), userID, &req)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Created(w, stat)
}

// GetUserStats returns the authenticated user's own usage statistics.
func (h *Handler) GetUserStats(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.JSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	params := pagination.FromRequest(r)

	query := &dto.UsageStatsQuery{
		Page:    params.Page,
		PerPage: params.PerPage,
	}

	if v := r.URL.Query().Get("model_id"); v != "" {
		query.ModelID = &v
	}
	if v := r.URL.Query().Get("start_date"); v != "" {
		query.StartDate = &v
	}
	if v := r.URL.Query().Get("end_date"); v != "" {
		query.EndDate = &v
	}

	result, err := h.getUserStatsUC.Execute(r.Context(), userID, query)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, result)
}
