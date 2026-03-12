package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"ranke-be/internal/middleware"
	"ranke-be/internal/service"
)

type AuthHandler struct {
	auth *service.AuthService
}

func NewAuthHandler(auth *service.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

type registerRequest struct {
	Email       string `json:"email" binding:"required,email"`
	DisplayName string `json:"display_name" binding:"required,min=1"`
	Password    string `json:"password" binding:"required,min=6"`
}

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

	Success(c, http.StatusCreated, gin.H{
		"user":  userResponse(user.ID, user.Email, user.DisplayName, user.AvatarUrl.String, user.CreatedAt.Time),
		"tokens": pair,
	})
}

type loginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
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

	Success(c, http.StatusOK, gin.H{
		"user":  userResponse(user.ID, user.Email, user.DisplayName, user.AvatarUrl.String, user.CreatedAt.Time),
		"tokens": pair,
	})
}

type appleRequest struct {
	IdentityToken string `json:"identity_token" binding:"required"`
}

func (h *AuthHandler) AppleSignIn(c *gin.Context) {
	var req appleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ValidationError(c, err.Error())
		return
	}

	user, pair, err := h.auth.AppleSignIn(c.Request.Context(), req.IdentityToken)
	if err != nil {
		Unauthorized(c, err.Error())
		return
	}

	Success(c, http.StatusOK, gin.H{
		"user":  userResponse(user.ID, user.Email, user.DisplayName, user.AvatarUrl.String, user.CreatedAt.Time),
		"tokens": pair,
	})
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
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

	Success(c, http.StatusOK, pair)
}

func (h *AuthHandler) Logout(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ValidationError(c, err.Error())
		return
	}

	_ = h.auth.Logout(c.Request.Context(), req.RefreshToken)
	Success(c, http.StatusOK, gin.H{"message": "logged out"})
}

// userResponse is a helper shared across auth handlers.
func userResponse(id any, email, displayName, avatarURL string, createdAt any) gin.H {
	resp := gin.H{
		"id":           id,
		"email":        email,
		"display_name": displayName,
		"created_at":   createdAt,
	}
	if avatarURL != "" {
		resp["avatar_url"] = avatarURL
	}
	return resp
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
