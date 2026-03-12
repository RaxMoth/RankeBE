package handler

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"

	db "ranke-be/internal/db/sqlc"
	"ranke-be/internal/middleware"
)

type UserHandler struct {
	queries *db.Queries
}

func NewUserHandler(queries *db.Queries) *UserHandler {
	return &UserHandler{queries: queries}
}

func (h *UserHandler) GetMe(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		Unauthorized(c, "authentication required")
		return
	}

	user, err := h.queries.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		NotFound(c, "user not found")
		return
	}

	Success(c, http.StatusOK, mapUser(user))
}

type updateMeRequest struct {
	DisplayName string `json:"display_name" binding:"required,min=1"`
}

func (h *UserHandler) UpdateMe(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		Unauthorized(c, "authentication required")
		return
	}

	var req updateMeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ValidationError(c, err.Error())
		return
	}

	user, err := h.queries.UpdateDisplayName(c.Request.Context(), db.UpdateDisplayNameParams{
		ID:          userID,
		DisplayName: req.DisplayName,
	})
	if err != nil {
		InternalError(c)
		return
	}

	Success(c, http.StatusOK, mapUser(user))
}

func mapUser(u db.User) gin.H {
	m := gin.H{
		"id":           formatUUID(u.ID),
		"email":        u.Email,
		"display_name": u.DisplayName,
		"auth_provider": u.AuthProvider,
		"created_at":   u.CreatedAt.Time,
	}
	if u.AvatarUrl.Valid {
		m["avatar_url"] = u.AvatarUrl.String
	}
	return m
}

func formatUUID(u pgtype.UUID) string {
	b := u.Bytes
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
