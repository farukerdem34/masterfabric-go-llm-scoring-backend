package iam

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/masterfabric-go/masterfabric/internal/application/iam/dto"
	"github.com/masterfabric-go/masterfabric/internal/application/iam/usecase"
	"github.com/masterfabric-go/masterfabric/internal/shared/config"
	"github.com/masterfabric-go/masterfabric/internal/shared/middleware"
	"github.com/masterfabric-go/masterfabric/internal/shared/pagination"
	"github.com/masterfabric-go/masterfabric/internal/shared/response"
	"github.com/masterfabric-go/masterfabric/internal/shared/validator"

	iamRepo "github.com/masterfabric-go/masterfabric/internal/domain/iam/repository"
)

// Handler provides IAM HTTP handlers.
type Handler struct {
	registerUC     *usecase.RegisterUseCase
	loginUC        *usecase.LoginUseCase
	assignRoleUC   *usecase.AssignRoleUseCase
	refreshTokenUC *usecase.RefreshTokenUseCase
	userRepo       iamRepo.UserRepository
	refreshTokenCfg config.RefreshTokenConfig
}

// NewHandler creates a new IAM handler.
func NewHandler(
	registerUC *usecase.RegisterUseCase,
	loginUC *usecase.LoginUseCase,
	assignRoleUC *usecase.AssignRoleUseCase,
	refreshTokenUC *usecase.RefreshTokenUseCase,
	userRepo iamRepo.UserRepository,
	refreshTokenCfg config.RefreshTokenConfig,
) *Handler {
	return &Handler{
		registerUC:     registerUC,
		loginUC:        loginUC,
		assignRoleUC:   assignRoleUC,
		refreshTokenUC: refreshTokenUC,
		userRepo:       userRepo,
		refreshTokenCfg: refreshTokenCfg,
	}
}

// Register handles user registration.
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req dto.RegisterRequest
	if err := validator.DecodeAndValidate(r, &req); err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	user, err := h.registerUC.Execute(r.Context(), req)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Created(w, user)
}

// Login handles user authentication.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req dto.LoginRequest
	if err := validator.DecodeAndValidate(r, &req); err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	result, err := h.loginUC.Execute(r.Context(), req)
	if err != nil {
		response.Error(w, err)
		return
	}

	// Generate refresh token
	if h.refreshTokenUC != nil {
		rawRefreshToken, _, err := h.refreshTokenUC.GenerateRefreshToken(r.Context(), result.User.ID, r)
		if err != nil {
			response.Error(w, err)
			return
		}

		// Set refresh token cookie
		http.SetCookie(w, &http.Cookie{
			Name:     h.refreshTokenCfg.CookieName,
			Value:    rawRefreshToken,
			Path:     h.refreshTokenCfg.CookiePath,
			MaxAge:   int(h.refreshTokenCfg.Duration.Seconds()),
			HttpOnly: true,
			Secure:   h.refreshTokenCfg.Secure,
			SameSite: http.SameSiteStrictMode,
		})
	}

	// Return access token
	response.JSON(w, http.StatusOK, dto.LoginResponseV2{
		AccessToken: result.Token,
		TokenType:   "Bearer",
		ExpiresIn:   h.refreshTokenCfg.AccessTokenTTL * 60,
		User:        result.User,
	})
}

// AssignRole handles role assignment.
func (h *Handler) AssignRole(w http.ResponseWriter, r *http.Request) {
	var req dto.AssignRoleRequest
	if err := validator.DecodeAndValidate(r, &req); err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	if err := h.assignRoleUC.Execute(r.Context(), req); err != nil {
		response.Error(w, err)
		return
	}

	response.NoContent(w)
}

// GetMe returns the current authenticated user.
func (h *Handler) GetMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.JSON(w, http.StatusUnauthorized, map[string]string{"error": "not authenticated"})
		return
	}

	user, err := h.userRepo.GetByID(r.Context(), userID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, dto.UserInfo{
		ID:        user.ID,
		Email:     user.Email,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Status:    string(user.Status),
		CreatedAt: user.CreatedAt,
	})
}

// GetUser returns a user by ID.
func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid user id"})
		return
	}

	user, err := h.userRepo.GetByID(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, dto.UserInfo{
		ID:        user.ID,
		Email:     user.Email,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Status:    string(user.Status),
		CreatedAt: user.CreatedAt,
	})
}

// ListUsers returns a paginated list of users.
func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	params := pagination.FromRequest(r)

	users, total, err := h.userRepo.List(r.Context(), params.Offset(), params.Limit())
	if err != nil {
		response.Error(w, err)
		return
	}

	var infos []dto.UserInfo
	for _, u := range users {
		infos = append(infos, dto.UserInfo{
			ID:        u.ID,
			Email:     u.Email,
			FirstName: u.FirstName,
			LastName:  u.LastName,
			Status:    string(u.Status),
			CreatedAt: u.CreatedAt,
		})
	}

	response.JSON(w, http.StatusOK, pagination.NewResult(infos, params, total))
}

// Refresh handles refresh token rotation.
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(h.refreshTokenCfg.CookieName)
	if err != nil {
		response.JSON(w, http.StatusUnauthorized, map[string]string{"error": "missing refresh token"})
		return
	}

	result, err := h.refreshTokenUC.Rotate(r.Context(), cookie.Value, r)
	if err != nil {
		// Clear the invalid cookie
		http.SetCookie(w, &http.Cookie{
			Name:     h.refreshTokenCfg.CookieName,
			Value:    "",
			Path:     h.refreshTokenCfg.CookiePath,
			MaxAge:   -1,
			HttpOnly: true,
		})
		response.Error(w, err)
		return
	}

	// Set new refresh token cookie
	http.SetCookie(w, &http.Cookie{
		Name:     h.refreshTokenCfg.CookieName,
		Value:    cookie.Value, // Will be replaced by new token from use case
		Path:     h.refreshTokenCfg.CookiePath,
		MaxAge:   int(h.refreshTokenCfg.Duration.Seconds()),
		HttpOnly: true,
		Secure:   h.refreshTokenCfg.Secure,
		SameSite: http.SameSiteStrictMode,
	})

	response.JSON(w, http.StatusOK, result)
}

// Logout revokes the current refresh token.
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(h.refreshTokenCfg.CookieName)
	if err != nil {
		// No cookie, already logged out
		response.NoContent(w)
		return
	}

	_ = h.refreshTokenUC.Logout(r.Context(), cookie.Value)

	// Clear cookie
	http.SetCookie(w, &http.Cookie{
		Name:     h.refreshTokenCfg.CookieName,
		Value:    "",
		Path:     h.refreshTokenCfg.CookiePath,
		MaxAge:   -1,
		HttpOnly: true,
	})

	response.NoContent(w)
}

// LogoutAll revokes all refresh tokens for the authenticated user.
func (h *Handler) LogoutAll(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.JSON(w, http.StatusUnauthorized, map[string]string{"error": "not authenticated"})
		return
	}

	if err := h.refreshTokenUC.LogoutAll(r.Context(), userID); err != nil {
		response.Error(w, err)
		return
	}

	// Clear cookie
	http.SetCookie(w, &http.Cookie{
		Name:     h.refreshTokenCfg.CookieName,
		Value:    "",
		Path:     h.refreshTokenCfg.CookiePath,
		MaxAge:   -1,
		HttpOnly: true,
	})

	response.NoContent(w)
}

// Sessions returns active sessions for the authenticated user.
func (h *Handler) Sessions(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.JSON(w, http.StatusUnauthorized, map[string]string{"error": "not authenticated"})
		return
	}

	sessions, err := h.refreshTokenUC.ListSessions(r.Context(), userID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, sessions)
}
