package handlers

import (
	"net/http"

	"gin-rest-template/internal/config"
	"gin-rest-template/internal/middleware"
	"gin-rest-template/internal/models"
	"gin-rest-template/internal/service"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *service.Service
	config  *config.Config
}

func NewHandler(service *service.Service, config *config.Config) *Handler {
	return &Handler{
		service: service,
		config:  config,
	}
}

// HealthCheck godoc
// @Summary Health check
// @Description Check if the API is running
// @Tags health
// @Accept json
// @Produce json
// @Success 200 {object} models.SuccessResponse
// @Router /health [get]
func (h *Handler) HealthCheck(c *gin.Context) {
	if err := h.service.HealthCheck(c.Request.Context()); err != nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponse{
			Error:   "service_unavailable",
			Message: "Database connection failed",
		})
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse{
		Message: "Service is healthy",
		Data: gin.H{
			"status":  "ok",
			"version": "1.0.0",
		},
	})
}

// Register godoc
// @Summary Register a new user
// @Description Register a new user with email and password
// @Tags auth
// @Accept json
// @Produce json
// @Param request body models.RegisterRequest true "Registration details"
// @Success 201 {object} models.SuccessResponse{data=models.UserResponse}
// @Failure 400 {object} models.ErrorResponse
// @Failure 409 {object} models.ErrorResponse
// @Router /api/v1/auth/register [post]
func (h *Handler) Register(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	user, err := h.service.Register(c.Request.Context(), &req)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "user with this email already exists" {
			statusCode = http.StatusConflict
		}
		c.JSON(statusCode, models.ErrorResponse{
			Error:   "registration_failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, models.SuccessResponse{
		Message: "User registered successfully",
		Data:    user,
	})
}

// Login godoc
// @Summary Login user
// @Description Authenticate user and return JWT tokens
// @Tags auth
// @Accept json
// @Produce json
// @Param request body models.LoginRequest true "Login credentials"
// @Success 200 {object} models.TokenResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Router /api/v1/auth/login [post]
func (h *Handler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	tokens, err := h.service.Login(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "authentication_failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, tokens)
}

// RefreshToken godoc
// @Summary Refresh access token
// @Description Get a new access token using refresh token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body models.RefreshTokenRequest true "Refresh token"
// @Success 200 {object} models.TokenResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Router /api/v1/auth/refresh [post]
func (h *Handler) RefreshToken(c *gin.Context) {
	var req models.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	tokens, err := h.service.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "token_refresh_failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, tokens)
}

// GetCurrentUser godoc
// @Summary Get current user
// @Description Get the currently authenticated user's profile
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} models.SuccessResponse{data=models.UserResponse}
// @Failure 401 {object} models.ErrorResponse
// @Router /api/v1/users/me [get]
func (h *Handler) GetCurrentUser(c *gin.Context) {
	userID := middleware.GetUserID(c)

	user, err := h.service.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error:   "user_not_found",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse{
		Message: "User retrieved successfully",
		Data:    user,
	})
}

// UpdateCurrentUser godoc
// @Summary Update current user
// @Description Update the currently authenticated user's profile
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.UpdateUserRequest true "Update details"
// @Success 200 {object} models.SuccessResponse{data=models.UserResponse}
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Router /api/v1/users/me [put]
func (h *Handler) UpdateCurrentUser(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req models.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	user, err := h.service.UpdateUser(c.Request.Context(), userID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "update_failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse{
		Message: "User updated successfully",
		Data:    user,
	})
}

// CreateAlbum godoc
// @Summary Create a new album
// @Description Create a new album for the authenticated user
// @Tags albums
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.Album true "Album details"
// @Success 201 {object} models.SuccessResponse{data=models.Album}
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Router /api/v1/albums [post]
func (h *Handler) CreateAlbum(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var album models.Album
	if err := c.ShouldBindJSON(&album); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	createdAlbum, err := h.service.CreateAlbum(c.Request.Context(), userID, &album)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "creation_failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, models.SuccessResponse{
		Message: "Album created successfully",
		Data:    createdAlbum,
	})
}

// GetAlbums godoc
// @Summary Get all albums
// @Description Get all albums for the authenticated user with pagination
// @Tags albums
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(10)
// @Param sort_by query string false "Sort by field" default(created_at)
// @Param order query string false "Sort order (asc/desc)" default(desc)
// @Success 200 {object} models.PaginatedResponse
// @Failure 401 {object} models.ErrorResponse
// @Router /api/v1/albums [get]
func (h *Handler) GetAlbums(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var pagination models.PaginationQuery
	if err := c.ShouldBindQuery(&pagination); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	// Set defaults
	if pagination.Page == 0 {
		pagination.Page = 1
	}
	if pagination.PageSize == 0 {
		pagination.PageSize = 10
	}
	if pagination.SortBy == "" {
		pagination.SortBy = "created_at"
	}
	if pagination.Order == "" {
		pagination.Order = "desc"
	}

	result, err := h.service.GetAlbums(c.Request.Context(), userID, &pagination)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "retrieval_failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetAlbumByID godoc
// @Summary Get album by ID
// @Description Get a specific album by ID
// @Tags albums
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Album ID"
// @Success 200 {object} models.SuccessResponse{data=models.Album}
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Router /api/v1/albums/{id} [get]
func (h *Handler) GetAlbumByID(c *gin.Context) {
	userID := middleware.GetUserID(c)
	albumID := c.Param("id")

	album, err := h.service.GetAlbumByID(c.Request.Context(), albumID, userID)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "album not found" || err.Error() == "unauthorized access to album" {
			statusCode = http.StatusNotFound
		}
		c.JSON(statusCode, models.ErrorResponse{
			Error:   "retrieval_failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse{
		Message: "Album retrieved successfully",
		Data:    album,
	})
}

// UpdateAlbum godoc
// @Summary Update album
// @Description Update an existing album
// @Tags albums
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Album ID"
// @Param request body models.Album true "Updated album details"
// @Success 200 {object} models.SuccessResponse{data=models.Album}
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Router /api/v1/albums/{id} [put]
func (h *Handler) UpdateAlbum(c *gin.Context) {
	userID := middleware.GetUserID(c)
	albumID := c.Param("id")

	var updates models.Album
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	album, err := h.service.UpdateAlbum(c.Request.Context(), albumID, userID, &updates)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "album not found" || err.Error() == "unauthorized access to album" {
			statusCode = http.StatusNotFound
		}
		c.JSON(statusCode, models.ErrorResponse{
			Error:   "update_failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse{
		Message: "Album updated successfully",
		Data:    album,
	})
}

// DeleteAlbum godoc
// @Summary Delete album
// @Description Delete an existing album
// @Tags albums
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Album ID"
// @Success 200 {object} models.SuccessResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Router /api/v1/albums/{id} [delete]
func (h *Handler) DeleteAlbum(c *gin.Context) {
	userID := middleware.GetUserID(c)
	albumID := c.Param("id")

	err := h.service.DeleteAlbum(c.Request.Context(), albumID, userID)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "album not found" || err.Error() == "unauthorized access to album" {
			statusCode = http.StatusNotFound
		}
		c.JSON(statusCode, models.ErrorResponse{
			Error:   "deletion_failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse{
		Message: "Album deleted successfully",
	})
}
