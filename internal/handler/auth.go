package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"ranke-be/internal/handler/dto"
	"ranke-be/internal/middleware"
	"ranke-be/internal/service"
)

type AuthHandler struct {
	auth *service.AuthService
}

func NewAuthHandler(auth *service.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

// ── request DTOs (camelCase to match the mobile client) ──────────────

type registerRequest struct {
	Email       string `json:"email"       binding:"required,email"`
	DisplayName string `json:"displayName" binding:"required,min=1"`
	Password    string `json:"password"    binding:"required,min=6"`
}

type loginRequest struct {
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type appleRequest struct {
	IdentityToken string `json:"identityToken" binding:"required"`
	FullName      string `json:"fullName"`
}

type refreshRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
}

// authResponse is the flat shape both mobile and the AuthRepository expect.
// We keep all four fields at the top level of `data` so the client can
// pluck `accessToken`/`refreshToken` and persist them in one pass.
func authResponse(svc *service.TokenPair, user dto.User) dto.AuthResponse {
	return dto.AuthResponse{
		User:         user,
		AccessToken:  svc.AccessToken,
		RefreshToken: svc.RefreshToken,
		ExpiresIn:    svc.ExpiresIn,
	}
}

// ── handlers ─────────────────────────────────────────────────────────

func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ValidationError(c, err.Error())
		return
	}

	user, pair, err := h.auth.Register(c.Request.Context(), req.Email, req.DisplayName, req.Password)
	if err != nil {
		Fail(c, http.StatusConflict, CodeValidationError, err.Error())
		return
	}

	Success(c, http.StatusCreated, authResponse(pair, dto.MapUser(*user, true)))
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ValidationError(c, err.Error())
		return
	}

	user, pair, err := h.auth.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		Unauthorized(c, "invalid credentials")
		return
	}

	Success(c, http.StatusOK, authResponse(pair, dto.MapUser(*user, true)))
}

func (h *AuthHandler) AppleSignIn(c *gin.Context) {
	var req appleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ValidationError(c, err.Error())
		return
	}

	user, pair, err := h.auth.AppleSignIn(c.Request.Context(), req.IdentityToken, req.FullName)
	if err != nil {
		Unauthorized(c, err.Error())
		return
	}

	Success(c, http.StatusOK, authResponse(pair, dto.MapUser(*user, true)))
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ValidationError(c, err.Error())
		return
	}

	pair, err := h.auth.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		Unauthorized(c, "invalid refresh token")
		return
	}

	Success(c, http.StatusOK, dto.RefreshResponse{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
		ExpiresIn:    pair.ExpiresIn,
	})
}

// Logout reads the refresh token from the body so the server can revoke it.
// We respond OK regardless of revocation success — the client clears its
// own storage after this call, so we must never block sign-out on a server
// error.
func (h *AuthHandler) Logout(c *gin.Context) {
	var req refreshRequest
	// Body is optional — a client without a stored refresh token may still
	// hit this endpoint to confirm sign-out. Don't 400 on missing fields.
	_ = c.ShouldBindJSON(&req)

	if req.RefreshToken != "" {
		_ = h.auth.Logout(c.Request.Context(), req.RefreshToken)
	}
	Success(c, http.StatusOK, gin.H{"message": "logged out"})
}

// helper for getting user ID or aborting
func requireUserID(c *gin.Context) (any, bool) {
	uid, ok := middleware.GetUserID(c)
	if !ok {
		Unauthorized(c, "authentication required")
		return nil, false
	}
	return uid, true
}
